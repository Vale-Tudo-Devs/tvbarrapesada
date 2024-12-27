package models

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"

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

// Save stores a TvChannel object in Redis.
// If the operation fails, it returns an error, otherwise it returns nil.
// The context parameter can be used to control timeout and cancellation.
func (r *RedisStore) Save(ctx context.Context, tvChannel TvChannel) error {
	// Set a hash with channel information
	channelKey := fmt.Sprintf("%s:%s", r.Prefix, tvChannel.ID)
	_, err := r.Client.HSet(ctx, channelKey, map[string]interface{}{
		"id":   tvChannel.ID,
		"name": strings.ToUpper(tvChannel.Name),
		"url":  tvChannel.URL,
	}).Result()
	if err != nil {
		return err
	}

	// Increase counter
	if err := r.Client.Incr(ctx, fmt.Sprintf("%s:counter", r.Prefix)).Err(); err != nil {
		return err
	}

	// Set indexes for id, name and URL
	idKey := fmt.Sprintf("%s:id:%s", r.Prefix, tvChannel.ID)
	if err := r.Client.Set(ctx, idKey, tvChannel.ID, 0).Err(); err != nil {
		return err
	}

	nameKey := fmt.Sprintf("%s:name:%s", r.Prefix, strings.ToUpper(tvChannel.Name))
	if err := r.Client.Set(ctx, nameKey, tvChannel.ID, 0).Err(); err != nil {
		return err
	}

	urlKey := fmt.Sprintf("%s:url:%s", r.Prefix, tvChannel.URL)
	if err := r.Client.Set(ctx, urlKey, tvChannel.ID, 0).Err(); err != nil {
		return err
	}

	log.Printf("Saved channel %s", strings.ToUpper(tvChannel.Name))
	return nil
}

// GetChannelByID retrieves a TV channel from Redis by its ID.
// It takes a context.Context and a channel ID as parameters.
// Returns a pointer to TvChannel if found, or an error if the operation fails.
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

	log.Printf("Retrieved channel %s", channel.Name)

	return channel, nil
}

// DeleteAll removes all entries from the Redis store. This operation clears all key-value pairs
// stored in the Redis database associated with this store instance.
// It requires a context for cancellation and timeout control.
// Returns an error if the operation fails, nil otherwise.
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

// GetChannelCounter retrieves the current counter value from Redis.
// The counter is stored with a key formatted as "{prefix}:counter".
// If the counter doesn't exist in Redis, it returns 0 without error.
// Returns the counter value and any error encountered during the operation.
func (r *RedisStore) GetChannelCounter(ctx context.Context) (int64, error) {
	counterKey := fmt.Sprintf("%s:counter", r.Prefix)
	count, err := r.Client.Get(ctx, counterKey).Int64()
	if err != nil {
		if err == redis.Nil {
			return 0, nil // Counter doesn't exist yet
		}
		return 0, fmt.Errorf("failed to get counter: %w", err)
	}
	return count, nil
}

// SearchChannelsByName searches for TV channels whose names match the given search pattern.
// The search is case-sensitive and uses Redis pattern matching.
// Returns a slice of TvChannel objects and any error encountered.
func (r *RedisStore) SearchChannelsByName(ctx context.Context, searchTerm string) ([]TvChannel, error) {
	// Split the search term by spaces and join with *
	searchTerm = strings.Join(strings.Fields(searchTerm), "*")
	searchTermUpper := strings.ToUpper(searchTerm)
	pattern := fmt.Sprintf("%s:name:*%s*", r.Prefix, searchTermUpper)
	var channels []TvChannel

	iter := r.Client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		nameKey := iter.Val()
		channelID, err := r.Client.Get(ctx, nameKey).Result()
		if err != nil {
			continue
		}

		channelKey := fmt.Sprintf("%s:%s", r.Prefix, channelID)
		data, err := r.Client.HGetAll(ctx, channelKey).Result()
		if err != nil {
			continue
		}

		if len(data) > 0 {
			channels = append(channels, TvChannel{
				ID:   data["id"],
				Name: data["name"],
				URL:  data["url"],
			})
		}
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	return channels, nil
}

// GetRandomChannel retrieves a random channel ID from the Redis store.
// It first gets the total channel count using GetChannelCounter and then
// generates a random number between 0 and count-1.
//
// The context parameter is used for cancellation and timeout control.
//
// Returns:
//   - int64: A random channel ID between 0 and (channel count - 1)
//   - error: An error if the channel counter could not be retrieved
func (r *RedisStore) GetRandomChannel(ctx context.Context) (int64, error) {
	count, err := r.GetChannelCounter(ctx)
	if err != nil {
		return 0, err
	}
	id := rand.Int63n(count)

	return id, nil
}

func (r *RedisStore) RegisterCurrentChannel(ctx context.Context, tvChannel *TvChannel) error {
	id := tvChannel.ID
	key := fmt.Sprintf("%s:current", r.Prefix)
	return r.Client.Set(ctx, key, id, 0).Err()
}

func (r *RedisStore) GetCurrentChannel(ctx context.Context) (*TvChannel, error) {
	key := fmt.Sprintf("%s:current", r.Prefix)
	idStr, err := r.Client.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, err
	}

	channel, err := r.GetChannelByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return channel, nil
}

func (r *RedisStore) GetAllChannels(ctx context.Context) (string, error) {
	pattern := fmt.Sprintf("%s:[0-9]*", r.Prefix)
	var channels []*TvChannel

	iter := r.Client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		data, err := r.Client.HGetAll(ctx, iter.Val()).Result()
		if err != nil {
			continue
		}

		if len(data) > 0 {
			channel := &TvChannel{
				ID:   data["id"],
				Name: data["name"],
			}
			channels = append(channels, channel)
		}
	}

	if err := iter.Err(); err != nil {
		return "", fmt.Errorf("scan failed: %w", err)
	}

	channelsCsv := channels2Csv(channels)
	fileName := fmt.Sprintf("/data/%s-catalog.csv", r.Prefix)
	err := os.WriteFile(fileName, channelsCsv, 0644)
	if err != nil {
		fmt.Printf("Error writing channels to file: %v\n", err)
	}
	return fileName, nil

}

func channels2Csv(channels []*TvChannel) []byte {
	var sb strings.Builder
	sb.WriteString("ID,Name\n")
	for _, channel := range channels {
		sb.WriteString(fmt.Sprintf("%s,%s\n", channel.ID, channel.Name))
	}
	return []byte(sb.String())
}
