package models

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/go-redis/redis/v8"
)

type TvChannel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type RedisStore struct {
	Client *redis.Client
	Prefix string
}

func NewAuthenticatedRedisClient(ctx context.Context) (*RedisStore, error) {
	addr, ok := os.LookupEnv("REDIS_ADDR")
	if !ok {
		return nil, fmt.Errorf("REDIS_ADDR environment variable is not set")
	}
	password := os.Getenv("REDIS_PASSWORD")

	db, ok := os.LookupEnv("REDIS_DB")
	if !ok {
		db = "0"
	}
	dbInt := 0
	if parsed, err := strconv.Atoi(db); err == nil {
		dbInt = parsed
	}
	return newRedisClient(ctx, addr, password, dbInt)
}

func newRedisClient(ctx context.Context, addr, password string, db int) (*RedisStore, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	return &RedisStore{Client: rdb}, nil
}

func (r *RedisStore) Save(ctx context.Context, tvChannel TvChannel) error {
	// Set a hash with channel information
	channelKey := fmt.Sprintf("%s:%s", r.Prefix, tvChannel.ID)
	_, err := r.Client.HSet(ctx, channelKey, map[string]interface{}{
		"id":   tvChannel.ID,
		"name": tvChannel.Name,
		"url":  tvChannel.URL,
	}).Result()
	if err != nil {
		return err
	}

	// Set indexes for name and URL
	nameKey := fmt.Sprintf("%s:name:%s", r.Prefix, tvChannel.Name)
	if err := r.Client.Set(ctx, nameKey, tvChannel.ID, 0).Err(); err != nil {
		return err
	}

	urlKey := fmt.Sprintf("%s:url:%s", r.Prefix, tvChannel.URL)
	if err := r.Client.Set(ctx, urlKey, tvChannel.ID, 0).Err(); err != nil {
		return err
	}

	log.Printf("Saved channel %s", tvChannel.Name)
	return nil
}

// GetChannelByID retrieves channel data by ID
func (r *RedisStore) GetChannelByID(ctx context.Context, id int64) (*TvChannel, error) {
	id_str := strconv.FormatInt(id, 10)
	channelKey := fmt.Sprintf("%s:%s", r.Prefix, id_str)

	data, err := r.Client.HGetAll(ctx, channelKey).Result()
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("channel not found: %s", id_str)
	}

	channel := &TvChannel{
		ID:   data["id"],
		Name: data["name"],
		URL:  data["url"],
	}

	return channel, nil
}

func (r *RedisStore) DeleteAll(ctx context.Context) error {
	pattern := fmt.Sprintf("%s:*", r.Prefix)
	iter := r.Client.Scan(ctx, 0, pattern, 0).Iterator()

	for iter.Next(ctx) {
		if err := r.Client.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("failed to delete key %s: %w", iter.Val(), err)
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	return nil
}