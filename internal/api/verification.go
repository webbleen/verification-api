package api

import (
	"net/http"
	"time"
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

	// Get client IP and User-Agent for logging
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	// Initialize services
	verificationService := services.NewVerificationService()
	redisService, err := services.NewRedisService()
	if err != nil {
		c.JSON(http.StatusInternalServerError, SendCodeResponse{
			Success: false,
			Message: "Service unavailable",
		})
		return
	}

	// Check rate limit using database
	rateLimited, err := verificationService.CheckRateLimit(projectID.(string), req.Email, clientIP)
	if err != nil {
		c.JSON(http.StatusInternalServerError, SendCodeResponse{
			Success: false,
			Message: "Service error",
		})
		return
	}

	if rateLimited {
		// Log rate limit attempt
		verificationService.LogVerificationAttempt(projectID.(string), req.Email, "send", false, clientIP, userAgent, "Rate limit exceeded")

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

	// Calculate expiration time
	expiresAt := time.Now().Add(time.Duration(config.AppConfig.CodeExpireMinutes) * time.Minute)

	// Store verification code to database
	if err := verificationService.CreateVerificationCode(projectID.(string), req.Email, code, expiresAt, clientIP, userAgent); err != nil {
		c.JSON(http.StatusInternalServerError, SendCodeResponse{
			Success: false,
			Message: "Failed to store verification code",
		})
		return
	}

	// Also store in Redis for quick access
	if err := redisService.StoreCode(projectID.(string), req.Email, code, config.AppConfig.CodeExpireMinutes); err != nil {
		// Log error but don't affect main flow
	}

	// Set rate limit in Redis
	if err := redisService.SetRateLimit(projectID.(string), req.Email, config.AppConfig.RateLimitMinutes); err != nil {
		// Log error but don't affect main flow
	}

	// Send email
	brevoService := services.NewBrevoService()
	if err := brevoService.SendVerificationCodeEmail(projectID.(string), req.Email, code, req.Language); err != nil {
		// Log failed email attempt
		verificationService.LogVerificationAttempt(projectID.(string), req.Email, "send", false, clientIP, userAgent, "Failed to send email: "+err.Error())

		c.JSON(http.StatusInternalServerError, SendCodeResponse{
			Success: false,
			Message: "Failed to send verification email",
		})
		return
	}

	// Log successful send
	verificationService.LogVerificationAttempt(projectID.(string), req.Email, "send", true, clientIP, userAgent, "")

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

	// Get client IP and User-Agent for logging
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	// Initialize services
	verificationService := services.NewVerificationService()
	redisService, err := services.NewRedisService()
	if err != nil {
		c.JSON(http.StatusInternalServerError, VerifyCodeResponse{
			Success: false,
			Message: "Service unavailable",
		})
		return
	}

	// Get verification code from database
	verificationCode, err := verificationService.GetVerificationCode(projectID.(string), req.Email)
	if err != nil {
		// Log failed verification attempt
		verificationService.LogVerificationAttempt(projectID.(string), req.Email, "verify", false, clientIP, userAgent, "Code not found or expired")

		c.JSON(http.StatusBadRequest, VerifyCodeResponse{
			Success: false,
			Message: "Verification code not found or expired",
		})
		return
	}

	// Compare verification codes
	if verificationCode.Code != req.Code {
		// Log failed verification attempt
		verificationService.LogVerificationAttempt(projectID.(string), req.Email, "verify", false, clientIP, userAgent, "Invalid code")

		c.JSON(http.StatusBadRequest, VerifyCodeResponse{
			Success: false,
			Message: "Invalid verification code",
		})
		return
	}

	// Mark code as used in database
	if err := verificationService.MarkCodeAsUsed(verificationCode.ID); err != nil {
		// Log error but don't fail the verification
	}

	// Delete verification code from Redis
	redisService.DeleteCode(projectID.(string), req.Email)

	// Log successful verification
	verificationService.LogVerificationAttempt(projectID.(string), req.Email, "verify", true, clientIP, userAgent, "")

	c.JSON(http.StatusOK, VerifyCodeResponse{
		Success: true,
		Message: "Verification code verified successfully",
	})
}
