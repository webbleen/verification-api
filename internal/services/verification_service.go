package services

import (
	"verification-api/internal/database"

	"gorm.io/gorm"
)

// VerificationService provides verification code operations
type VerificationService struct {
	db *gorm.DB
}

// NewVerificationService creates a new verification service
func NewVerificationService() *VerificationService {
	return &VerificationService{
		db: database.GetDB(),
	}
}

// VerificationService is kept for potential future use
// All verification code operations now use Redis directly
