package health

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const healthKeyPrefix = "health:"

type Repository interface {
	Get(ctx context.Context, deviceID string) (*HealthStatus, error)
	Save(ctx context.Context, h *HealthStatus, ttl time.Duration) error
	Delete(ctx context.Context, deviceID string) error
}

type RedisRepository struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{
		client: client,
	}
}

func (r *RedisRepository) key(deviceID string) string {
	return healthKeyPrefix + deviceID
}

func (r *RedisRepository) Get(ctx context.Context, deviceID string) (*HealthStatus, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("health: empty device id")
	}

	key := r.key(deviceID)
	s, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var h HealthStatus
	if err := json.Unmarshal([]byte(s), &h); err != nil {
		return nil, err
	}
	return &h, nil
}

func (r *RedisRepository) Save(ctx context.Context, h *HealthStatus, ttl time.Duration) error {
	if h == nil {
		return fmt.Errorf("health: nil status")
	}
	if h.DeviceID == "" {
		return fmt.Errorf("health: empty device id")
	}
	if h.Data == nil {
		h.Data = make(map[string]interface{})
	}
	if h.LastCheck.IsZero() {
		h.LastCheck = time.Now()
	}

	b, err := json.Marshal(h)
	if err != nil {
		return err
	}

	key := r.key(h.DeviceID)
	if ttl > 0 {
		return r.client.Set(ctx, key, b, ttl).Err()
	}
	return r.client.Set(ctx, key, b, 0).Err()
}

func (r *RedisRepository) Delete(ctx context.Context, deviceID string) error {
	if deviceID == "" {
		return fmt.Errorf("health: empty device id")
	}
	key := r.key(deviceID)
	return r.client.Del(ctx, key).Err()
}

