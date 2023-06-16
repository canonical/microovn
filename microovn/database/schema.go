// Package database provides the database access functions and schema.
package database

import (
	"context"
	"database/sql"

	"github.com/lxc/lxd/lxd/db/schema"
)

// SchemaExtensions is a list of schema extensions that can be passed to the MicroCluster daemon.
// Each entry will increase the database schema version by one, and will be applied after internal schema updates.
var SchemaExtensions = map[int]schema.Update{
	1: schemaUpdate1,
	2: schemaUpdateCascadeDeleteServices,
}

func schemaUpdate1(ctx context.Context, tx *sql.Tx) error {
	stmt := `
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
  FOREIGN KEY (member_id) REFERENCES "internal_cluster_members" (id)
  UNIQUE(member_id, service)
);
  `

	_, err := tx.Exec(stmt)

	return err
}

// schemaUpdateCascadeDeleteServices ensures that records from 'services' are properly deleted
// when associated 'internal_cluster_member' is removed.
func schemaUpdateCascadeDeleteServices(_ context.Context, tx *sql.Tx) error {
	stmt := `
PRAGMA foreign_keys = OFF;
CREATE TABLE services_new (
  id                            INTEGER  PRIMARY KEY AUTOINCREMENT NOT NULL,
  member_id                     INTEGER  NOT  NULL,
  service                       TEXT     NOT  NULL,
  FOREIGN KEY (member_id) REFERENCES "internal_cluster_members" (id) ON DELETE CASCADE
  UNIQUE(member_id, service)
);

INSERT INTO services_new SELECT id, member_id, service FROM services;

DROP TABLE services;
ALTER TABLE services_new RENAME TO services;
PRAGMA foreign_keys = ON;
`
	_, err := tx.Exec(stmt)

	return err
}
