package migrations

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/go-redis/redis/v8"
)

type Migration struct {
	ID      string
	Up      func(ctx context.Context, rdb *redis.Client) error
	Down    func(ctx context.Context, rdb *redis.Client) error
	Comment string
}

type MigrationRunner struct {
	rdb         *redis.Client
	migrations  []Migration
	migKey      string
}

func NewMigrationRunner(rdb *redis.Client, migrations []Migration) *MigrationRunner {
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].ID < migrations[j].ID
	})
	return &MigrationRunner{
		rdb:        rdb,
		migrations: migrations,
		migKey:     "migrations:applied",
	}
}

func (mr *MigrationRunner) Run(ctx context.Context) error {
	applied, err := mr.rdb.SMembers(ctx, mr.migKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedSet := make(map[string]bool)
	for _, id := range applied {
		appliedSet[id] = true
	}

	for _, migration := range mr.migrations {
		if appliedSet[migration.ID] {
			continue
		}

		fmt.Printf("Running migration: %s - %s\n", migration.ID, migration.Comment)

		if err := migration.Up(ctx, mr.rdb); err != nil {
			return fmt.Errorf("migration %s failed: %w", migration.ID, err)
		}

		if err := mr.rdb.SAdd(ctx, mr.migKey, migration.ID).Err(); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", migration.ID, err)
		}

		fmt.Printf("Migration %s completed successfully\n", migration.ID)
	}

	return nil
}

func (mr *MigrationRunner) Status(ctx context.Context) ([]string, []string, error) {
	applied, err := mr.rdb.SMembers(ctx, mr.migKey).Result()
	if err != nil {
		return nil, nil, err
	}

	appliedSet := make(map[string]bool)
	for _, id := range applied {
		appliedSet[id] = true
	}

	var pending []string
	for _, migration := range mr.migrations {
		if !appliedSet[migration.ID] {
			pending = append(pending, migration.ID)
		}
	}

	return applied, pending, nil
}

func (mr *MigrationRunner) Rollback(ctx context.Context, targetID string) error {
	applied, _, err := mr.Status(ctx)
	if err != nil {
		return err
	}

	appliedSet := make(map[string]bool)
	for _, id := range applied {
		appliedSet[id] = true
	}

	var toRollback []Migration
	for _, migration := range mr.migrations {
		if appliedSet[migration.ID] && migration.ID > targetID {
			toRollback = append(toRollback, migration)
		}
	}

	for i := len(toRollback) - 1; i >= 0; i-- {
		migration := toRollback[i]
		fmt.Printf("Rolling back migration: %s\n", migration.ID)

		if migration.Down != nil {
			if err := migration.Down(ctx, mr.rdb); err != nil {
				return fmt.Errorf("rollback %s failed: %w", migration.ID, err)
			}
		}

		if err := mr.rdb.SRem(ctx, mr.migKey, migration.ID).Err(); err != nil {
			return fmt.Errorf("failed to remove migration %s: %w", migration.ID, err)
		}

		fmt.Printf("Migration %s rolled back successfully\n", migration.ID)
	}

	return nil
}

func RecordMigrationTimestamp(ctx context.Context, rdb *redis.Client, migrationID string) error {
	key := "migration:timestamp:" + migrationID
	return rdb.Set(ctx, key, time.Now().Unix(), 0).Err()
}

func GetMigrationTimestamp(ctx context.Context, rdb *redis.Client, migrationID string) (int64, error) {
	key := "migration:timestamp:" + migrationID
	val, err := rdb.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}
