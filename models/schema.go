package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system.
type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// Transaction represents a financial transaction.
type Transaction struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	MobileID        string    `gorm:"uniqueIndex;not null" json:"mobile_id"` // Unique ID from SMS database
	RawText         string    `json:"raw_text"`
	Amount          float64   `json:"amount"`
	Merchant        string    `json:"merchant"`
	Category        string    `json:"category"`
	IsManual        bool      `gorm:"default:false" json:"is_manual"`
	TransactionDate time.Time `json:"transaction_date"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CategoryRule represents a rule for categorizing transactions.
type CategoryRule struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Pattern        string    `gorm:"not null" json:"pattern"` // Keyword to look for (case-insensitive)
	TargetCategory string    `gorm:"not null" json:"target_category"`
	TargetMerchant string    `gorm:"not null" json:"target_merchant"`
	CreatedAt      time.Time `json:"created_at"`
}
