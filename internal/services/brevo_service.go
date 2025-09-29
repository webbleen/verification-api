package services

import (
	"auth-mail/internal/config"
	"auth-mail/internal/models"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// BrevoService provides Brevo email service
type BrevoService struct {
	APIKey    string
	FromEmail string
	FromName  string
}

// NewBrevoService creates a new Brevo service instance
func NewBrevoService() *BrevoService {
	return &BrevoService{
		APIKey:    config.AppConfig.BrevoAPIKey,
		FromEmail: config.AppConfig.BrevoFromEmail,
		FromName:  config.AppConfig.BrevoFromName,
	}
}

// EmailRequest represents Brevo email request structure
type EmailRequest struct {
	Sender      EmailSender `json:"sender"`
	To          []EmailTo   `json:"to"`
	Subject     string      `json:"subject"`
	HTMLContent string      `json:"htmlContent"`
	TextContent string      `json:"textContent"`
}

type EmailSender struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type EmailTo struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// SendVerificationCodeEmail sends verification code email (supports multi-project)
func (s *BrevoService) SendVerificationCodeEmail(projectID, to, code string) error {
	// Get project configuration
	projectConfig := s.getProjectConfig(projectID)

	subject := fmt.Sprintf("验证码 - %s", projectConfig.ProjectName)
	htmlContent := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<title>验证码</title>
		</head>
		<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="background-color: #f8f9fa; padding: 30px; border-radius: 10px; text-align: center;">
				<h1 style="color: #333; margin-bottom: 20px;">%s 验证码</h1>
				<p style="color: #666; font-size: 16px; margin-bottom: 20px;">您的验证码是：</p>
				<div style="background-color: #007bff; color: white; padding: 20px; border-radius: 10px; font-size: 32px; font-weight: bold; letter-spacing: 5px; margin: 20px 0;">
					%s
				</div>
				<p style="color: #999; font-size: 14px; margin-top: 20px;">验证码5分钟内有效，请勿泄露给他人。</p>
				<p style="color: #999; font-size: 12px; margin-top: 30px;">如果这不是您的操作，请忽略此邮件。</p>
			</div>
		</body>
		</html>
	`, projectConfig.ProjectName, code)

	textContent := fmt.Sprintf(`
		%s 验证码
		
		您的验证码是：%s
		
		验证码5分钟内有效，请勿泄露给他人。
		
		如果这不是您的操作，请忽略此邮件。
	`, projectConfig.ProjectName, code)

	emailReq := EmailRequest{
		Sender: EmailSender{
			Name:  projectConfig.FromName,
			Email: projectConfig.FromEmail,
		},
		To: []EmailTo{
			{Email: to},
		},
		Subject:     subject,
		HTMLContent: htmlContent,
		TextContent: textContent,
	}

	return s.sendEmail(emailReq)
}

// getProjectConfig gets project configuration
func (s *BrevoService) getProjectConfig(projectID string) *models.ProjectConfig {
	// Should get configuration from project manager
	// For now, return default configuration
	return &models.ProjectConfig{
		ProjectID:   projectID,
		ProjectName: "Default Project",
		FromEmail:   s.FromEmail,
		FromName:    s.FromName,
	}
}

// sendEmail sends email via Brevo API
func (s *BrevoService) sendEmail(req EmailRequest) error {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal email request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", "https://api.brevo.com/v3/sendEmail", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("api-key", s.APIKey)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("brevo API error: status %d", resp.StatusCode)
	}

	return nil
}
