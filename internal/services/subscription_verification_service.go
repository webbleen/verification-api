package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"verification-api/internal/config"
	"verification-api/internal/database"
	"verification-api/internal/models"
	"verification-api/pkg/logging"
)

// SubscriptionVerificationService provides subscription verification operations
type SubscriptionVerificationService struct {
	httpClient *http.Client
}

// NewSubscriptionVerificationService creates a new subscription verification service
func NewSubscriptionVerificationService() *SubscriptionVerificationService {
	return &SubscriptionVerificationService{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AppleReceiptResponse represents Apple receipt verification response
type AppleReceiptResponse struct {
	Status      int    `json:"status"`
	Environment string `json:"environment"`
	Receipt     struct {
		ReceiptType                string `json:"receipt_type"`
		BundleID                   string `json:"bundle_id"`
		ApplicationVersion         string `json:"application_version"`
		InApp                     []struct {
			TransactionID         string `json:"transaction_id"`
			OriginalTransactionID string `json:"original_transaction_id"`
			ProductID             string `json:"product_id"`
			PurchaseDate          string `json:"purchase_date_ms"`
			ExpiresDate           string `json:"expires_date_ms"`
			IsTrialPeriod         string `json:"is_trial_period"`
		} `json:"in_app"`
		LatestReceiptInfo []struct {
			TransactionID         string `json:"transaction_id"`
			OriginalTransactionID string `json:"original_transaction_id"`
			ProductID             string `json:"product_id"`
			PurchaseDate          string `json:"purchase_date_ms"`
			ExpiresDate           string `json:"expires_date_ms"`
			IsTrialPeriod         string `json:"is_trial_period"`
		} `json:"latest_receipt_info"`
	} `json:"receipt"`
	LatestReceipt string `json:"latest_receipt"`
}

// VerifyAppleReceipt verifies iOS receipt
// Returns error code 21007 means receipt is from sandbox, should retry with sandbox URL
func (s *SubscriptionVerificationService) VerifyAppleReceipt(projectID, receiptData, userID string) (*models.Subscription, error) {
	// Try production first
	subscription, err := s.verifyWithApple(receiptData, "production", projectID, userID)
	if err != nil {
		// If error is 21007 (sandbox receipt), retry with sandbox
		if appleErr, ok := err.(*AppleVerificationError); ok && appleErr.Status == 21007 {
			logging.Infof("Receipt is from sandbox, retrying with sandbox URL")
			return s.verifyWithApple(receiptData, "sandbox", projectID, userID)
		}
		return nil, err
	}
	return subscription, nil
}

// verifyWithApple verifies receipt with Apple's API
func (s *SubscriptionVerificationService) verifyWithApple(receiptData, environment, projectID, userID string) (*models.Subscription, error) {
	var url string
	if environment == "production" {
		url = "https://buy.itunes.apple.com/verifyReceipt"
	} else {
		url = "https://sandbox.itunes.apple.com/verifyReceipt"
	}

	// Prepare request body
	requestBody := map[string]interface{}{
		"receipt-data": receiptData,
	}
	if config.AppConfig.AppStoreSharedSecret != "" {
		requestBody["password"] = config.AppConfig.AppStoreSharedSecret
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make request
	resp, err := s.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to verify receipt: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var appleResp AppleReceiptResponse
	if err := json.Unmarshal(body, &appleResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check status
	if appleResp.Status != 0 {
		return nil, &AppleVerificationError{Status: appleResp.Status}
	}

	// Parse receipt and create subscription
	if len(appleResp.Receipt.LatestReceiptInfo) == 0 {
		return nil, fmt.Errorf("no subscription found in receipt")
	}

	// Get the latest receipt info
	latestReceiptInfo := appleResp.Receipt.LatestReceiptInfo[len(appleResp.Receipt.LatestReceiptInfo)-1]

	// Parse dates
	purchaseDate, err := parseAppleTimestamp(latestReceiptInfo.PurchaseDate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse purchase date: %w", err)
	}

	expiresDate, err := parseAppleTimestamp(latestReceiptInfo.ExpiresDate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expires date: %w", err)
	}

	// Determine status
	status := "active"
	if expiresDate.Before(time.Now()) {
		status = "expired"
	}

	// Determine plan from product ID
	plan := getPlanFromProductID(latestReceiptInfo.ProductID)

	// Create subscription model
	subscription := &models.Subscription{
		UserID:                userID,
		ProjectID:             projectID,
		Platform:              "ios",
		Plan:                   plan,
		Status:                 status,
		StartDate:              purchaseDate,
		EndDate:                expiresDate,
		ProductID:              latestReceiptInfo.ProductID,
		TransactionID:          latestReceiptInfo.TransactionID,
		OriginalTransactionID: latestReceiptInfo.OriginalTransactionID,
		Environment:            appleResp.Environment,
		PurchaseDate:           purchaseDate,
		ExpiresDate:            expiresDate,
		AutoRenewStatus:        true, // Default, will be updated by webhook
		LatestReceipt:          appleResp.LatestReceipt,
		LatestReceiptInfo:      string(body),
	}

	// Save or update subscription
	if err := database.CreateOrUpdateSubscription(subscription); err != nil {
		return nil, fmt.Errorf("failed to save subscription: %w", err)
	}

	return subscription, nil
}

// VerifyGooglePlayPurchase verifies Android purchase (placeholder for future implementation)
func (s *SubscriptionVerificationService) VerifyGooglePlayPurchase(projectID, purchaseToken, productID, userID string) (*models.Subscription, error) {
	// TODO: Implement Google Play verification
	return nil, fmt.Errorf("Google Play verification not yet implemented")
}

// AppleVerificationError represents Apple verification error
type AppleVerificationError struct {
	Status int
}

func (e *AppleVerificationError) Error() string {
	return fmt.Sprintf("Apple verification failed with status: %d", e.Status)
}

// parseAppleTimestamp parses Apple timestamp (milliseconds since epoch)
func parseAppleTimestamp(timestampStr string) (time.Time, error) {
	if timestampStr == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	var timestamp int64
	if _, err := fmt.Sscanf(timestampStr, "%d", &timestamp); err != nil {
		return time.Time{}, err
	}
	return time.Unix(timestamp/1000, (timestamp%1000)*1000000), nil
}

// getPlanFromProductID determines plan from product ID
func getPlanFromProductID(productID string) string {
	switch productID {
	case "com.dailyzen.monthly", "monthly":
		return "monthly"
	case "com.dailyzen.yearly", "yearly":
		return "yearly"
	default:
		return "basic"
	}
}

