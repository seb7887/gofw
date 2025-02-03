package sietch

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-redis/redis/v8"
	"time"
)

type RedisConnector[T any, ID comparable] struct {
	client     *redis.Client
	defaultTTL time.Duration
	getID      func(*T) ID
	keyFunc    func(ID) string
}

func NewRedisConnector[T any, ID comparable](client *redis.Client, defaultTTL time.Duration, getID func(*T) ID, keyFunc func(ID) string) *RedisConnector[T, ID] {
	return &RedisConnector[T, ID]{client, defaultTTL, getID, keyFunc}
}

func (r *RedisConnector[T, ID]) Create(ctx context.Context, item *T) error {
	key := r.keyFunc(r.getID(item))
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, string(data), r.defaultTTL).Err()
}

func (r *RedisConnector[T, ID]) Get(ctx context.Context, id ID) (*T, error) {
	key := r.keyFunc(id)
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrItemNotFound
		}
		return nil, err
	}

	var item T
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *RedisConnector[T, ID]) BatchCreate(ctx context.Context, items []T) error {
	pipe := r.client.Pipeline()
	for _, item := range items {
		key := r.keyFunc(r.getID(&item))
		data, err := json.Marshal(item)
		if err != nil {
			return err
		}
		pipe.Set(ctx, key, data, r.defaultTTL)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisConnector[T, ID]) Query(_ context.Context, _ *Filter) ([]T, error) {
	return nil, ErrUnsupportedOperation
}

func (r *RedisConnector[T, ID]) Update(ctx context.Context, item *T) error {
	return r.Create(ctx, item)
}

func (r *RedisConnector[T, ID]) BatchUpdate(ctx context.Context, items []T) error {
	return r.BatchCreate(ctx, items)
}

func (r *RedisConnector[T, ID]) Delete(ctx context.Context, id ID) error {
	key := r.keyFunc(id)
	return r.client.Del(ctx, key).Err()
}

func (r *RedisConnector[T, ID]) BatchDelete(ctx context.Context, items []ID) error {
	pipe := r.client.Pipeline()
	for _, item := range items {
		key := r.keyFunc(item)
		pipe.Del(ctx, key)
	}
	_, err := pipe.Exec(ctx)
	return err
}
