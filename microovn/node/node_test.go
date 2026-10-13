package node

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/canonical/lxd/lxd/db/query"
	"github.com/canonical/microcluster/v3/microcluster/db"
	_ "github.com/mattn/go-sqlite3"

	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/database"
)

// Use the real service schema and query with SQLite and LXD's ten-second
// transaction helper, while modelling retries without a running dqlite cluster.
type serviceDatabase struct {
	db            *sql.DB
	attempts      int
	inTransaction bool
	committed     bool
	callbackCtx   context.Context
	beginError    error
	commitError   error
}

func newServiceDatabase(t *testing.T) *serviceDatabase {
	t.Helper()
	sqlDB, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	sqlDB.SetMaxOpenConns(1)
	_, err = sqlDB.Exec(`
CREATE TABLE core_cluster_members (id INTEGER PRIMARY KEY, name TEXT);
INSERT INTO core_cluster_members VALUES (1, 'local'), (2, 'remote');
`)
	if err != nil {
		t.Fatal(err)
	}
	for _, update := range database.SchemaExtensions {
		if err := query.Transaction(context.Background(), sqlDB, update); err != nil {
			t.Fatal(err)
		}
	}
	_, err = sqlDB.Exec(`
INSERT INTO services VALUES (1, 1, 'switch'), (2, 1, 'central'),
                            (3, 1, 'chassis'), (4, 2, 'bgp');
`)
	if err != nil {
		t.Fatal(err)
	}
	// Other packages register statements for tables unrelated to this fixture.
	if err := db.PrepareStmts(sqlDB, true); err != nil {
		t.Fatal(err)
	}
	return &serviceDatabase{db: sqlDB, attempts: 1}
}

func (d *serviceDatabase) transaction(ctx context.Context, callback func(context.Context, *sql.Tx) error) error {
	if d.beginError != nil {
		return d.beginError
	}
	for attempt := 0; attempt < d.attempts; attempt++ {
		err := d.runTransaction(ctx, callback, attempt == d.attempts-1)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *serviceDatabase) runTransaction(ctx context.Context, callback func(context.Context, *sql.Tx) error, last bool) error {
	retryErr := errors.New("retry transaction")
	d.inTransaction = true
	defer func() { d.inTransaction = false }()
	err := query.Transaction(ctx, d.db, func(ctx context.Context, tx *sql.Tx) error {
		d.callbackCtx = ctx
		if err := callback(ctx, tx); err != nil {
			return err
		}
		if !last {
			return retryErr
		}
		// Model a transaction error after a successful read, before commit.
		return d.commitError
	})
	if errors.Is(err, retryErr) {
		return nil
	}
	if err != nil {
		return err
	}
	d.committed = true
	return nil
}

func TestActivateEnabledServicesSlowActivation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		database := newServiceDatabase(t)
		database.attempts = 2
		var activated []types.SrvName
		started := time.Now()
		var activationCtx context.Context
		err := activateEnabledServices(context.Background(), "local", true, database.transaction, func(ctx context.Context, service types.SrvName, enable bool) error {
			if database.inTransaction || !database.committed {
				t.Fatal("activation started before the transaction committed")
			}
			if database.callbackCtx.Err() != context.Canceled {
				t.Fatal("transaction context was not cancelled on return")
			}
			if !enable {
				t.Fatal("activation lost the enable flag")
			}
			activationCtx = ctx
			select {
			case <-time.After(4 * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}
			activated = append(activated, service)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if elapsed := time.Since(started); elapsed != 12*time.Second {
			t.Fatalf("activation took %s, want 12s", elapsed)
		}
		want := []types.SrvName{types.SrvCentral, types.SrvChassis, types.SrvSwitch}
		if !reflect.DeepEqual(activated, want) {
			t.Fatalf("activated %v, want %v exactly once despite transaction retry", activated, want)
		}
		if activationCtx.Err() != context.Canceled {
			t.Fatal("activation context was not cancelled on return")
		}
	})
}

func TestActivateEnabledServicesCancellation(t *testing.T) {
	for _, test := range []struct {
		name    string
		parent  func() (context.Context, context.CancelFunc)
		elapsed time.Duration
		wantErr error
	}{
		{"service deadline", func() (context.Context, context.CancelFunc) {
			return context.WithCancel(context.Background())
		}, 30 * time.Second, context.DeadlineExceeded},
		{"parent deadline", func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), 5*time.Second)
		}, 5 * time.Second, context.DeadlineExceeded},
		{"parent cancellation", func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				time.Sleep(time.Second)
				cancel()
			}()
			return ctx, cancel
		}, time.Second, context.Canceled},
	} {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				database := newServiceDatabase(t)
				ctx, cancel := test.parent()
				defer cancel()
				started := time.Now()
				calls := 0
				err := activateEnabledServices(ctx, "local", false, database.transaction, func(ctx context.Context, _ types.SrvName, enable bool) error {
					calls++
					if enable {
						t.Fatal("activation lost the disabled enable flag")
					}
					<-ctx.Done()
					return ctx.Err()
				})
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("error = %v, want %v", err, test.wantErr)
				}
				if calls != 1 {
					t.Fatalf("activated %d services, want stop after first failure", calls)
				}
				if elapsed := time.Since(started); elapsed != test.elapsed {
					t.Fatalf("elapsed = %s, want %s", elapsed, test.elapsed)
				}
			})
		})
	}
}

func TestActivateEnabledServicesPerServiceTimeout(t *testing.T) {
	for _, test := range []struct {
		name           string
		parentDeadline time.Duration
		secondDelay    time.Duration
		wantElapsed    time.Duration
		wantCalls      int
		wantErr        error
	}{
		{"fresh budget for each service", 0, 20 * time.Second, 60 * time.Second, 3, nil},
		{"later service deadline", 0, 31 * time.Second, 50 * time.Second, 2, context.DeadlineExceeded},
		{"parent deadline across services", 35 * time.Second, 20 * time.Second, 35 * time.Second, 2, context.DeadlineExceeded},
	} {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				store := newServiceDatabase(t)
				ctx := context.Background()
				if test.parentDeadline > 0 {
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, test.parentDeadline)
					defer cancel()
				}
				started := time.Now()
				var contexts []context.Context
				err := activateEnabledServices(ctx, "local", true, store.transaction, func(serviceCtx context.Context, _ types.SrvName, _ bool) error {
					if len(contexts) > 0 && contexts[len(contexts)-1].Err() != context.Canceled {
						t.Fatal("previous service context was not cancelled before the next activation")
					}
					contexts = append(contexts, serviceCtx)
					delay := 20 * time.Second
					if len(contexts) == 2 {
						delay = test.secondDelay
					}
					select {
					case <-time.After(delay):
						return nil
					case <-serviceCtx.Done():
						return serviceCtx.Err()
					}
				})
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("error = %v, want %v", err, test.wantErr)
				}
				if len(contexts) != test.wantCalls {
					t.Fatalf("activation calls = %d, want %d", len(contexts), test.wantCalls)
				}
				if elapsed := time.Since(started); elapsed != test.wantElapsed {
					t.Fatalf("elapsed = %s, want %s", elapsed, test.wantElapsed)
				}
				for _, serviceCtx := range contexts {
					if serviceCtx.Err() == nil {
						t.Fatal("service context was not cancelled on return")
					}
				}
			})
		})
	}
}

func TestActivateEnabledServicesDatabaseErrors(t *testing.T) {
	for _, stage := range []string{"begin", "query", "commit"} {
		t.Run(stage, func(t *testing.T) {
			database := newServiceDatabase(t)
			wantErr := errors.New("database failure")
			switch stage {
			case "begin":
				database.beginError = wantErr
			case "query":
				if _, err := database.db.Exec("DROP TABLE services"); err != nil {
					t.Fatal(err)
				}
			case "commit":
				database.commitError = wantErr
			}
			err := activateEnabledServices(context.Background(), "local", true, database.transaction, func(context.Context, types.SrvName, bool) error {
				t.Fatal("activated a service after a database error")
				return nil
			})
			if err == nil {
				t.Fatal("database failure was not propagated")
			}
			if stage != "query" && !errors.Is(err, wantErr) {
				t.Fatalf("error = %v, want %v", err, wantErr)
			}
		})
	}
}

func TestActivateEnabledServicesSnapFailure(t *testing.T) {
	database := newServiceDatabase(t)
	dir := t.TempDir()
	logPath := filepath.Join(dir, "commands")
	// Exercise activateService and snap.Start's real command path, without snapd.
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$COMMAND_LOG\"\n[ \"$2\" != microovn.ovn-northd ]\n"
	if err := os.WriteFile(filepath.Join(dir, "snapctl"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("COMMAND_LOG", logPath)
	err := activateEnabledServices(context.Background(), "local", true, database.transaction, activateService)
	if err == nil || !strings.Contains(err.Error(), "failed to start OVN northd:") {
		t.Fatalf("error = %v, want northd startup failure", err)
	}
	commands, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	want := "start microovn.ovn-ovsdb-server-nb --enable\nstart microovn.ovn-ovsdb-server-sb --enable\nstart microovn.ovn-northd --enable\n"
	if string(commands) != want {
		t.Fatalf("commands = %q, want %q with no subsequent service activation", commands, want)
	}
}

func TestActivateEnabledServicesFailurePreservesDesiredState(t *testing.T) {
	store := newServiceDatabase(t)
	ctx := context.Background()
	readServices := func() []database.Service {
		t.Helper()
		var services []database.Service
		err := store.transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
			var err error
			services, err = database.GetServices(ctx, tx)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return services
	}
	wantServices := readServices()
	wantErr := errors.New("service startup failed")
	fail := true
	var attempts []types.SrvName
	activate := func(_ context.Context, service types.SrvName, enable bool) error {
		if !enable {
			t.Fatal("activation lost the enable flag")
		}
		attempts = append(attempts, service)
		if fail && service == types.SrvChassis {
			return wantErr
		}
		return nil
	}
	err := activateEnabledServices(ctx, "local", true, store.transaction, activate)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	wantAttempts := []types.SrvName{types.SrvCentral, types.SrvChassis}
	if !reflect.DeepEqual(attempts, wantAttempts) {
		t.Fatalf("activation attempts = %v, want %v", attempts, wantAttempts)
	}
	if services := readServices(); !reflect.DeepEqual(services, wantServices) {
		t.Fatalf("desired services after failure = %v, want unchanged %v", services, wantServices)
	}

	// Model the next startup using the same persisted desired state. Retry both
	// the previously started service and the failed service, then finish the list.
	fail = false
	if err := activateEnabledServices(ctx, "local", true, store.transaction, activate); err != nil {
		t.Fatal(err)
	}
	wantAttempts = append(wantAttempts, types.SrvCentral, types.SrvChassis, types.SrvSwitch)
	if !reflect.DeepEqual(attempts, wantAttempts) {
		t.Fatalf("activation attempts across startups = %v, want %v", attempts, wantAttempts)
	}
	if services := readServices(); !reflect.DeepEqual(services, wantServices) {
		t.Fatalf("desired services after retry = %v, want unchanged %v", services, wantServices)
	}
}
