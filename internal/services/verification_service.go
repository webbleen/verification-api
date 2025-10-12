package services

import (
	"fmt"
	"time"
	"verification-api/internal/database"
	"verification-api/internal/models"

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

// CreateVerificationCode creates a new verification code record
func (s *VerificationService) CreateVerificationCode(projectID, email, code string, expiresAt time.Time, ipAddress, userAgent string) error {
	verificationCode := &models.VerificationCode{
		ProjectID: projectID,
		Email:     email,
		Code:      code,
		IsUsed:    false,
		ExpiresAt: expiresAt,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	if err := s.db.Create(verificationCode).Error; err != nil {
		return fmt.Errorf("failed to create verification code: %w", err)
	}

	return nil
}

// GetVerificationCode gets verification code by project ID and email
func (s *VerificationService) GetVerificationCode(projectID, email string) (*models.VerificationCode, error) {
	var verificationCode models.VerificationCode
	result := s.db.Where("project_id = ? AND email = ? AND is_used = ? AND expires_at > ?",
		projectID, email, false, time.Now()).First(&verificationCode)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("verification code not found or expired")
		}
		return nil, result.Error
	}

	return &verificationCode, nil
}

// MarkCodeAsUsed marks verification code as used
func (s *VerificationService) MarkCodeAsUsed(codeID uint) error {
	now := time.Now()
	result := s.db.Model(&models.VerificationCode{}).
		Where("id = ?", codeID).
		Updates(map[string]interface{}{
			"is_used": true,
			"used_at": &now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to mark code as used: %w", result.Error)
	}

	return nil
}

// CleanupExpiredCodes removes expired verification codes
func (s *VerificationService) CleanupExpiredCodes() error {
	result := s.db.Where("expires_at < ? OR (is_used = ? AND used_at < ?)",
		time.Now(), true, time.Now().Add(-24*time.Hour)).Delete(&models.VerificationCode{})

	if result.Error != nil {
		return fmt.Errorf("failed to cleanup expired codes: %w", result.Error)
	}

	return nil
}

// LogVerificationAttempt logs verification attempt
func (s *VerificationService) LogVerificationAttempt(projectID, email, action string, success bool, ipAddress, userAgent, errorMsg string) error {
	log := &models.VerificationLog{
		ProjectID:   projectID,
		Email:       email,
		Action:      action,
		Success:     success,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		ErrorMsg:    errorMsg,
		RequestTime: time.Now(),
	}

	if err := s.db.Create(log).Error; err != nil {
		return fmt.Errorf("failed to log verification attempt: %w", err)
	}

	return nil
}

// GetVerificationStats gets verification statistics for a project
func (s *VerificationService) GetVerificationStats(projectID string, days int) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	startDate := time.Now().AddDate(0, 0, -days)

	// Total codes sent
	var totalCodesSent int64
	s.db.Model(&models.VerificationCode{}).
		Where("project_id = ? AND created_at >= ?", projectID, startDate).
		Count(&totalCodesSent)
	stats["total_codes_sent"] = totalCodesSent

	// Successful verifications
	var successfulVerifications int64
	s.db.Model(&models.VerificationLog{}).
		Where("project_id = ? AND action = ? AND success = ? AND created_at >= ?",
			projectID, "verify", true, startDate).
		Count(&successfulVerifications)
	stats["successful_verifications"] = successfulVerifications

	// Failed verifications
	var failedVerifications int64
	s.db.Model(&models.VerificationLog{}).
		Where("project_id = ? AND action = ? AND success = ? AND created_at >= ?",
			projectID, "verify", false, startDate).
		Count(&failedVerifications)
	stats["failed_verifications"] = failedVerifications

	// Success rate
	if totalCodesSent > 0 {
		stats["success_rate"] = float64(successfulVerifications) / float64(totalCodesSent) * 100
	} else {
		stats["success_rate"] = 0.0
	}

	// Daily breakdown
	var dailyStats []map[string]interface{}
	s.db.Raw(`
		SELECT 
			DATE(created_at) as date,
			COUNT(*) as codes_sent,
			SUM(CASE WHEN action = 'verify' AND success = true THEN 1 ELSE 0 END) as successful_verifications
		FROM verification_log 
		WHERE project_id = ? AND created_at >= ?
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`, projectID, startDate).Scan(&dailyStats)
	stats["daily_breakdown"] = dailyStats

	return stats, nil
}

// CheckRateLimit checks if rate limit is exceeded
func (s *VerificationService) CheckRateLimit(projectID, email, ipAddress string) (bool, error) {
	// Get project rate limit settings
	projectService := NewProjectService()
	project, err := projectService.GetProjectByID(projectID)
	if err != nil {
		return false, err
	}

	// Check email rate limit (per hour)
	var emailCount int64
	oneHourAgo := time.Now().Add(-time.Hour)
	s.db.Model(&models.VerificationCode{}).
		Where("project_id = ? AND email = ? AND created_at >= ?",
			projectID, email, oneHourAgo).
		Count(&emailCount)

	if emailCount >= int64(project.RateLimit) {
		return true, nil
	}

	// Check IP rate limit (per hour)
	var ipCount int64
	s.db.Model(&models.VerificationCode{}).
		Where("project_id = ? AND ip_address = ? AND created_at >= ?",
			projectID, ipAddress, oneHourAgo).
		Count(&ipCount)

	if ipCount >= int64(project.RateLimit) {
		return true, nil
	}

	return false, nil
}
