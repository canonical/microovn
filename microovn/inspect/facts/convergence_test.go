package facts

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestProbeDatabaseCommunicationConvergence(t *testing.T) {
	tests := []struct {
		name          string
		sbValues      []int64
		deadline      time.Duration
		wantConverged bool
	}{
		{name: "immediate", sbValues: []int64{10}, deadline: time.Second, wantConverged: true},
		{name: "delayed", sbValues: []int64{8, 9, 10}, deadline: time.Second, wantConverged: true},
		{name: "deadline", sbValues: []int64{9}, deadline: 10 * time.Millisecond, wantConverged: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			index := 0
			readSB := func(context.Context) (int64, error) {
				if index < len(test.sbValues)-1 {
					value := test.sbValues[index]
					index++
					return value, nil
				}
				return test.sbValues[len(test.sbValues)-1], nil
			}

			evidence := probeDatabaseCommunication(
				context.Background(),
				func(context.Context) (int64, error) { return 10, nil },
				readSB,
				time.Millisecond,
				test.deadline,
			)

			assertBoolPointer(t, "NBReachable", evidence.NBReachable, true)
			assertBoolPointer(t, "SBReachable", evidence.SBReachable, true)
			assertBoolPointer(t, "Converged", evidence.Converged, test.wantConverged)
			if evidence.NBCfg == nil || *evidence.NBCfg != 10 {
				t.Fatalf("NBCfg = %v, want 10", evidence.NBCfg)
			}
			if evidence.SBCfg == nil {
				t.Fatal("SBCfg is nil, want an observed value")
			}
		})
	}
}

func TestProbeDatabaseCommunicationFailures(t *testing.T) {
	t.Run("northbound unreachable", func(t *testing.T) {
		evidence := probeDatabaseCommunication(
			context.Background(),
			func(context.Context) (int64, error) { return 0, errors.New("unreachable") },
			func(context.Context) (int64, error) { return 0, nil },
			time.Millisecond,
			time.Second,
		)

		assertBoolPointer(t, "NBReachable", evidence.NBReachable, false)
		if evidence.SBReachable != nil {
			t.Fatalf("SBReachable = %v, want unknown", *evidence.SBReachable)
		}
	})

	t.Run("southbound unreachable on first read", func(t *testing.T) {
		evidence := probeDatabaseCommunication(
			context.Background(),
			func(context.Context) (int64, error) { return 10, nil },
			func(context.Context) (int64, error) { return 0, errors.New("unreachable") },
			time.Millisecond,
			time.Second,
		)

		assertBoolPointer(t, "NBReachable", evidence.NBReachable, true)
		assertBoolPointer(t, "SBReachable", evidence.SBReachable, false)
		if evidence.SBCfg != nil {
			t.Fatalf("SBCfg = %v, want no observation", *evidence.SBCfg)
		}
		if evidence.Converged != nil {
			t.Fatalf("Converged = %v, want unknown", *evidence.Converged)
		}
	})

	t.Run("southbound preserves prior observation", func(t *testing.T) {
		calls := 0
		readSB := func(context.Context) (int64, error) {
			calls++
			if calls == 1 {
				return 5, nil
			}
			return 0, errors.New("unreachable")
		}

		evidence := probeDatabaseCommunication(
			context.Background(),
			func(context.Context) (int64, error) { return 10, nil },
			readSB,
			time.Millisecond,
			time.Second,
		)

		assertBoolPointer(t, "SBReachable", evidence.SBReachable, true)
		if evidence.SBCfg == nil || *evidence.SBCfg != 5 {
			t.Fatalf("SBCfg = %v, want preserved value 5", evidence.SBCfg)
		}
		if evidence.Converged != nil {
			t.Fatalf("Converged = %v, want unknown", *evidence.Converged)
		}
	})
}

func TestProbeDatabaseCommunicationMalformedOutput(t *testing.T) {
	t.Run("northbound", func(t *testing.T) {
		evidence := probeDatabaseCommunication(
			context.Background(),
			func(context.Context) (int64, error) { return 0, dbCfgParseError{database: "NB"} },
			func(context.Context) (int64, error) { return 0, nil },
			time.Millisecond,
			time.Second,
		)

		assertBoolPointer(t, "NBReachable", evidence.NBReachable, true)
		if evidence.Converged != nil {
			t.Fatalf("Converged = %v, want unknown", *evidence.Converged)
		}
		if !strings.Contains(evidence.CollectionError, "invalid NB nb_cfg value") {
			t.Fatalf("CollectionError = %q", evidence.CollectionError)
		}
	})

	t.Run("southbound first read", func(t *testing.T) {
		evidence := probeDatabaseCommunication(
			context.Background(),
			func(context.Context) (int64, error) { return 10, nil },
			func(context.Context) (int64, error) { return 0, dbCfgParseError{database: "SB"} },
			time.Millisecond,
			time.Second,
		)

		assertBoolPointer(t, "SBReachable", evidence.SBReachable, true)
		if evidence.SBCfg != nil || evidence.Converged != nil {
			t.Fatalf("evidence = %#v, want no SB value and unknown convergence", evidence)
		}
	})

	t.Run("southbound preserves prior observation", func(t *testing.T) {
		calls := 0
		evidence := probeDatabaseCommunication(
			context.Background(),
			func(context.Context) (int64, error) { return 10, nil },
			func(context.Context) (int64, error) {
				calls++
				if calls == 1 {
					return 5, nil
				}
				return 0, dbCfgParseError{database: "SB"}
			},
			time.Millisecond,
			time.Second,
		)

		assertBoolPointer(t, "SBReachable", evidence.SBReachable, true)
		if evidence.SBCfg == nil || *evidence.SBCfg != 5 {
			t.Fatalf("SBCfg = %v, want preserved value 5", evidence.SBCfg)
		}
		if evidence.Converged != nil {
			t.Fatalf("Converged = %v, want unknown", *evidence.Converged)
		}
	})
}

func TestProbeDatabaseCommunicationCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	evidence := probeDatabaseCommunication(
		ctx,
		func(ctx context.Context) (int64, error) { return 0, ctx.Err() },
		func(context.Context) (int64, error) { return 0, nil },
		time.Millisecond,
		time.Second,
	)

	if evidence.NBReachable != nil {
		t.Fatalf("NBReachable = %v, want unknown", *evidence.NBReachable)
	}
	if !strings.Contains(evidence.CollectionError, context.Canceled.Error()) {
		t.Fatalf("CollectionError = %q, want cancellation cause", evidence.CollectionError)
	}
}

func TestProbeDatabaseCommunicationFirstSBReadDeadline(t *testing.T) {
	evidence := probeDatabaseCommunication(
		context.Background(),
		func(context.Context) (int64, error) { return 10, nil },
		func(ctx context.Context) (int64, error) {
			<-ctx.Done()
			return 0, ctx.Err()
		},
		time.Millisecond,
		time.Millisecond,
	)

	assertBoolPointer(t, "NBReachable", evidence.NBReachable, true)
	if evidence.SBReachable != nil || evidence.SBCfg != nil || evidence.Converged != nil {
		t.Fatalf("evidence = %#v, want unknown SB state", evidence)
	}
	if !strings.Contains(evidence.CollectionError, context.DeadlineExceeded.Error()) {
		t.Fatalf("CollectionError = %q, want deadline cause", evidence.CollectionError)
	}
}

func assertBoolPointer(t *testing.T, name string, got *bool, want bool) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s is nil, want %t", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %t, want %t", name, *got, want)
	}
}
