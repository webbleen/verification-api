package services

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"verification-api/internal/config"
	"verification-api/internal/database"
	"verification-api/internal/models"
	"verification-api/pkg/logging"

	"github.com/golang-jwt/jwt/v5"
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
		ReceiptType        string `json:"receipt_type"`
		BundleID           string `json:"bundle_id"`
		ApplicationVersion string `json:"application_version"`
		InApp              []struct {
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
		Plan:                  plan,
		Status:                status,
		StartDate:             purchaseDate,
		EndDate:               expiresDate,
		ProductID:             latestReceiptInfo.ProductID,
		TransactionID:         latestReceiptInfo.TransactionID,
		OriginalTransactionID: latestReceiptInfo.OriginalTransactionID,
		Environment:           appleResp.Environment,
		PurchaseDate:          purchaseDate,
		ExpiresDate:           expiresDate,
		AutoRenewStatus:       true, // Default, will be updated by webhook
		LatestReceipt:         appleResp.LatestReceipt,
		LatestReceiptInfo:     string(body),
	}

	// Save or update subscription
	if err := database.CreateOrUpdateSubscription(subscription); err != nil {
		return nil, fmt.Errorf("failed to save subscription: %w", err)
	}

	return subscription, nil
}

// VerifyAppleTransaction verifies iOS transaction using App Store Server API (modern approach)
// Uses signed_transaction JWT or transaction_id to query App Store Server API
func (s *SubscriptionVerificationService) VerifyAppleTransaction(projectID, signedTransaction, transactionID, productID, userID string) (*models.Subscription, error) {
	// Parse signed_transaction JWT if provided
	var actualTransactionID string
	var bundleID string
	var environment string

	if signedTransaction != "" {
		// Parse JWT to extract transaction_id (without verification, just parsing)
		parser := jwt.NewParser(jwt.WithoutClaimsValidation())
		token, err := parser.Parse(signedTransaction, func(token *jwt.Token) (interface{}, error) {
			// Apple uses ES256, we don't need to verify signature here, just parse
			return nil, nil
		})

		if err == nil {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if tid, ok := claims["transactionId"].(string); ok {
					actualTransactionID = tid
				}
				if bid, ok := claims["bundleId"].(string); ok {
					bundleID = bid // Store for potential future use
					_ = bundleID   // Suppress unused warning for now
				}
				if env, ok := claims["environment"].(string); ok {
					environment = env
				}
			}
		}
	}

	// Use provided transaction_id if JWT parsing didn't work
	if actualTransactionID == "" {
		actualTransactionID = transactionID
	}

	if actualTransactionID == "" {
		return nil, fmt.Errorf("transaction_id is required")
	}

	// Determine environment (default to production if not specified)
	if environment == "" {
		environment = "Production"
	}

	// Get project to retrieve bundle_id (using database directly to avoid circular import)
	db := database.GetDB()
	var project models.Project
	if err := db.Where("project_id = ? AND is_active = ?", projectID, true).First(&project).Error; err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Generate JWT token for App Store Server API authentication
	authToken, err := s.generateAppStoreJWT(project.BundleID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth token: %w", err)
	}

	// Call App Store Server API
	apiURL := fmt.Sprintf("https://api.storekit.itunes.apple.com/inApps/v1/transactions/%s", actualTransactionID)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call App Store Server API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("App Store Server API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse transaction response
	var transactionResp struct {
		SignedTransactionInfo string `json:"signedTransactionInfo"`
	}

	if err := json.Unmarshal(body, &transactionResp); err != nil {
		return nil, fmt.Errorf("failed to parse transaction response: %w", err)
	}

	// signedTransactionInfo is a JWT (header.payload.signature), not base64-encoded JSON
	// Parse it as JWT to extract claims
	parts := strings.Split(transactionResp.SignedTransactionInfo, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format: expected 3 parts, got %d", len(parts))
	}

	// Decode payload (second part) - JWT uses base64.RawURLEncoding
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var transactionInfo struct {
		TransactionID         string `json:"transactionId"`
		OriginalTransactionID string `json:"originalTransactionId"`
		ProductID             string `json:"productId"`
		PurchaseDate          int64  `json:"purchaseDate"`
		ExpiresDate           int64  `json:"expiresDate"`
		Environment           string `json:"environment"`
		IsInBillingRetry      bool   `json:"isInBillingRetry"`
		IsInGracePeriod       bool   `json:"isInGracePeriod"`
		IsTrialPeriod         bool   `json:"isTrialPeriod"`
		AppAccountToken       string `json:"appAccountToken"` // Extract appAccountToken
	}

	if err := json.Unmarshal(payload, &transactionInfo); err != nil {
		return nil, fmt.Errorf("failed to parse transaction info: %w", err)
	}

	// Parse dates
	purchaseDate := time.Unix(transactionInfo.PurchaseDate/1000, 0)
	expiresDate := time.Unix(transactionInfo.ExpiresDate/1000, 0)

	// Determine status
	status := "active"
	if expiresDate.Before(time.Now()) {
		status = "expired"
	}
	if transactionInfo.IsInBillingRetry {
		status = "billing_retry"
	}
	if transactionInfo.IsInGracePeriod {
		status = "grace_period"
	}

	// Determine plan from product ID
	plan := getPlanFromProductID(transactionInfo.ProductID)
	if productID != "" && productID != transactionInfo.ProductID {
		// Use provided product_id if different
		plan = getPlanFromProductID(productID)
	}

	// Normalize environment
	env := strings.ToLower(transactionInfo.Environment)
	if env == "production" {
		env = "production"
	} else {
		env = "sandbox"
	}

	// Use appAccountToken from API response if available, otherwise use provided userID
	finalUserID := userID
	if transactionInfo.AppAccountToken != "" {
		finalUserID = transactionInfo.AppAccountToken
		logging.Infof("Using appAccountToken from App Store Server API: %s", finalUserID)
	} else {
		// appAccountToken is empty (client didn't set it), use provided userID
		logging.Infof("No appAccountToken in API response, using provided userID: %s", finalUserID)
	}

	// Create subscription model
	subscription := &models.Subscription{
		UserID:                finalUserID,
		ProjectID:             projectID,
		Platform:              "ios",
		Plan:                  plan,
		Status:                status,
		StartDate:             purchaseDate,
		EndDate:               expiresDate,
		ProductID:             transactionInfo.ProductID,
		TransactionID:         transactionInfo.TransactionID,
		OriginalTransactionID: transactionInfo.OriginalTransactionID,
		Environment:           env,
		PurchaseDate:          purchaseDate,
		ExpiresDate:           expiresDate,
		AutoRenewStatus:       true, // Will be updated by webhook
		LatestReceipt:         signedTransaction,
		LatestReceiptInfo:     string(body),
	}

	// Save or update subscription
	if err := database.CreateOrUpdateSubscription(subscription); err != nil {
		return nil, fmt.Errorf("failed to save subscription: %w", err)
	}

	return subscription, nil
}

// generateAppStoreJWT generates JWT token for App Store Server API authentication
// bundleID is optional and can be empty (Apple allows omitting bid in JWT)
func (s *SubscriptionVerificationService) generateAppStoreJWT(bundleID string) (string, error) {
	keyID := config.AppConfig.AppStoreKeyID
	issuerID := config.AppConfig.AppStoreIssuerID
	privateKey := config.AppConfig.AppStorePrivateKey

	if keyID == "" || issuerID == "" || privateKey == "" {
		return "", fmt.Errorf("App Store API credentials not configured")
	}

	// Load private key from environment variable (base64 or PEM)
	key, err := loadPrivateKeyFromString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to load private key: %w", err)
	}

	// Create JWT token
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": issuerID,
		"iat": now.Unix(),
		"exp": now.Add(20 * time.Minute).Unix(),
		"aud": "appstoreconnect-v1",
	}
	// Add bundle_id only if provided (optional field)
	if bundleID != "" {
		claims["bid"] = bundleID
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)

	token.Header["kid"] = keyID

	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// loadPrivateKeyFromString loads ECDSA private key from string (PEM or base64)
func loadPrivateKeyFromString(keyStr string) (*ecdsa.PrivateKey, error) {
	// Try to decode as base64 first
	decoded, err := base64.StdEncoding.DecodeString(keyStr)
	if err == nil {
		keyStr = string(decoded)
	}

	// Parse PEM
	block, _ := pem.Decode([]byte(keyStr))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	// Parse private key
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	ecdsaKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not ECDSA private key")
	}

	return ecdsaKey, nil
}

// VerifyGooglePlayPurchase verifies Android purchase using Google Play Developer API
func (s *SubscriptionVerificationService) VerifyGooglePlayPurchase(projectID, purchaseToken, productID, userID string) (*models.Subscription, error) {
	// TODO: Implement Google Play verification using Google Play Developer API
	// API: GET https://androidpublisher.googleapis.com/androidpublisher/v3/applications/{packageName}/purchases/subscriptions/{subscriptionId}/tokens/{token}
	// Requires: Google Service Account credentials
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
