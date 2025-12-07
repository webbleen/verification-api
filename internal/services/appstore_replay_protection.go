package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
	"verification-api/pkg/logging"
)

// ReplayProtection 重放攻击防护
type ReplayProtection struct {
	processedNotifications map[string]time.Time
	mutex                  sync.RWMutex
	cleanupInterval        time.Duration
	notificationTTL        time.Duration
	stopCleanup            chan bool
}

// NewReplayProtection 创建重放攻击防护实例
func NewReplayProtection() *ReplayProtection {
	rp := &ReplayProtection{
		processedNotifications: make(map[string]time.Time),
		cleanupInterval:        time.Hour,      // 每小时清理一次
		notificationTTL:        time.Hour * 24, // 通知记录保存24小时
		stopCleanup:            make(chan bool),
	}

	// 启动清理协程
	go rp.startCleanupRoutine()

	return rp
}

// IsReplay 检查是否为重放攻击
// 返回 true 如果是重放，false 如果不是
func (rp *ReplayProtection) IsReplay(notificationUUID string, timestamp int64) bool {
	if notificationUUID == "" {
		// 如果没有 UUID，无法判断，返回 false（允许处理）
		logging.Infof("Notification UUID is empty, skipping replay check")
		return false
	}

	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	// 生成通知的唯一标识符
	notificationID := rp.generateNotificationID(notificationUUID, timestamp)

	// 检查是否已处理过
	if processedTime, exists := rp.processedNotifications[notificationID]; exists {
		logging.Infof("Replay detected - notification_id: %s, previously processed at: %v", notificationID, processedTime)
		return true
	}

	// 记录通知
	rp.processedNotifications[notificationID] = time.Now()
	logging.Infof("New notification recorded - notification_id: %s", notificationID)

	return false
}

// generateNotificationID 生成通知的唯一标识符
func (rp *ReplayProtection) generateNotificationID(notificationUUID string, timestamp int64) string {
	// 使用 SHA256 哈希生成唯一标识符
	data := fmt.Sprintf("%s:%d", notificationUUID, timestamp)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// startCleanupRoutine 启动清理协程
func (rp *ReplayProtection) startCleanupRoutine() {
	ticker := time.NewTicker(rp.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rp.cleanup()
		case <-rp.stopCleanup:
			return
		}
	}
}

// cleanup 清理过期的通知记录
func (rp *ReplayProtection) cleanup() {
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	now := time.Now()
	initialCount := len(rp.processedNotifications)

	for notificationID, processedTime := range rp.processedNotifications {
		if now.Sub(processedTime) > rp.notificationTTL {
			delete(rp.processedNotifications, notificationID)
		}
	}

	cleanedCount := initialCount - len(rp.processedNotifications)
	if cleanedCount > 0 {
		logging.Infof("Replay protection cleanup: removed %d expired notifications, remaining: %d", cleanedCount, len(rp.processedNotifications))
	}
}

// GetStats 获取统计信息
func (rp *ReplayProtection) GetStats() map[string]interface{} {
	rp.mutex.RLock()
	defer rp.mutex.RUnlock()

	return map[string]interface{}{
		"total_processed":  len(rp.processedNotifications),
		"cleanup_interval": rp.cleanupInterval.String(),
		"notification_ttl": rp.notificationTTL.String(),
	}
}

// Clear 清空所有记录（用于测试）
func (rp *ReplayProtection) Clear() {
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	rp.processedNotifications = make(map[string]time.Time)
}

// Stop 停止清理协程
func (rp *ReplayProtection) Stop() {
	close(rp.stopCleanup)
}

