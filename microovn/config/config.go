package config

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/canonical/microcluster/v2/state"
	"github.com/canonical/microovn/microovn/database"
)

// SetConfig function inserts or updates rows in the "config" table of the MicroOVNs database.
func SetConfig(ctx context.Context, s state.State, key string, value string) error {
	// Upsert config value in the database
	err := s.Database().Transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		exists, err := database.ConfigItemExists(ctx, tx, key)
		if err != nil {
			return fmt.Errorf("failed to check if config '%s' exists: %s", key, err)
		}
		if exists {
			err = database.UpdateConfigItem(ctx, tx, key, database.ConfigItem{Key: key, Value: value})
		} else {
			_, err = database.CreateConfigItem(ctx, tx, database.ConfigItem{Key: key, Value: value})
		}
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to set config '%s' into database: %s", key, err)
	}
	return nil
}

// GetConfig function retrieves items from the config table of the MicroOVN's database. In case that a row
// with the given key does not exist in the table, both returned item and error are nil.
func GetConfig(ctx context.Context, s state.State, key string) (*database.ConfigItem, error) {
	var err error
	var item *database.ConfigItem
	err = s.Database().Transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		exists, err := database.ConfigItemExists(ctx, tx, key)
		if err != nil {
			return fmt.Errorf("failed to check if config '%s' exists: %s", key, err)
		}

		if !exists {
			return nil
		}

		item, err = database.GetConfigItem(ctx, tx, key)
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get config '%s' from database: %v", key, err)
	}
	return item, nil
}

// DeleteConfig removes an item with the specified key from the config table of the MicroOVN's database.
// If the item is not present in the table, this function returns successfully.
func DeleteConfig(ctx context.Context, s state.State, key string) error {
	err := s.Database().Transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		exists, err := database.ConfigItemExists(ctx, tx, key)
		if err != nil {
			return fmt.Errorf("failed to check if config '%s' exists: %s", key, err)
		}
		if !exists {
			return nil
		}
		return database.DeleteConfigItem(ctx, tx, key)
	})

	if err != nil {
		return fmt.Errorf("failed to delete config '%s' from database: %s", key, err)
	}
	return nil
}
