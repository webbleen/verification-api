package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"verification-api/internal/models"
	"verification-api/pkg/logging"
)

// WebhookNotifier handles webhook notifications to App Backend
type WebhookNotifier struct {
	httpClient *http.Client
}

// NewWebhookNotifier creates a new webhook notifier
func NewWebhookNotifier() *WebhookNotifier {
	return &WebhookNotifier{
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // 10 second timeout
		},
	}
}

// WebhookPayload represents the payload sent to App Backend
type WebhookPayload struct {
	Event                 string `json:"event"`                   // e.g., "subscription.updated"
	TransactionID         string `json:"transaction_id"`          // App Store/Google Play transaction ID
	OriginalTransactionID string `json:"original_transaction_id"` // Original transaction ID (for renewals)
	AppAccountToken       string `json:"app_account_token"`       // App Account Token (UUID format)
	Status                string `json:"status"`                  // Subscription status: active, cancelled, expired, refunded, etc.
	ProductID             string `json:"product_id"`              // Product ID
	ExpiresDate           string `json:"expires_date"`            // ISO 8601 format
	Platform              string `json:"platform"`                // ios or android
	Timestamp             string `json:"timestamp"`               // ISO 8601 format
}

// NotifyAppBackend sends webhook notification to App Backend
// This function is called asynchronously (in goroutine) to avoid blocking
func (wn *WebhookNotifier) NotifyAppBackend(callbackURL string, secret string, subscription *models.Subscription) {
	if callbackURL == "" {
		// No webhook configured, skip
		return
	}

	// Create payload
	payload := WebhookPayload{
		Event:                 "subscription.updated",
		TransactionID:         subscription.TransactionID,
		OriginalTransactionID: subscription.OriginalTransactionID,
		AppAccountToken:       subscription.AppAccountToken,
		Status:                subscription.Status,
		ProductID:             subscription.ProductID,
		ExpiresDate:           subscription.ExpiresDate.Format(time.RFC3339),
		Platform:              subscription.Platform,
		Timestamp:             time.Now().Format(time.RFC3339),
	}

	// Send with retry mechanism
	wn.sendWithRetry(callbackURL, secret, payload)
}

// sendWithRetry sends webhook with retry mechanism
// Retry schedule: 1s, 5s, 30s (3 attempts total)
func (wn *WebhookNotifier) sendWithRetry(callbackURL string, secret string, payload WebhookPayload) {
	retryDelays := []time.Duration{1 * time.Second, 5 * time.Second, 30 * time.Second}
	maxRetries := len(retryDelays)

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := wn.sendWebhook(callbackURL, secret, payload)
		if err == nil {
			logging.Infof("Webhook notification sent successfully - url: %s, transaction: %s, attempt: %d",
				callbackURL, payload.TransactionID, attempt+1)
			return
		}

		logging.Errorf("Webhook notification failed - url: %s, transaction: %s, attempt: %d, error: %v",
			callbackURL, payload.TransactionID, attempt+1, err)

		// If not the last attempt, wait before retry
		if attempt < maxRetries-1 {
			time.Sleep(retryDelays[attempt])
		}
	}

	logging.Errorf("Webhook notification failed after %d attempts - url: %s, transaction: %s",
		maxRetries, callbackURL, payload.TransactionID)
}

// sendWebhook sends a single webhook request
func (wn *WebhookNotifier) sendWebhook(callbackURL string, secret string, payload WebhookPayload) error {
	// Marshal payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", callbackURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "UnionHub-Webhook/1.0")

	// Add signature if secret is provided
	if secret != "" {
		signature := wn.generateSignature(jsonData, secret)
		req.Header.Set("X-UnionHub-Signature", signature)
	}

	// Send request
	resp, err := wn.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// generateSignature generates HMAC-SHA256 signature for webhook payload
func (wn *WebhookNotifier) generateSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}
