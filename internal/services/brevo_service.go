package services

import (
	"context"
	"fmt"
	"verification-api/internal/config"
	"verification-api/internal/models"

	brevo "github.com/getbrevo/brevo-go/lib"
)

// BrevoService provides Brevo email service using official SDK
type BrevoService struct {
	client    *brevo.APIClient
	FromEmail string
	FromName  string
}

// NewBrevoService creates a new Brevo service instance
func NewBrevoService() *BrevoService {
	// 创建 Brevo 配置
	cfg := brevo.NewConfiguration()
	cfg.AddDefaultHeader("api-key", config.AppConfig.BrevoAPIKey)

	// 创建 API 客户端
	client := brevo.NewAPIClient(cfg)

	return &BrevoService{
		client:    client,
		FromEmail: config.AppConfig.BrevoFromEmail,
		FromName:  config.AppConfig.BrevoFromName,
	}
}

// SendVerificationCodeEmail sends verification code email (supports multi-project and multi-language)
func (s *BrevoService) SendVerificationCodeEmail(projectID, to, code, language string) error {
	// Get project configuration
	projectConfig := s.getProjectConfig(projectID)

	// Debug: Log language parameter
	fmt.Printf("DEBUG: Language parameter received: '%s'\n", language)

	// Get email content based on language
	// 从配置中获取过期时间，默认为5分钟
	expireMinutes := config.AppConfig.CodeExpireMinutes
	subject, htmlContent, textContent := s.getEmailContent(language, projectConfig.ProjectName, code, expireMinutes)

	// Debug: Log generated content
	fmt.Printf("DEBUG: Generated subject: '%s'\n", subject)

	// 使用官方 SDK 发送邮件
	return s.sendEmailWithSDK(projectConfig.FromName, projectConfig.FromEmail, to, subject, htmlContent, textContent)
}

// getProjectConfig gets project configuration
func (s *BrevoService) getProjectConfig(projectID string) *models.ProjectConfig {
	// Get project configuration from database
	projectService := NewProjectService()
	project, err := projectService.GetProjectByID(projectID)
	if err != nil {
		// Fallback to default configuration if project not found
		return &models.ProjectConfig{
			ProjectID:   projectID,
			ProjectName: "Default Project",
			FromEmail:   s.FromEmail, // Use service default email
			FromName:    s.FromName,  // Use service default name
		}
	}

	// Convert database model to config model
	// Always use the service's configured email (single sender limitation)
	// But use project-specific from_name
	return &models.ProjectConfig{
		ProjectID:   project.ProjectID,
		ProjectName: project.ProjectName,
		FromEmail:   s.FromEmail,      // Force use service default email
		FromName:    project.FromName, // Use project-specific from_name
	}
}

// sendEmailWithSDK sends email using official Brevo SDK
func (s *BrevoService) sendEmailWithSDK(fromName, fromEmail, to, subject, htmlContent, textContent string) error {
	ctx := context.Background()

	// 创建发送者信息
	sender := brevo.SendSmtpEmailSender{
		Name:  fromName,
		Email: fromEmail,
	}

	// 创建收件人信息
	toList := []brevo.SendSmtpEmailTo{
		{
			Email: to,
		},
	}

	// 创建邮件请求
	emailRequest := brevo.SendSmtpEmail{
		Sender:      &sender,
		To:          toList,
		Subject:     subject,
		HtmlContent: htmlContent,
		TextContent: textContent,
	}

	// 发送邮件
	_, httpResp, err := s.client.TransactionalEmailsApi.SendTransacEmail(ctx, emailRequest)
	if err != nil {
		return fmt.Errorf("failed to send email via Brevo SDK: %w", err)
	}

	// 检查响应状态
	if httpResp.StatusCode != 200 && httpResp.StatusCode != 201 {
		return fmt.Errorf("brevo API error: status %d", httpResp.StatusCode)
	}

	return nil
}

// getEmailContent 根据语言获取邮件内容
func (s *BrevoService) getEmailContent(language, projectName, verificationCode string, expireMinutes int) (subject, htmlContent, textContent string) {
	// 默认使用英文
	if language == "" {
		language = "en"
	}

	// 多语言邮件内容（支持动态过期时间和团队名称）
	emailContent := map[string]map[string]string{
		"en": {
			"subject": "%s Verification Code",
			"body":    "Your verification code is: %s\n\nThis code will expire in %d minutes.\n\nIf you didn't request this code, please ignore this email.\n\nBest regards,\n%s Team",
		},
		"zh-CN": {
			"subject": "%s 验证码",
			"body":    "您的验证码是：%s\n\n此验证码将在%d分钟后过期。\n\n如果您没有请求此验证码，请忽略此邮件。\n\n祝好，\n%s 团队",
		},
		"zh-TW": {
			"subject": "%s 驗證碼",
			"body":    "您的驗證碼是：%s\n\n此驗證碼將在%d分鐘後過期。\n\n如果您沒有請求此驗證碼，請忽略此郵件。\n\n祝好，\n%s 團隊",
		},
		"ja": {
			"subject": "%s 認証コード",
			"body":    "認証コードは %s です。\n\nこのコードは%d分後に期限切れになります。\n\nこのコードをリクエストしていない場合は、このメールを無視してください。\n\nよろしくお願いします、\n%s チーム",
		},
		"ko": {
			"subject": "%s 인증 코드",
			"body":    "인증 코드는 %s 입니다.\n\n이 코드는 %d분 후에 만료됩니다.\n\n이 코드를 요청하지 않으셨다면 이 이메일을 무시하세요.\n\n감사합니다,\n%s 팀",
		},
		"es": {
			"subject": "Código de verificación %s",
			"body":    "Su código de verificación es: %s\n\nEste código expirará en %d minutos.\n\nSi no solicitó este código, ignore este correo.\n\nSaludos,\nEquipo %s",
		},
		"fr": {
			"subject": "Code de vérification %s",
			"body":    "Votre code de vérification est : %s\n\nCe code expirera dans %d minutes.\n\nSi vous n'avez pas demandé ce code, ignorez cet e-mail.\n\nCordialement,\nÉquipe %s",
		},
		"de": {
			"subject": "%s Bestätigungscode",
			"body":    "Ihr Bestätigungscode lautet: %s\n\nDieser Code läuft in %d Minuten ab.\n\nFalls Sie diesen Code nicht angefordert haben, ignorieren Sie diese E-Mail.\n\nMit freundlichen Grüßen,\n%s Team",
		},
	}

	// 获取对应语言的内容，如果不存在则使用英文
	content, exists := emailContent[language]
	if !exists {
		content = emailContent["en"]
	}

	// 格式化主题
	subject = fmt.Sprintf(content["subject"], projectName)

	// 生成纯文本内容（支持动态过期时间和团队名称）
	textContent = fmt.Sprintf(content["body"], verificationCode, expireMinutes, projectName)

	// 生成简单的 HTML 内容（保持纯文本格式）
	htmlContent = fmt.Sprintf(`<pre style="font-family: monospace; white-space: pre-wrap;">%s</pre>`, textContent)

	return subject, htmlContent, textContent
}
