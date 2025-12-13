package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"verification-api/internal/models"
	"verification-api/internal/services"
	"verification-api/pkg/logging"

	"github.com/gin-gonic/gin"
)

// extractBundleIDFromJWT extracts bundle_id from signed_transaction JWT
func extractBundleIDFromJWT(signedTransaction string) (string, error) {
	// JWT format: header.payload.signature
	parts := strings.Split(signedTransaction, ".")
	if len(parts) != 3 {
		return "", nil
	}

	// Decode payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}

	// Parse claims
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", err
	}

	// Extract bundle_id
	if bundleID, ok := claims["bundleId"].(string); ok {
		return bundleID, nil
	}

	return "", nil
}

// VerifySubscriptionRequest represents verify subscription request
// Supports platform-specific fields as per industry standards
type VerifySubscriptionRequest struct {
	Platform  string `json:"platform" binding:"required,oneof=ios android"` // ios or android
	UserID    string `json:"user_id" binding:"required"`                    // User ID from the app
	ProductID string `json:"product_id" binding:"required"`                 // Product ID (required for both platforms)

	// iOS specific fields
	SignedTransaction string `json:"signed_transaction,omitempty"` // JWT signed transaction (iOS)
	TransactionID     string `json:"transaction_id,omitempty"`     // Transaction ID (iOS)

	// Android specific fields
	PurchaseToken string `json:"purchase_token,omitempty"` // Purchase token (Android)

	// Legacy support (deprecated, use platform-specific fields)
	ReceiptData string `json:"receipt_data,omitempty"` // Legacy: Base64 receipt (iOS) or purchase token (Android)
	AppID       string `json:"app_id,omitempty"`       // Legacy: Bundle ID (iOS) or Package Name (Android)
}

// VerifySubscriptionResponse represents verify subscription response
type VerifySubscriptionResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	IsActive    bool   `json:"is_active"`
	Platform    string `json:"platform,omitempty"`     // Platform: ios or android
	ExpiresDate string `json:"expires_date,omitempty"` // ISO 8601 format
	ProductID   string `json:"product_id,omitempty"`
	AutoRenew   bool   `json:"auto_renew,omitempty"`

	// Legacy support (deprecated)
	ExpiresAt string `json:"expires_at,omitempty"` // Deprecated: use expires_date
}

// VerifySubscription verifies subscription receipt/token
// POST /api/subscription/verify
// Supports both new platform-specific format and legacy format
func VerifySubscription(c *gin.Context) {
	var req VerifySubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, VerifySubscriptionResponse{
			Success: false,
			Message: "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate platform-specific fields
	if req.Platform == "ios" {
		// iOS requires signed_transaction or transaction_id (or legacy receipt_data)
		if req.SignedTransaction == "" && req.TransactionID == "" && req.ReceiptData == "" {
			c.JSON(http.StatusBadRequest, VerifySubscriptionResponse{
				Success: false,
				Message: "iOS requires signed_transaction or transaction_id",
			})
			return
		}
	} else if req.Platform == "android" {
		// Android requires purchase_token (or legacy receipt_data)
		if req.PurchaseToken == "" && req.ReceiptData == "" {
			c.JSON(http.StatusBadRequest, VerifySubscriptionResponse{
				Success: false,
				Message: "Android requires purchase_token",
			})
			return
		}
	}

	// Get project - try to get from bundle_id/package_name if available
	// Otherwise, we'll need to extract from signed_transaction or use default
	projectService := services.NewProjectService()
	var project *models.Project
	var err error

	// Try to get project from app_id or extract from signed_transaction
	var bundleID string

	if req.AppID != "" {
		// Use provided app_id
		if req.Platform == "ios" {
			project, err = projectService.GetProjectByBundleID(req.AppID)
		} else {
			project, err = projectService.GetProjectByPackageName(req.AppID)
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, VerifySubscriptionResponse{
				Success: false,
				Message: "App not found: " + err.Error(),
			})
			return
		}
	} else if req.Platform == "ios" && req.SignedTransaction != "" {
		// Try to extract bundle_id from signed_transaction JWT
		bundleID, err = extractBundleIDFromJWT(req.SignedTransaction)
		if err != nil || bundleID == "" {
			c.JSON(http.StatusBadRequest, VerifySubscriptionResponse{
				Success: false,
				Message: "app_id is required (could not extract bundle_id from signed_transaction)",
			})
			return
		}
		project, err = projectService.GetProjectByBundleID(bundleID)
		if err != nil {
			c.JSON(http.StatusBadRequest, VerifySubscriptionResponse{
				Success: false,
				Message: "App not found for bundle_id: " + bundleID,
			})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, VerifySubscriptionResponse{
			Success: false,
			Message: "app_id is required (or provide signed_transaction for iOS)",
		})
		return
	}

	// 添加详细日志：项目信息
	logging.Infof("验证订阅请求 - ProjectID: %s, ProjectName: %s, BundleID: %s, UserID: %s, TransactionID: %s, ProductID: %s, Platform: %s",
		project.ProjectID, project.ProjectName, project.BundleID, req.UserID, req.TransactionID, req.ProductID, req.Platform)

	// Verify receipt/token
	verificationService := services.NewSubscriptionVerificationService()
	var subscription *models.Subscription

	if req.Platform == "ios" {
		// Use new format if available, fallback to legacy
		if req.SignedTransaction != "" || req.TransactionID != "" {
			subscription, err = verificationService.VerifyAppleTransaction(
				project.ProjectID,
				req.SignedTransaction,
				req.TransactionID,
				req.ProductID,
				req.UserID,
			)
		} else {
			// Legacy format
			subscription, err = verificationService.VerifyAppleReceipt(project.ProjectID, req.ReceiptData, req.UserID)
		}
	} else {
		// Android
		purchaseToken := req.PurchaseToken
		if purchaseToken == "" {
			purchaseToken = req.ReceiptData // Legacy support
		}
		subscription, err = verificationService.VerifyGooglePlayPurchase(
			project.ProjectID,
			purchaseToken,
			req.ProductID,
			req.UserID,
		)
	}

	if err != nil {
		// 添加详细日志：验证失败
		logging.Errorf("订阅验证失败 - ProjectID: %s, ProjectName: %s, BundleID: %s, UserID: %s, TransactionID: %s, Error: %v",
			project.ProjectID, project.ProjectName, project.BundleID, req.UserID, req.TransactionID, err)
		c.JSON(http.StatusBadRequest, VerifySubscriptionResponse{
			Success: false,
			Message: "Verification failed: " + err.Error(),
		})
		return
	}

	// 添加详细日志：验证成功
	isActive := subscription.Status == "active" && subscription.ExpiresDate.After(time.Now())
	logging.Infof("订阅验证成功 - ProjectID: %s, UserID: %s, TransactionID: %s, Status: %s, IsActive: %v, ExpiresDate: %s",
		project.ProjectID, req.UserID, subscription.TransactionID, subscription.Status, isActive, subscription.ExpiresDate.Format(time.RFC3339))

	// Notify App Backend via webhook if configured (optional, for pre-order flow)
	if project.WebhookCallbackURL != "" {
		go func() {
			webhookNotifier := services.NewWebhookNotifier()
			webhookNotifier.NotifyAppBackend(project.WebhookCallbackURL, project.WebhookSecret, subscription)
		}()
	}

	c.JSON(http.StatusOK, VerifySubscriptionResponse{
		Success:     true,
		Message:     "Subscription verified successfully",
		IsActive:    isActive,
		Platform:    subscription.Platform,
		ExpiresDate: subscription.ExpiresDate.Format(time.RFC3339),
		ExpiresAt:   subscription.ExpiresDate.Format(time.RFC3339), // Legacy support
		ProductID:   subscription.ProductID,
		AutoRenew:   subscription.AutoRenewStatus,
	})
}
