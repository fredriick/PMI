package migrations

import (
	"context"
	"testing"

	"github.com/go-redis/redis/v8"
)

func TestRunMigrations(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15})
	_ = rdb.FlushDB(context.Background()).Err()
	defer rdb.Close()

	migrations := GetMigrations()
	runner := NewMigrationRunner(rdb, migrations)

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	applied, pending, err := runner.Status(context.Background())
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	if len(applied) != len(migrations) {
		t.Fatalf("expected %d applied, got %d", len(migrations), len(applied))
	}
	if len(pending) != 0 {
		t.Fatalf("expected 0 pending, got %d", len(pending))
	}
}

func TestRunMigrationsIdempotent(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15})
	_ = rdb.FlushDB(context.Background()).Err()
	defer rdb.Close()

	migrations := GetMigrations()
	runner := NewMigrationRunner(rdb, migrations)

	_ = runner.Run(context.Background())
	_ = runner.Run(context.Background())

	_, pending, _ := runner.Status(context.Background())
	if len(pending) != 0 {
		t.Fatalf("expected 0 pending after second run, got %d", len(pending))
	}
}

func TestRollback(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15})
	_ = rdb.FlushDB(context.Background()).Err()
	defer rdb.Close()

	migrations := GetMigrations()
	runner := NewMigrationRunner(rdb, migrations)

	_ = runner.Run(context.Background())
	if err := runner.Rollback(context.Background(), "001_initial_schema"); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	_, pending, _ := runner.Status(context.Background())
	if len(pending) != len(migrations) {
		t.Fatalf("expected %d pending after rollback, got %d", len(migrations), len(pending))
	}
}

func TestRecordMigrationTimestamp(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15})
	_ = rdb.FlushDB(context.Background()).Err()
	defer rdb.Close()

	if err := RecordMigrationTimestamp(context.Background(), rdb, "test_migration"); err != nil {
		t.Fatalf("RecordMigrationTimestamp failed: %v", err)
	}

	ts, err := GetMigrationTimestamp(context.Background(), rdb, "test_migration")
	if err != nil {
		t.Fatalf("GetMigrationTimestamp failed: %v", err)
	}

	if ts <= 0 {
		t.Fatalf("expected timestamp > 0, got %d", ts)
	}
}
