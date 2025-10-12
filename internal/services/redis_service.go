package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"
	"verification-api/internal/config"

	"github.com/redis/go-redis/v9"
)

// RedisService provides Redis operations
type RedisService struct {
	client *redis.Client
}

// NewRedisService creates a new Redis service instance
func NewRedisService() (*RedisService, error) {
	opt, err := redis.ParseURL(config.AppConfig.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisService{client: client}, nil
}

// GenerateCode generates a 6-digit verification code
func (r *RedisService) GenerateCode() (string, error) {
	bytes := make([]byte, 3)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Generate 6-digit verification code
	code := (int(bytes[0])<<16 | int(bytes[1])<<8 | int(bytes[2])) % 1000000
	return fmt.Sprintf("%06d", code), nil
}

// StoreCode stores verification code (supports multi-project)
func (r *RedisService) StoreCode(projectID, email, code string, expireMinutes int) error {
	ctx := context.Background()
	key := fmt.Sprintf("verification_code:%s:%s", projectID, email)

	data := map[string]interface{}{
		"code":       code,
		"project_id": projectID,
		"created_at": time.Now().Unix(),
	}

	expire := time.Duration(expireMinutes) * time.Minute
	if err := r.client.HSet(ctx, key, data).Err(); err != nil {
		return err
	}
	return r.client.Expire(ctx, key, expire).Err()
}

// GetCode gets verification code (supports multi-project)
func (r *RedisService) GetCode(projectID, email string) (string, error) {
	ctx := context.Background()
	key := fmt.Sprintf("verification_code:%s:%s", projectID, email)

	code, err := r.client.HGet(ctx, key, "code").Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("verification code not found or expired")
		}
		return "", err
	}

	return code, nil
}

// DeleteCode deletes verification code (supports multi-project)
func (r *RedisService) DeleteCode(projectID, email string) error {
	ctx := context.Background()
	key := fmt.Sprintf("verification_code:%s:%s", projectID, email)
	return r.client.Del(ctx, key).Err()
}

// SetRateLimit sets rate limit (supports multi-project)
func (r *RedisService) SetRateLimit(projectID, email string, limitMinutes int) error {
	ctx := context.Background()
	key := fmt.Sprintf("rate_limit:%s:%s", projectID, email)
	expire := time.Duration(limitMinutes) * time.Minute
	return r.client.Set(ctx, key, "1", expire).Err()
}

// CheckRateLimit checks rate limit (supports multi-project)
func (r *RedisService) CheckRateLimit(projectID, email string) (bool, error) {
	ctx := context.Background()
	key := fmt.Sprintf("rate_limit:%s:%s", projectID, email)

	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return exists > 0, nil
}
