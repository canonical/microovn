// Package database provides the database access functions and schema.
package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/canonical/lxd/lxd/db/query"
	"github.com/canonical/lxd/lxd/db/schema"
)

// SchemaExtensions is a list of schema extensions that can be passed to the MicroCluster daemon.
// Each entry will increase the database schema version by one, and will be applied after internal schema updates.
var SchemaExtensions = []schema.Update{
	schemaUpdate1,
	schemaUpdate2,
	schemaUpdate3,
}

// getClusterTableName returns the name of the table that holds the record of cluster members from sqlite_master.
// Prior to microcluster V2, this table was called `internal_cluster_members`, but now it is `core_cluster_members`.
// Since extensions to the database may be at an earlier version (either 1 or 2), this helper will dynamically determine the table name to use.
func getClusterTableName(ctx context.Context, tx *sql.Tx) (string, error) {
	stmt := "SELECT name FROM sqlite_master WHERE name = 'internal_cluster_members' OR name = 'core_cluster_members'"
	tables, err := query.SelectStrings(ctx, tx, stmt)
	if err != nil {
		return "", err
	}

	if len(tables) != 1 || tables[0] == "" {
		return "", fmt.Errorf("No cluster members table found")
	}

	return tables[0], nil
}

func schemaUpdate1(ctx context.Context, tx *sql.Tx) error {
	tableName, err := getClusterTableName(ctx, tx)
	if err != nil {
		return err
	}

	stmt := fmt.Sprintf(`
CREATE TABLE config (
  id                            INTEGER  PRIMARY KEY AUTOINCREMENT NOT NULL,
  key                           TEXT     NOT  NULL,
  value                         TEXT     NOT  NULL,
  UNIQUE(key)
);

CREATE TABLE services (
  id                            INTEGER  PRIMARY KEY AUTOINCREMENT NOT NULL,
  member_id                     INTEGER  NOT  NULL,
  service                       TEXT     NOT  NULL,
  FOREIGN KEY (member_id) REFERENCES "%s" (id)
  UNIQUE(member_id, service)
);
  `, tableName)

	_, err = tx.ExecContext(ctx, stmt)

	return err
}

// schemaUpdate2 ensures that records from 'services' are properly deleted
// when associated 'internal_cluster_member' is removed.
func schemaUpdate2(ctx context.Context, tx *sql.Tx) error {
	tableName, err := getClusterTableName(ctx, tx)
	if err != nil {
		return err
	}

	stmt := fmt.Sprintf(`
PRAGMA foreign_keys = OFF;
CREATE TABLE services_new (
  id                            INTEGER  PRIMARY KEY AUTOINCREMENT NOT NULL,
  member_id                     INTEGER  NOT  NULL,
  service                       TEXT     NOT  NULL,
  FOREIGN KEY (member_id) REFERENCES "%s" (id) ON DELETE CASCADE
  UNIQUE(member_id, service)
);

INSERT INTO services_new SELECT id, member_id, service FROM services;

DROP TABLE services;
ALTER TABLE services_new RENAME TO services;
PRAGMA foreign_keys = ON;
`, tableName)

	_, err = tx.ExecContext(ctx, stmt)

	return err
}

// schemaUpdate3 ensures that the `services` table properly references the new table name `core_cluster_members`.
func schemaUpdate3(ctx context.Context, tx *sql.Tx) error {
	stmt := `
CREATE TABLE services_new (
  id                            INTEGER  PRIMARY KEY AUTOINCREMENT NOT NULL,
  member_id                     INTEGER  NOT  NULL,
  service                       TEXT     NOT  NULL,
  FOREIGN KEY (member_id) REFERENCES "core_cluster_members" (id) ON DELETE CASCADE
  UNIQUE(member_id, service)
);

INSERT INTO services_new SELECT id, member_id, service FROM services;

DROP TABLE services;
ALTER TABLE services_new RENAME TO services;
	`

	_, err := tx.ExecContext(ctx, stmt)

	return err
}
