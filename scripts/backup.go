package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

type BackupEntry struct {
	Key   string      `json:"key"`
	Type  string      `json:"type"`
	TTL   int64       `json:"ttl_seconds"`
	Value interface{} `json:"value"`
}

type BackupMetadata struct {
	Timestamp time.Time     `json:"timestamp"`
	Keys      int           `json:"keys_count"`
	Databases int           `json:"databases"`
	Size      int64         `json:"size_bytes"`
	Entries   []BackupEntry `json:"entries,omitempty"`
}

func main() {
	ctx := context.Background()

	redisAddr := flag.String("redis", "localhost:6379", "Redis address")
	redisPassword := flag.String("password", "", "Redis password")
	redisDB := flag.Int("db", 0, "Redis database")
	backupDir := flag.String("dir", "./backups", "Backup directory")
	action := flag.String("action", "backup", "Action: backup, restore, list")
	backupFile := flag.String("file", "", "Backup file for restore")
	pattern := flag.String("pattern", "*", "Key pattern for backup (Redis glob)")
	flag.Parse()

	rdb := redis.NewClient(&redis.Options{
		Addr:     *redisAddr,
		Password: *redisPassword,
		DB:       *redisDB,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer rdb.Close()

	switch *action {
	case "backup":
		doBackup(ctx, rdb, *backupDir, *pattern)
	case "restore":
		if *backupFile == "" {
			log.Fatal("Backup file is required for restore (use -file flag)")
		}
		doRestore(ctx, rdb, *backupDir, *backupFile)
	case "list":
		listBackups(*backupDir)
	default:
		log.Fatalf("Unknown action: %s (use backup, restore, or list)", *action)
	}
}

func doBackup(ctx context.Context, rdb *redis.Client, backupDir, pattern string) {
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		log.Fatalf("Failed to create backup directory: %v", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("redis_backup_%s.json.gz", timestamp)
	filepath := filepath.Join(backupDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		log.Fatalf("Failed to create backup file: %v", err)
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()

	encoder := json.NewEncoder(gw)
	encoder.SetIndent("", "  ")

	keys, err := rdb.Keys(ctx, pattern).Result()
	if err != nil {
		log.Fatalf("Failed to scan keys: %v", err)
	}

	log.Printf("Found %d keys matching pattern: %s", len(keys), pattern)

	entries := make([]BackupEntry, 0, len(keys))
	for _, key := range keys {
		entry, err := backupKey(ctx, rdb, key)
		if err != nil {
			log.Printf("Warning: failed to backup key %s: %v", key, err)
			continue
		}
		entries = append(entries, entry)
	}

	metadata := BackupMetadata{
		Timestamp: time.Now(),
		Keys:      len(entries),
		Databases: 1,
		Size:      getEnvSize(entries),
		Entries:   entries,
	}

	if err := encoder.Encode(metadata); err != nil {
		log.Fatalf("Failed to write backup: %v", err)
	}

	log.Printf("Backup created: %s (%d keys)", filepath, len(entries))
}

func doRestore(ctx context.Context, rdb *redis.Client, backupDir, filename string) {
	filepath := filepath.Join(backupDir, filename)
	file, err := os.Open(filepath)
	if err != nil {
		log.Fatalf("Failed to open backup file: %v", err)
	}
	defer file.Close()

	gr, err := gzip.NewReader(file)
	if err != nil {
		log.Fatalf("Failed to decompress backup: %v", err)
	}
	defer gr.Close()

	var metadata BackupMetadata
	if err := json.NewDecoder(gr).Decode(&metadata); err != nil {
		log.Fatalf("Failed to parse backup: %v", err)
	}

	log.Printf("Restoring backup from %s (%d keys)", metadata.Timestamp.Format(time.RFC3339), metadata.Keys)

	pipe := rdb.Pipeline()
	for _, entry := range metadata.Entries {
		switch entry.Type {
		case "string":
			pipe.Set(ctx, entry.Key, entry.Value, 0)
		case "hash":
			if val, ok := entry.Value.(map[string]interface{}); ok {
				for field, value := range val {
					pipe.HSet(ctx, entry.Key, field, value)
				}
			}
		case "list":
			if val, ok := entry.Value.([]interface{}); ok {
				for _, item := range val {
					pipe.RPush(ctx, entry.Key, item)
				}
			}
		case "set":
			if val, ok := entry.Value.([]interface{}); ok {
				for _, item := range val {
					pipe.SAdd(ctx, entry.Key, item)
				}
			}
		case "zset":
			if val, ok := entry.Value.([]interface{}); ok {
				for _, item := range val {
					if pair, ok := item.(map[string]interface{}); ok {
						if score, ok := pair["score"].(float64); ok {
							pipe.ZAdd(ctx, entry.Key, &redis.Z{Score: score, Member: pair["member"]})
						}
					}
				}
			}
		}
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		log.Fatalf("Failed to restore backup: %v", err)
	}

	log.Printf("Restore complete: %d keys restored", metadata.Keys)
}

func listBackups(backupDir string) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		log.Fatalf("Failed to read backup directory: %v", err)
	}

	fmt.Printf("%-40s %-12s %-15s\n", "FILENAME", "SIZE", "DATE")
	fmt.Println(strings.Repeat("-", 70))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json.gz") {
			continue
		}
		info, _ := entry.Info()
		fmt.Printf("%-40s %-12s %-15s\n", entry.Name(), formatSize(info.Size()), info.ModTime().Format("2006-01-02 15:04"))
	}
}

func backupKey(ctx context.Context, rdb *redis.Client, key string) (BackupEntry, error) {
	entry := BackupEntry{Key: key}

	keyType, err := rdb.Type(ctx, key).Result()
	if err != nil {
		return entry, err
	}
	entry.Type = keyType

	ttl, err := rdb.TTL(ctx, key).Result()
	if err != nil {
		return entry, err
	}
	entry.TTL = int64(ttl.Seconds())

	switch keyType {
	case "string":
		val, err := rdb.Get(ctx, key).Result()
		if err != nil && err != redis.Nil {
			return entry, err
		}
		entry.Value = val
	case "hash":
		val, err := rdb.HGetAll(ctx, key).Result()
		if err != nil {
			return entry, err
		}
		entry.Value = val
	case "list":
		val, err := rdb.LRange(ctx, key, 0, -1).Result()
		if err != nil {
			return entry, err
		}
		entry.Value = val
	case "set":
		val, err := rdb.SMembers(ctx, key).Result()
		if err != nil {
			return entry, err
		}
		entry.Value = val
	case "zset":
		val, err := rdb.ZRangeWithScores(ctx, key, 0, -1).Result()
		if err != nil {
			return entry, err
		}
		zset := make([]map[string]interface{}, 0, len(val))
		for _, z := range val {
			zset = append(zset, map[string]interface{}{
				"score":  z.Score,
				"member": z.Member,
			})
		}
		entry.Value = zset
	default:
		entry.Value = nil
	}

	return entry, nil
}

func getEnvSize(entries []BackupEntry) int64 {
	size := int64(0)
	for _, e := range entries {
		size += int64(len(e.Key))
		switch v := e.Value.(type) {
		case string:
			size += int64(len(v))
		case map[string]interface{}:
			for _, val := range v {
				if s, ok := val.(string); ok {
					size += int64(len(s))
				}
			}
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok {
					size += int64(len(s))
				}
			}
		}
	}
	return size
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
