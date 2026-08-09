package migrations

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

func GetMigrations() []Migration {
	return []Migration{
		{
			ID:      "001_initial_schema",
			Comment: "Initialize Redis keys and indexes",
			Up:      migrate001Up,
			Down:    migrate001Down,
		},
		{
			ID:      "002_add_referral_tracking",
			Comment: "Add referral tracking support",
			Up:      migrate002Up,
			Down:    migrate002Down,
		},
		{
			ID:      "003_add_request_logging",
			Comment: "Initialize request logging keys",
			Up:      migrate003Up,
			Down:    migrate003Down,
		},
		{
			ID:      "004_add_scaling_policies",
			Comment: "Add default scaling policies",
			Up:      migrate004Up,
			Down:    migrate004Down,
		},
	}
}

func migrate001Up(ctx context.Context, rdb *redis.Client) error {
	now := time.Now().Unix()
	pipe := rdb.Pipeline()
	pipe.Set(ctx, "schema:version", "004", 0)
	pipe.Set(ctx, "schema:initialized_at", now, 0)
	pipe.SAdd(ctx, "indexes:countries", "US", "EU", "APAC")
	_, err := pipe.Exec(ctx)
	return err
}

func migrate001Down(ctx context.Context, rdb *redis.Client) error {
	pipe := rdb.Pipeline()
	pipe.Del(ctx, "schema:version")
	pipe.Del(ctx, "schema:initialized_at")
	pipe.Del(ctx, "indexes:countries")
	_, err := pipe.Exec(ctx)
	return err
}

func migrate002Up(ctx context.Context, rdb *redis.Client) error {
	return rdb.Set(ctx, "feature:referrals:enabled", "true", 0).Err()
}

func migrate002Down(ctx context.Context, rdb *redis.Client) error {
	pipe := rdb.Pipeline()
	pipe.Del(ctx, "feature:referrals:enabled")
	keys, _ := rdb.Keys(ctx, "referrals:*").Result()
	for _, key := range keys {
		pipe.Del(ctx, key)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func migrate003Up(ctx context.Context, rdb *redis.Client) error {
	return rdb.Set(ctx, "feature:request_logging:enabled", "false", 0).Err()
}

func migrate003Down(ctx context.Context, rdb *redis.Client) error {
	keys, _ := rdb.Keys(ctx, "reqlog:*").Result()
	pipe := rdb.Pipeline()
	for _, key := range keys {
		pipe.Del(ctx, key)
	}
	pipe.Del(ctx, "feature:request_logging:enabled")
	_, err := pipe.Exec(ctx)
	return err
}

func migrate004Up(ctx context.Context, rdb *redis.Client) error {
	return rdb.Set(ctx, "feature:auto_scaling:enabled", "true", 0).Err()
}

func migrate004Down(ctx context.Context, rdb *redis.Client) error {
	return rdb.Del(ctx, "feature:auto_scaling:enabled").Err()
}
