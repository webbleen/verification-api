package models

// AppStoreNotification represents App Store Server Notification V2
type AppStoreNotification struct {
	NotificationType   string `json:"notification_type"`
	Subtype            string `json:"subtype,omitempty"`
	NotificationUUID   string `json:"notification_uuid"`
	DataVersion        string `json:"data_version"`
	SignedDate         int64  `json:"signed_date"`
	Data               NotificationData `json:"data"`
}

// NotificationData contains notification data
type NotificationData struct {
	AppAppleID            int    `json:"app_apple_id"`
	BundleID              string `json:"bundle_id"`
	BundleVersion         string `json:"bundle_version"`
	Environment           string `json:"environment"`
	SignedTransactionInfo string `json:"signed_transaction_info"`
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

