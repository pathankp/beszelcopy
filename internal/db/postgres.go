// Package db provides PostgreSQL database connection management and multi-tenancy support
package db

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	*gorm.DB
}

var DB *Database

// InitPostgreSQL initializes the PostgreSQL database connection
func InitPostgreSQL() (*Database, error) {
	dsn := getPostgresDSN()
	if dsn == "" {
		return nil, fmt.Errorf("PostgreSQL DSN not configured")
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("PostgreSQL connection established")

	DB = &Database{DB: db}
	return DB, nil
}

// getPostgresDSN retrieves the PostgreSQL connection string from environment variables
func getPostgresDSN() string {
	// Check for SONAR_HUB_POSTGRES_DSN first
	if dsn := os.Getenv("SONAR_HUB_POSTGRES_DSN"); dsn != "" {
		return dsn
	}

	// Check for DATABASE_URL (common in cloud environments)
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}

	// Build DSN from individual components
	host := getEnv("SONAR_HUB_POSTGRES_HOST", "localhost")
	port := getEnv("SONAR_HUB_POSTGRES_PORT", "5432")
	user := getEnv("SONAR_HUB_POSTGRES_USER", "sonar")
	password := getEnv("SONAR_HUB_POSTGRES_PASSWORD", "")
	dbname := getEnv("SONAR_HUB_POSTGRES_DB", "sonar")
	sslmode := getEnv("SONAR_HUB_POSTGRES_SSLMODE", "disable")

	if password == "" {
		return ""
	}

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Close closes the database connection
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
