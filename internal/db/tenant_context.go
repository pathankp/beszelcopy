package db

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type contextKey string

const (
	tenantContextKey contextKey = "tenant_id"
	schemaContextKey contextKey = "schema_name"
)

// TenantContext holds tenant information for a request
type TenantContext struct {
	TenantID   string
	SchemaName string
}

// WithTenantContext adds tenant information to the context
func WithTenantContext(ctx context.Context, tenantID, schemaName string) context.Context {
	ctx = context.WithValue(ctx, tenantContextKey, tenantID)
	ctx = context.WithValue(ctx, schemaContextKey, schemaName)
	return ctx
}

// GetTenantFromContext retrieves tenant information from the context
func GetTenantFromContext(ctx context.Context) (*TenantContext, error) {
	tenantID, ok := ctx.Value(tenantContextKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant ID not found in context")
	}

	schemaName, ok := ctx.Value(schemaContextKey).(string)
	if !ok || schemaName == "" {
		return nil, fmt.Errorf("schema name not found in context")
	}

	return &TenantContext{
		TenantID:   tenantID,
		SchemaName: schemaName,
	}, nil
}

// WithTenantSchema returns a new GORM DB instance with the tenant's schema set
func WithTenantSchema(db *gorm.DB, ctx context.Context) (*gorm.DB, error) {
	tenant, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Set the search_path to the tenant's schema
	return db.Exec(fmt.Sprintf("SET search_path TO %s", tenant.SchemaName)), nil
}

// GetTenantDB returns a GORM DB instance configured for the tenant's schema
func GetTenantDB(ctx context.Context) (*gorm.DB, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	tenant, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Create a new session with the tenant's schema
	return DB.Exec(fmt.Sprintf("SET search_path TO %s", tenant.SchemaName)), nil
}

// FindTenantByID retrieves a tenant by ID
func FindTenantByID(db *gorm.DB, tenantID string) (*Tenant, error) {
	var tenant Tenant
	if err := db.Where("id = ? AND active = ?", tenantID, true).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

// FindTenantByName retrieves a tenant by name
func FindTenantByName(db *gorm.DB, name string) (*Tenant, error) {
	var tenant Tenant
	if err := db.Where("name = ? AND active = ?", name, true).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

// CreateTenant creates a new tenant and its schema
func CreateTenant(db *gorm.DB, name, schemaName string) (*Tenant, error) {
	tenant := &Tenant{
		Name:       name,
		SchemaName: schemaName,
		Active:     true,
	}

	// Create tenant record
	if err := db.Create(tenant).Error; err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	// Create tenant schema
	if err := CreateTenantSchema(db, schemaName); err != nil {
		// Rollback tenant creation if schema creation fails
		db.Delete(tenant)
		return nil, err
	}

	return tenant, nil
}
