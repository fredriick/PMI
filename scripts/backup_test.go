package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-redis/redis/v8"
)

func TestBackupAndRestore(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15})
	_ = rdb.FlushDB(context.Background()).Err()
	defer rdb.Close()

	ctx := context.Background()
	rdb.Set(ctx, "test:string", "hello", 0)
	rdb.HSet(ctx, "test:hash", map[string]string{"field1": "value1"})
	rdb.RPush(ctx, "test:list", "item1", "item2")
	rdb.SAdd(ctx, "test:set", "member1", "member2")
	rdb.ZAdd(ctx, "test:zset", &redis.Z{Score: 1.0, Member: "z1"})

	backupDir, err := os.MkdirTemp("", "redis-backup-test")
	if err != nil {
		t.Fatalf(" MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(backupDir)

	doBackup(ctx, rdb, backupDir, "*")

	files, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected at least one backup file")
	}

	_ = rdb.FlushDB(ctx).Err()

	doRestore(ctx, rdb, backupDir, files[0].Name())

	val, _ := rdb.Get(ctx, "test:string").Result()
	if val != "hello" {
		t.Fatalf("expected 'hello', got %s", val)
	}

	hash, _ := rdb.HGetAll(ctx, "test:hash").Result()
	if hash["field1"] != "value1" {
		t.Fatalf("expected hash field1='value1', got %s", hash["field1"])
	}

	list, _ := rdb.LRange(ctx, "test:list", 0, -1).Result()
	if len(list) != 2 || list[0] != "item1" {
		t.Fatalf("unexpected list: %v", list)
	}

	set, _ := rdb.SMembers(ctx, "test:set").Result()
	found := false
	for _, m := range set {
		if m == "member1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected member1 in set")
	}

	zset, _ := rdb.ZRangeWithScores(ctx, "test:zset", 0, -1).Result()
	if len(zset) != 1 || zset[0].Score != 1.0 {
		t.Fatalf("unexpected zset: %v", zset)
	}
}

func TestBackupJSONFormat(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15})
	_ = rdb.FlushDB(context.Background()).Err()
	defer rdb.Close()

	ctx := context.Background()
	rdb.Set(ctx, "backup:test", "value", 0)

	backupDir, err := os.MkdirTemp("", "redis-backup-format-test")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(backupDir)

	doBackup(ctx, rdb, backupDir, "*")

	files, _ := os.ReadDir(backupDir)
	if len(files) == 0 {
		t.Fatal("expected backup file")
	}

	f, _ := os.Open(filepath.Join(backupDir, files[0].Name()))
	defer f.Close()

	gr, _ := gzip.NewReader(f)
	defer gr.Close()

	var metadata BackupMetadata
	_ = json.NewDecoder(gr).Decode(&metadata)

	if metadata.Keys <= 0 {
		t.Fatalf("expected keys > 0, got %d", metadata.Keys)
	}
	if metadata.Size <= 0 {
		t.Fatalf("expected size > 0, got %d", metadata.Size)
	}
}

func TestFormatSize(t *testing.T) {
	if formatSize(0) != "0 B" {
		t.Fatalf("expected '0 B', got %s", formatSize(0))
	}
	if formatSize(1024) != "1.0 KB" {
		t.Fatalf("expected '1.0 KB', got %s", formatSize(1024))
	}
	if formatSize(1024*1024) != "1.0 MB" {
		t.Fatalf("expected '1.0 MB', got %s", formatSize(1024*1024))
	}
}

func TestBackupSkipsMetaKeys(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15})
	_ = rdb.FlushDB(context.Background()).Err()
	defer rdb.Close()

	ctx := context.Background()
	rdb.Set(ctx, "skip:test", "value", 0)

	backupDir, _ := os.MkdirTemp("", "redis-backup-skip-test")
	defer os.RemoveAll(backupDir)

	doBackup(ctx, rdb, backupDir, "*")

	files, _ := os.ReadDir(backupDir)
	f, _ := os.Open(filepath.Join(backupDir, files[0].Name()))
	defer f.Close()

	gr, _ := gzip.NewReader(f)
	defer gr.Close()

	var metadata BackupMetadata
	_ = json.NewDecoder(gr).Decode(&metadata)

	found := false
	for _, entry := range metadata.Entries {
		if entry.Key == "skip:test" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected to find key 'skip:test' in backup")
	}
}
