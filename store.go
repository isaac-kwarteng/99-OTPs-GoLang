package main

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const otpTTL = 2 * time.Minute

type OTPStore interface {
	Save(ctx context.Context, phoneNumber, otp string, ttl time.Duration) error
	ConsumeIfMatch(ctx context.Context, phoneNumber, otp string) (bool, error)
}

type RedisOTPStore struct {
	client *redis.Client
}

func NewRedisOTPStore(redisURL string) (*RedisOTPStore, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return &RedisOTPStore{client: redis.NewClient(options)}, nil
}

func (store *RedisOTPStore) Ping(ctx context.Context) error {
	return store.client.Ping(ctx).Err()
}

func (store *RedisOTPStore) Close() error {
	return store.client.Close()
}

func (store *RedisOTPStore) Save(ctx context.Context, phoneNumber, otp string, ttl time.Duration) error {
	return store.client.Set(ctx, "otp:"+phoneNumber, otp, ttl).Err()
}

func (store *RedisOTPStore) ConsumeIfMatch(ctx context.Context, phoneNumber, otp string) (bool, error) {
	result, err := store.client.Eval(ctx, `
		local stored = redis.call('GET', KEYS[1])
		if stored == ARGV[1] then
			redis.call('DEL', KEYS[1])
			return 1
		end
		return 0
	`, []string{"otp:" + phoneNumber}, otp).Int()
	return result == 1, err
}