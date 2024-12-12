package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/wrale/wrale-signage/internal/wsignd/ratelimit"
)

// Store implements rate limit storage using Redis
type Store struct {
	client *redis.Client
}

// NewStore creates a new Redis-backed rate limit store
func NewStore(client *redis.Client) *Store {
	return &Store{client: client}
}

// keyStr converts a LimitKey to a Redis key
func (s *Store) keyStr(key ratelimit.LimitKey) string {
	return fmt.Sprintf("rate:%s:%s:%s:%s",
		key.Type,
		key.Token,
		key.RemoteIP,
		key.Endpoint,
	)
}

// Increment attempts to increment a counter and returns current count
func (s *Store) Increment(ctx context.Context, key ratelimit.LimitKey, limit ratelimit.Limit) (int, error) {
	redisKey := s.keyStr(key)

	pipe := s.client.Pipeline()

	// Get current value
	getCmd := pipe.Get(ctx, redisKey)

	// Increment
	incrCmd := pipe.Incr(ctx, redisKey)

	// Set expiry if new key
	pipe.Expire(ctx, redisKey, limit.Period)

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return 0, fmt.Errorf("%w: %v", ratelimit.ErrStoreError, err)
	}

	// Get results
	count := 1 // Default for new keys
	if val, err := getCmd.Result(); err == nil {
		count, _ = strconv.Atoi(val)
	}

	// Check limit
	if count > limit.Rate+limit.BurstSize {
		return count, ratelimit.ErrLimitExceeded
	}

	return count, nil
}

// Reset clears a rate limit counter
func (s *Store) Reset(ctx context.Context, key ratelimit.LimitKey) error {
	err := s.client.Del(ctx, s.keyStr(key)).Err()
	if err != nil {
		return fmt.Errorf("%w: %v", ratelimit.ErrStoreError, err)
	}
	return nil
}

// IsExceeded checks if a limit has been exceeded without incrementing
func (s *Store) IsExceeded(ctx context.Context, key ratelimit.LimitKey, limit ratelimit.Limit) (bool, error) {
	val, err := s.client.Get(ctx, s.keyStr(key)).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("%w: %v", ratelimit.ErrStoreError, err)
	}

	count, _ := strconv.Atoi(val)
	return count > limit.Rate+limit.BurstSize, nil
}

// keyInfo is stored with metadata about the rate limit
type keyInfo struct {
	Type     string    `json:"type"`
	Created  time.Time `json:"created"`
	LastSeen time.Time `json:"last_seen"`
}

// storeKeyInfo stores metadata about a rate limit key
func (s *Store) storeKeyInfo(ctx context.Context, key ratelimit.LimitKey) error {
	info := keyInfo{
		Type:     key.Type,
		Created:  time.Now(),
		LastSeen: time.Now(),
	}

	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("%w: %v", ratelimit.ErrStoreError, err)
	}

	infoKey := fmt.Sprintf("rate:info:%s", s.keyStr(key))
	err = s.client.Set(ctx, infoKey, data, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("%w: %v", ratelimit.ErrStoreError, err)
	}

	return nil
}

// updateKeyInfo updates the last seen time for a key
func (s *Store) updateKeyInfo(ctx context.Context, key ratelimit.LimitKey) error {
	infoKey := fmt.Sprintf("rate:info:%s", s.keyStr(key))

	var info keyInfo
	data, err := s.client.Get(ctx, infoKey).Result()
	if err == redis.Nil {
		return s.storeKeyInfo(ctx, key)
	}
	if err != nil {
		return fmt.Errorf("%w: %v", ratelimit.ErrStoreError, err)
	}

	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return fmt.Errorf("%w: %v", ratelimit.ErrStoreError, err)
	}

	info.LastSeen = time.Now()
	data, err = json.Marshal(info)
	if err != nil {
		return fmt.Errorf("%w: %v", ratelimit.ErrStoreError, err)
	}

	err = s.client.Set(ctx, infoKey, data, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("%w: %v", ratelimit.ErrStoreError, err)
	}

	return nil
}
