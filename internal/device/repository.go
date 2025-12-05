package device

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	deviceIDsKey        = "device:ids"
	deviceIDKeyPrefix   = "device:id:"
	deviceAddrKeyPrefix = "device:addr:"
)

type Repository interface {
	List(ctx context.Context) ([]*Device, error)
	GetByID(ctx context.Context, id string) (*Device, error)
	Save(ctx context.Context, d *Device) error
	DeleteByID(ctx context.Context, id string) error
}

type RedisRepository struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{
		client: client,
	}
}

func (r *RedisRepository) idKey(id string) string {
	return deviceIDKeyPrefix + id
}

func (r *RedisRepository) addrKey(address string) string {
	return deviceAddrKeyPrefix + address
}

func (r *RedisRepository) List(ctx context.Context) ([]*Device, error) {
	ids, err := r.client.SMembers(ctx, deviceIDsKey).Result()
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []*Device{}, nil
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = r.idKey(id)
	}

	values, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	res := make([]*Device, 0, len(values))
	for _, v := range values {
		if v == nil {
			continue
		}
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("device: invalid value type")
		}
		var d Device
		if err := json.Unmarshal([]byte(s), &d); err != nil {
			return nil, err
		}
		res = append(res, &d)
	}
	return res, nil
}

func (r *RedisRepository) GetByID(ctx context.Context, id string) (*Device, error) {
	if id == "" {
		return nil, fmt.Errorf("device: empty id")
	}

	key := r.idKey(id)
	s, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var d Device
	if err := json.Unmarshal([]byte(s), &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *RedisRepository) Save(ctx context.Context, d *Device) error {
	if d == nil {
		return fmt.Errorf("device: nil device")
	}
	if d.IntervalSec <= 0 {
		return fmt.Errorf("device: invalid interval_sec")
	}
	if d.ID == "" {
		d.ID = uuid.NewString()
	}

	key := r.idKey(d.ID)

	var old Device
	oldAddress := ""
	existing, err := r.client.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	if err == nil {
		if err := json.Unmarshal([]byte(existing), &old); err != nil {
			return err
		}
		oldAddress = old.Address
	}

	b, err := json.Marshal(d)
	if err != nil {
		return err
	}

	pipe := r.client.TxPipeline()

	pipe.Set(ctx, key, b, 0)
	pipe.SAdd(ctx, deviceIDsKey, d.ID)

	if d.Address != "" {
		pipe.SAdd(ctx, r.addrKey(d.Address), d.ID)
	}

	if oldAddress != "" && oldAddress != d.Address {
		pipe.SRem(ctx, r.addrKey(oldAddress), d.ID)
	}

	_, err = pipe.Exec(ctx)
	return err
}

func (r *RedisRepository) DeleteByID(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("device: empty id")
	}

	key := r.idKey(id)
	s, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}

	var d Device
	if err := json.Unmarshal([]byte(s), &d); err != nil {
		return err
	}

	pipe := r.client.TxPipeline()

	pipe.Del(ctx, key)
	pipe.SRem(ctx, deviceIDsKey, id)
	if d.Address != "" {
		pipe.SRem(ctx, r.addrKey(d.Address), id)
	}

	_, err = pipe.Exec(ctx)
	return err
}
