package db

import (
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

// Tenant represents a tenant in the multi-tenant system
type Tenant struct {
	ID             string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name           string `gorm:"not null;uniqueIndex"`
	SchemaName     string `gorm:"not null;uniqueIndex"`
	Active         bool   `gorm:"default:true"`
	CreatedAt      int64  `gorm:"autoCreateTime"`
	UpdatedAt      int64  `gorm:"autoUpdateTime"`
}

// Account represents a user account in the public schema
type Account struct {
	ID        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Email     string `gorm:"not null;uniqueIndex"`
	Password  string `gorm:"not null"`
	Name      string
	Role      string `gorm:"default:'user'"`
	TenantID  string `gorm:"type:uuid;index"`
	Active    bool   `gorm:"default:true"`
	CreatedAt int64  `gorm:"autoCreateTime"`
	UpdatedAt int64  `gorm:"autoUpdateTime"`
}

// Subscription represents a tenant subscription
type Subscription struct {
	ID         string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID   string `gorm:"type:uuid;index;not null"`
	Plan       string `gorm:"default:'free'"`
	Status     string `gorm:"default:'active'"`
	StartDate  int64
	EndDate    int64
	CreatedAt  int64 `gorm:"autoCreateTime"`
	UpdatedAt  int64 `gorm:"autoUpdateTime"`
}

// AuditLog represents an audit log entry in the public schema
type AuditLog struct {
	ID        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID  string `gorm:"type:uuid;index"`
	AccountID string `gorm:"type:uuid;index"`
	Action    string `gorm:"not null"`
	Resource  string
	Details   string `gorm:"type:text"`
	IPAddress string
	CreatedAt int64 `gorm:"autoCreateTime"`
}

// RunMigrations runs all database migrations
func RunMigrations(db *gorm.DB) error {
	slog.Info("Running database migrations...")

	// Create public schema tables
	if err := createPublicSchemaTables(db); err != nil {
		return fmt.Errorf("failed to create public schema tables: %w", err)
	}

	slog.Info("Database migrations completed successfully")
	return nil
}

// createPublicSchemaTables creates the core tables in the public schema
func createPublicSchemaTables(db *gorm.DB) error {
	// Enable UUID extension if not already enabled
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		return fmt.Errorf("failed to enable uuid-ossp extension: %w", err)
	}

	// Auto-migrate public schema tables
	tables := []interface{}{
		&Tenant{},
		&Account{},
		&Subscription{},
		&AuditLog{},
	}

	for _, table := range tables {
		if err := db.AutoMigrate(table); err != nil {
			return fmt.Errorf("failed to migrate table: %w", err)
		}
	}

	slog.Info("Public schema tables created successfully")
	return nil
}

// CreateTenantSchema creates a new tenant-specific schema with all required tables
func CreateTenantSchema(db *gorm.DB, schemaName string) error {
	// Create schema
	if err := db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schemaName)).Error; err != nil {
		return fmt.Errorf("failed to create schema %s: %w", schemaName, err)
	}

	// Create tenant-specific tables
	// Note: In Phase 0.1, we're just creating the framework
	// The actual PocketBase collections will be migrated later
	
	// For now, create a basic systems table as a placeholder
	createSystemsTable := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.systems (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			host VARCHAR(255),
			port INTEGER,
			created_at BIGINT,
			updated_at BIGINT
		)
	`, schemaName)

	if err := db.Exec(createSystemsTable).Error; err != nil {
		return fmt.Errorf("failed to create systems table in schema %s: %w", schemaName, err)
	}

	slog.Info("Tenant schema created", "schema", schemaName)
	return nil
}

// DropTenantSchema drops a tenant-specific schema
func DropTenantSchema(db *gorm.DB, schemaName string) error {
	if err := db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName)).Error; err != nil {
		return fmt.Errorf("failed to drop schema %s: %w", schemaName, err)
	}
	slog.Info("Tenant schema dropped", "schema", schemaName)
	return nil
}
