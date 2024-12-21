package models

import (
	"context"
	"fmt"
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

const (
	channelHashPrefix = "channel:"      // For storing full channel data
	nameIndexPrefix   = "channel:name:" // For name->id lookup
)

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

func (s *RedisStore) Save(ctx context.Context, tvChannel TvChannel) error {
	// Create name index
	channelKey := fmt.Sprintf("%s:%s", s.Prefix, tvChannel.ID)
	_, err := s.Client.HSet(ctx, channelKey, map[string]interface{}{
		"id":   tvChannel.ID,
		"name": tvChannel.Name,
		"url":  tvChannel.URL,
	}).Result()
	if err != nil {
		return err
	}
	nameKey := fmt.Sprintf("%s:name:%s", s.Prefix, tvChannel.Name)
	return s.Client.Set(ctx, nameKey, tvChannel.ID, 0).Err()
}

// GetChannelByID retrieves channel data by ID
func (s *RedisStore) GetChannelByID(ctx context.Context, id string) (*TvChannel, error) {
	channelKey := fmt.Sprintf("%s:%s", s.Prefix, id)

	data, err := s.Client.HGetAll(ctx, channelKey).Result()
	if err != nil {
		return nil, err
	}

	return &TvChannel{
		ID:   data["id"],
		Name: data["name"],
		URL:  data["url"],
	}, nil
}

// GetChannelIDByName retrieves channel ID by name
func (s *RedisStore) GetChannelIDByName(ctx context.Context, name string) (string, error) {
	return s.Client.Get(ctx, nameIndexPrefix+name).Result()
}

func (s *RedisStore) FlushChannels(ctx context.Context, prefix string) error {
	keys, err := s.Client.Keys(ctx, fmt.Sprintf("%s*", prefix)).Result()
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return s.Client.Del(ctx, keys...).Err()
}
