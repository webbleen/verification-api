package database

import (
	"auth-mail/internal/config"
	"auth-mail/internal/models"
	"auth-mail/pkg/logging"
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var (
	DB          *gorm.DB
	RedisClient *redis.Client
)

// InitDatabase initializes database connection
func InitDatabase() error {
	// Initialize PostgreSQL
	if err := initPostgres(); err != nil {
		return fmt.Errorf("failed to initialize PostgreSQL: %w", err)
	}

	// Initialize Redis
	if err := initRedis(); err != nil {
		return fmt.Errorf("failed to initialize Redis: %w", err)
	}

	// Auto migrate tables
	if err := autoMigrate(); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// Insert default data
	if err := insertDefaultData(); err != nil {
		return fmt.Errorf("failed to insert default data: %w", err)
	}

	return nil
}

// initPostgres initializes PostgreSQL connection
func initPostgres() error {
	var err error
	var dsn string

	// Get database URL from environment
	if dsn = config.AppConfig.DatabaseURL; dsn == "" {
		// Fallback to SQLite for development
		logging.Infof("Database URL not set, using SQLite for development")
		DB, err = gorm.Open(sqlite.Open("auth-mail.db"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
			NamingStrategy: schema.NamingStrategy{
				SingularTable: true,
			},
		})
	} else {
		// Use PostgreSQL for production
		DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
			NamingStrategy: schema.NamingStrategy{
				SingularTable: true,
			},
		})
	}

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	logging.Infof("Database connected successfully")
	return nil
}

// initRedis initializes Redis connection
func initRedis() error {
	opt, err := redis.ParseURL(config.AppConfig.RedisURL)
	if err != nil {
		return fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	RedisClient = redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = RedisClient.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logging.Infof("Redis connected successfully")
	return nil
}

// autoMigrate performs database migration
func autoMigrate() error {
	return DB.AutoMigrate(
		&models.Project{},
		&models.VerificationCode{},
		&models.VerificationLog{},
		&models.RateLimit{},
	)
}

// insertDefaultData inserts default data
func insertDefaultData() error {
	// Insert default project
	defaultProject := models.Project{
		ProjectID:   "default",
		ProjectName: "Default Project",
		APIKey:      "default-api-key",
		FromName:    config.AppConfig.BrevoFromName,
		IsActive:    true,
		Description: "Default project for testing and development",
		RateLimit:   60,   // 60 requests per hour
		MaxRequests: 1000, // 1000 requests per day
	}

	// Use FirstOrCreate to avoid duplicates
	result := DB.Where("project_id = ?", "default").FirstOrCreate(&defaultProject)
	if result.Error != nil {
		return fmt.Errorf("failed to create default project: %w", result.Error)
	}

	logging.Infof("Default data inserted successfully")
	return nil
}

// GetDB returns database instance
func GetDB() *gorm.DB {
	return DB
}

// GetRedis returns Redis client
func GetRedis() *redis.Client {
	return RedisClient
}

// CloseDatabase closes database connections
func CloseDatabase() error {
	// Close PostgreSQL
	if sqlDB, err := DB.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			logging.Errorf("Failed to close database: %v", err)
		}
	}

	// Close Redis
	if RedisClient != nil {
		if err := RedisClient.Close(); err != nil {
			logging.Errorf("Failed to close Redis: %v", err)
		}
	}

	return nil
}

// SetCache sets cache with expiration
func SetCache(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return RedisClient.Set(ctx, key, value, expiration).Err()
}

// GetCache gets cache value
func GetCache(ctx context.Context, key string) (string, error) {
	return RedisClient.Get(ctx, key).Result()
}

// DeleteCache deletes cache
func DeleteCache(ctx context.Context, key string) error {
	return RedisClient.Del(ctx, key).Err()
}
