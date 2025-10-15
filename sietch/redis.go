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
	if item == nil {
		return errors.New("item cannot be nil")
	}
	key := r.keyFunc(r.getID(item))
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, r.defaultTTL).Err()
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
	if len(items) == 0 {
		return nil
	}
	
	// Preparar todos los datos primero
	var commands []struct {
		key  string
		data []byte
	}
	
	for _, item := range items {
		key := r.keyFunc(r.getID(&item))
		data, err := json.Marshal(item)
		if err != nil {
			return err
		}
		commands = append(commands, struct {
			key  string
			data []byte
		}{key, data})
	}
	
	// Ahora ejecutar todas las operaciones
	pipe := r.client.Pipeline()
	for _, cmd := range commands {
		pipe.Set(ctx, cmd.key, cmd.data, r.defaultTTL)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisConnector[T, ID]) Query(_ context.Context, _ *Filter) ([]T, error) {
	return nil, ErrUnsupportedOperation
}

func (r *RedisConnector[T, ID]) Update(ctx context.Context, item *T) error {
	if item == nil {
		return errors.New("item cannot be nil")
	}
	return r.Create(ctx, item)
}

func (r *RedisConnector[T, ID]) BatchUpdate(ctx context.Context, items []T) error {
	if len(items) == 0 {
		return nil
	}
	return r.BatchCreate(ctx, items)
}

func (r *RedisConnector[T, ID]) Delete(ctx context.Context, id ID) error {
	key := r.keyFunc(id)
	result, err := r.client.Del(ctx, key).Result()
	if err != nil {
		return err
	}
	if result == 0 {
		return ErrItemNotFound
	}
	return nil
}

func (r *RedisConnector[T, ID]) BatchDelete(ctx context.Context, items []ID) error {
	if len(items) == 0 {
		return nil
	}
	pipe := r.client.Pipeline()
	for _, item := range items {
		key := r.keyFunc(item)
		pipe.Del(ctx, key)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// Count is not supported by Redis connector
func (r *RedisConnector[T, ID]) Count(_ context.Context, _ *Filter) (int64, error) {
	return 0, ErrUnsupportedOperation
}

// WithTx is not supported by Redis connector
func (r *RedisConnector[T, ID]) WithTx(_ context.Context, _ TxFunc[T, ID]) error {
	return ErrUnsupportedOperation
}

// Exists checks if an entity with the given ID exists in Redis
func (r *RedisConnector[T, ID]) Exists(ctx context.Context, id ID) (bool, error) {
	key := r.keyFunc(id)
	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// Upsert creates a new entity or updates an existing one in Redis
// For Redis, this is the same as Create/Update since SET always upserts
func (r *RedisConnector[T, ID]) Upsert(ctx context.Context, item *T) error {
	return r.Create(ctx, item)
}

// BatchUpsert creates or updates multiple entities in Redis
// For Redis, this is the same as BatchCreate since SET always upserts
func (r *RedisConnector[T, ID]) BatchUpsert(ctx context.Context, items []T) error {
	return r.BatchCreate(ctx, items)
}
