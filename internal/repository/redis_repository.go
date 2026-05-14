package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisRepository struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{client: client}
}

// ========== SESSION ==========

func (r *RedisRepository) SetSession(ctx context.Context, token, nasabahID string, ttl time.Duration) error {
	key := fmt.Sprintf("session:%s", token)
	return r.client.Set(ctx, key, nasabahID, ttl).Err()
}

func (r *RedisRepository) GetSession(ctx context.Context, token string) (string, error) {
	key := fmt.Sprintf("session:%s", token)
	return r.client.Get(ctx, key).Result()
}

func (r *RedisRepository) DeleteSession(ctx context.Context, token string) error {
	key := fmt.Sprintf("session:%s", token)
	return r.client.Del(ctx, key).Err()
}

func (r *RedisRepository) GetSessionTTL(ctx context.Context, token string) (time.Duration, error) {
	key := fmt.Sprintf("session:%s", token)
	return r.client.TTL(ctx, key).Result()
}

// ========== RATE LIMITER ==========

func (r *RedisRepository) IncrRateLimit(ctx context.Context, identifier string, window time.Duration) (int64, error) {
	key := fmt.Sprintf("ratelimit:%s", identifier)

	pipe := r.client.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

func (r *RedisRepository) GetRateLimit(ctx context.Context, identifier string) (int64, error) {
	key := fmt.Sprintf("ratelimit:%s", identifier)
	val, err := r.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

func (r *RedisRepository) GetRateLimitTTL(ctx context.Context, identifier string) (time.Duration, error) {
	key := fmt.Sprintf("ratelimit:%s", identifier)
	return r.client.TTL(ctx, key).Result()
}

// ========== CACHE SALDO (referensi ke PostgreSQL, disimpan sementara) ==========

func (r *RedisRepository) SetCacheSaldo(ctx context.Context, accountID string, saldo float64, ttl time.Duration) error {
	key := fmt.Sprintf("saldo:%s", accountID)
	return r.client.Set(ctx, key, saldo, ttl).Err()
}

func (r *RedisRepository) GetCacheSaldo(ctx context.Context, accountID string) (float64, error) {
	key := fmt.Sprintf("saldo:%s", accountID)
	return r.client.Get(ctx, key).Float64()
}

func (r *RedisRepository) DeleteCacheSaldo(ctx context.Context, accountID string) error {
	key := fmt.Sprintf("saldo:%s", accountID)
	return r.client.Del(ctx, key).Err()
}

// ========== BLACKLIST TOKEN (logout) ==========

func (r *RedisRepository) BlacklistToken(ctx context.Context, token string, ttl time.Duration) error {
	key := fmt.Sprintf("blacklist:%s", token)
	return r.client.Set(ctx, key, "1", ttl).Err()
}

func (r *RedisRepository) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	key := fmt.Sprintf("blacklist:%s", token)
	val, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
}
