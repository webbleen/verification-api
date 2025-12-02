package api

import (
	"net/http"
	"verification-api/internal/config"
	"verification-api/internal/services"

	"github.com/gin-gonic/gin"
)

// SendCodeRequest represents send verification code request
type SendCodeRequest struct {
	Email     string `json:"email" binding:"required,email"`
	ProjectID string `json:"project_id" binding:"required"`
	Language  string `json:"language,omitempty"`
}

// SendCodeResponse represents send verification code response
type SendCodeResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// VerifyCodeRequest represents verify verification code request
type VerifyCodeRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Code      string `json:"code" binding:"required,len=6"`
	ProjectID string `json:"project_id" binding:"required"`
}

// VerifyCodeResponse represents verify verification code response
type VerifyCodeResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// SendVerificationCode sends verification code
func SendVerificationCode(c *gin.Context) {
	var req SendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, SendCodeResponse{
			Success: false,
			Message: "Invalid request format: " + err.Error(),
		})
		return
	}

	// Get project ID from context (set by middleware)
	projectID, exists := c.Get("project_id")
	if !exists {
		projectID = req.ProjectID // If middleware didn't set it, use project ID from request
	}

	// Initialize services
	redisService, err := services.NewRedisService()
	if err != nil {
		c.JSON(http.StatusInternalServerError, SendCodeResponse{
			Success: false,
			Message: "Service unavailable",
		})
		return
	}

	// Check rate limit using Redis
	rateLimited, err := redisService.CheckRateLimit(projectID.(string), req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, SendCodeResponse{
			Success: false,
			Message: "Service error",
		})
		return
	}

	if rateLimited {
		c.JSON(http.StatusTooManyRequests, SendCodeResponse{
			Success: false,
			Message: "Please wait before requesting another verification code",
		})
		return
	}

	// Generate verification code
	code, err := redisService.GenerateCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, SendCodeResponse{
			Success: false,
			Message: "Failed to generate verification code",
		})
		return
	}

	// Store verification code in Redis (with TTL, auto-expire)
	if err := redisService.StoreCode(projectID.(string), req.Email, code, config.AppConfig.CodeExpireMinutes); err != nil {
		c.JSON(http.StatusInternalServerError, SendCodeResponse{
			Success: false,
			Message: "Failed to store verification code",
		})
		return
	}

	// Set rate limit in Redis
	if err := redisService.SetRateLimit(projectID.(string), req.Email, config.AppConfig.RateLimitMinutes); err != nil {
		// Log error but don't affect main flow
	}

	// Send email
	brevoService := services.NewBrevoService()
	if err := brevoService.SendVerificationCodeEmail(projectID.(string), req.Email, code, req.Language); err != nil {
		c.JSON(http.StatusInternalServerError, SendCodeResponse{
			Success: false,
			Message: "Failed to send verification email",
		})
		return
	}

	c.JSON(http.StatusOK, SendCodeResponse{
		Success: true,
		Message: "Verification code sent successfully",
	})
}

// VerifyCode verifies verification code
func VerifyCode(c *gin.Context) {
	var req VerifyCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, VerifyCodeResponse{
			Success: false,
			Message: "Invalid request format: " + err.Error(),
		})
		return
	}

	// Get project ID from context (set by middleware)
	projectID, exists := c.Get("project_id")
	if !exists {
		projectID = req.ProjectID // If middleware didn't set it, use project ID from request
	}

	// Initialize services
	redisService, err := services.NewRedisService()
	if err != nil {
		c.JSON(http.StatusInternalServerError, VerifyCodeResponse{
			Success: false,
			Message: "Service unavailable",
		})
		return
	}

	// Get verification code from Redis
	storedCode, err := redisService.GetCode(projectID.(string), req.Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, VerifyCodeResponse{
			Success: false,
			Message: "Verification code not found or expired",
		})
		return
	}

	// Compare verification codes
	if storedCode != req.Code {
		c.JSON(http.StatusBadRequest, VerifyCodeResponse{
			Success: false,
			Message: "Invalid verification code",
		})
		return
	}

	// Delete verification code from Redis (mark as used)
	redisService.DeleteCode(projectID.(string), req.Email)

	c.JSON(http.StatusOK, VerifyCodeResponse{
		Success: true,
		Message: "Verification code verified successfully",
	})
}
