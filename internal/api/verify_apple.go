package api

import (
	"net/http"
	"time"
	"verification-api/internal/database"
	"verification-api/internal/models"
	"verification-api/internal/services"
	"verification-api/pkg/logging"

	"github.com/gin-gonic/gin"
)

// VerifyAppleRequest 验证 Apple 交易请求
type VerifyAppleRequest struct {
	TransactionID string `json:"transaction_id" binding:"required"` // 交易ID
	ProjectID    string `json:"project_id,omitempty"`               // 项目ID（可选，如果不提供则从 header 获取）
}

// VerifyAppleResponse 验证 Apple 交易响应
type VerifyAppleResponse struct {
	Success     bool                `json:"success"`
	UserID      string              `json:"user_id"`      // device_id（从 appAccountToken 解析）
	Entitlements EntitlementsResult `json:"entitlements"` // 权益信息
}

// EntitlementsResult 权益结果
type EntitlementsResult struct {
	Subscription SubscriptionResult `json:"subscription"` // 订阅权益
	Lifetime     LifetimeResult     `json:"lifetime"`     // 终身购买权益
}

// SubscriptionResult 订阅结果
type SubscriptionResult struct {
	Active    bool   `json:"active"`     // 是否有效
	ExpiresAt string `json:"expires_at"` // ISO 8601 格式的过期时间
}

// LifetimeResult 终身购买结果
type LifetimeResult struct {
	HasPurchase bool `json:"has_purchase"` // 是否有终身购买
}

// VerifyApple 验证 Apple 交易
// @Summary 验证 Apple 交易
// @Description App Server 调用此接口验证交易，UnionHub 调 App Store Server API 并返回业务可用结果
// @Tags Verify
// @Accept json
// @Produce json
// @Param X-Project-ID header string true "项目ID"
// @Param X-API-Key header string true "API密钥"
// @Param request body VerifyAppleRequest true "验证请求"
// @Success 200 {object} VerifyAppleResponse
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/verify/apple [post]
func VerifyApple(c *gin.Context) {
	// 获取项目认证信息
	projectID := c.GetHeader("X-Project-ID")
	apiKey := c.GetHeader("X-API-Key")

	if projectID == "" || apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Missing X-Project-ID or X-API-Key header",
		})
		return
	}

	// 验证项目
	projectService := services.NewProjectService()
	project, err := projectService.GetProjectByID(projectID)
	if err != nil || project.APIKey != apiKey {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Invalid project credentials",
		})
		return
	}

	// 解析请求
	var req VerifyAppleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	// 如果请求中提供了 project_id，使用请求中的（允许覆盖）
	if req.ProjectID != "" {
		projectID = req.ProjectID
	}

	logging.Infof("验证 Apple 交易 - ProjectID: %s, TransactionID: %s", projectID, req.TransactionID)

	// 调用验证服务
	verificationService := services.NewSubscriptionVerificationService()
	
	// 使用空字符串作为 userID，因为我们会从 appAccountToken 解析
	subscription, err := verificationService.VerifyAppleTransaction(
		projectID,
		"", // signedTransaction（不需要，因为我们有 transactionID）
		req.TransactionID,
		"", // productID（可选）
		"", // userID（会从 appAccountToken 解析）
	)

	if err != nil {
		logging.Errorf("验证 Apple 交易失败 - ProjectID: %s, TransactionID: %s, Error: %v", projectID, req.TransactionID, err)
		c.JSON(http.StatusBadRequest, VerifyAppleResponse{
			Success: false,
		})
		return
	}

	// 从 subscription 获取 user_id（device_id）
	// 如果 subscription.UserID 为空，说明 appAccountToken 还未绑定
	// 此时需要从 App Backend 查询映射关系
	userID := subscription.UserID
	if userID == "" {
		// 尝试从 App Backend 查询映射关系
		// 注意：这里需要先获取 appAccountToken，但 subscription 中可能没有
		// 我们需要从 App Store Server API 响应中获取 appAccountToken
		// 暂时返回错误，因为无法确定 user_id
		logging.Warnf("无法确定 user_id - TransactionID: %s, Subscription.UserID 为空", req.TransactionID)
		c.JSON(http.StatusBadRequest, VerifyAppleResponse{
			Success: false,
		})
		return
	}

	// 判断订阅是否有效
	isActive := subscription.Status == "active" && subscription.ExpiresDate.After(time.Now())
	expiresAtStr := ""
	if subscription.ExpiresDate.After(time.Now()) {
		expiresAtStr = subscription.ExpiresDate.Format(time.RFC3339)
	}

	// 检查是否有终身购买（通过查询 transactions 表，type = non_consumable）
	// 注意：这里需要先查询 transactions 表，但目前 transactions 表可能还没有数据
	// 暂时返回 false，后续可以通过 Server Notification 更新
	hasLifetime := false
	var transaction models.Transaction
	if err := database.GetDB().Where("project_id = ? AND app_account_token = ? AND type = ?", 
		projectID, userID, "non_consumable").First(&transaction).Error; err == nil {
		hasLifetime = true
	}

	logging.Infof("验证 Apple 交易成功 - ProjectID: %s, UserID: %s, IsActive: %v, ExpiresAt: %s",
		projectID, userID, isActive, expiresAtStr)

	// 返回业务可用结果
	c.JSON(http.StatusOK, VerifyAppleResponse{
		Success: true,
		UserID:  userID,
		Entitlements: EntitlementsResult{
			Subscription: SubscriptionResult{
				Active:    isActive,
				ExpiresAt: expiresAtStr,
			},
			Lifetime: LifetimeResult{
				HasPurchase: hasLifetime,
			},
		},
	})
}

