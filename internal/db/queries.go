package db

import (
	"fmt"

	"gorm.io/gorm"
)

// ListTenants returns all active tenants
func ListTenants(db *gorm.DB) ([]Tenant, error) {
	var tenants []Tenant
	if err := db.Where("active = ?", true).Find(&tenants).Error; err != nil {
		return nil, err
	}
	return tenants, nil
}

// FindAccountByEmail retrieves an account by email
func FindAccountByEmail(db *gorm.DB, email string) (*Account, error) {
	var account Account
	if err := db.Where("email = ? AND active = ?", email, true).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

// FindAccountByID retrieves an account by ID
func FindAccountByID(db *gorm.DB, accountID string) (*Account, error) {
	var account Account
	if err := db.Where("id = ? AND active = ?", accountID, true).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

// CreateAccount creates a new user account
func CreateAccount(db *gorm.DB, email, password, name, tenantID string) (*Account, error) {
	account := &Account{
		Email:    email,
		Password: password,
		Name:     name,
		TenantID: tenantID,
		Active:   true,
	}

	if err := db.Create(account).Error; err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return account, nil
}

// UpdateAccount updates an account
func UpdateAccount(db *gorm.DB, account *Account) error {
	return db.Save(account).Error
}

// DeactivateAccount deactivates an account (soft delete)
func DeactivateAccount(db *gorm.DB, accountID string) error {
	return db.Model(&Account{}).Where("id = ?", accountID).Update("active", false).Error
}

// CreateAuditLog creates a new audit log entry
func CreateAuditLog(db *gorm.DB, tenantID, accountID, action, resource, details, ipAddress string) error {
	log := &AuditLog{
		TenantID:  tenantID,
		AccountID: accountID,
		Action:    action,
		Resource:  resource,
		Details:   details,
		IPAddress: ipAddress,
	}

	return db.Create(log).Error
}

// FindSubscriptionByTenantID retrieves a subscription by tenant ID
func FindSubscriptionByTenantID(db *gorm.DB, tenantID string) (*Subscription, error) {
	var subscription Subscription
	if err := db.Where("tenant_id = ?", tenantID).First(&subscription).Error; err != nil {
		return nil, err
	}
	return &subscription, nil
}

// CreateSubscription creates a new subscription
func CreateSubscription(db *gorm.DB, tenantID, plan string) (*Subscription, error) {
	subscription := &Subscription{
		TenantID: tenantID,
		Plan:     plan,
		Status:   "active",
	}

	if err := db.Create(subscription).Error; err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	return subscription, nil
}

// UpdateSubscription updates a subscription
func UpdateSubscription(db *gorm.DB, subscription *Subscription) error {
	return db.Save(subscription).Error
}
