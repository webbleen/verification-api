package models

// AppStoreNotification represents App Store Server Notification V2
// Apple uses camelCase for field names
type AppStoreNotification struct {
	NotificationType   string          `json:"notificationType"`   // e.g., "SUBSCRIBED", "DID_RENEW"
	Subtype            string          `json:"subtype,omitempty"`  // Optional subtype
	NotificationUUID   string          `json:"notificationUUID"`    // Unique notification ID
	DataVersion        string          `json:"dataVersion"`         // Version of the data format
	SignedDate         int64           `json:"signedDate"`          // Timestamp when notification was signed
	Data               NotificationData `json:"data"`               // Notification data payload
}

// NotificationData contains notification data
// Apple uses camelCase for field names
type NotificationData struct {
	AppAppleID            int    `json:"appAppleId"`            // Apple App ID
	BundleID              string `json:"bundleId"`              // App bundle identifier
	BundleVersion         string `json:"bundleVersion"`         // App version
	Environment           string `json:"environment"`           // "Sandbox" or "Production"
	SignedTransactionInfo string `json:"signedTransactionInfo"`  // JWT containing transaction info
}

// TransactionInfo represents decoded transaction information
type TransactionInfo struct {
	TransactionID         string `json:"transaction_id"`
	OriginalTransactionID string `json:"original_transaction_id"`
	ProductID             string `json:"product_id"`
	PurchaseDateMS        int64  `json:"purchase_date_ms"`
	ExpiresDateMS         int64  `json:"expires_date_ms"`
	AutoRenewStatus       int    `json:"auto_renew_status"`
	Environment           string `json:"environment"`
}

