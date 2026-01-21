package api

import (
	"bytes"
	"crypto/tls"
	"database/sql"
	"fmt"
	"html/template"
	"kept/internal/models"
	"log"
	"net/smtp"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type EmailReminderData struct {
	RecipientName string
	Title         string
	Description   string
	DueDate       string
	DueDateRaw    *time.Time
	PromiseTo     string
	AppURL        string
	PromiseID     int
	Year          int
}

// LoadEmailTemplate loads and parses the email template
func LoadEmailTemplate() (*template.Template, error) {
	templatePath := filepath.Join(".", "reminder-email-template.html")
	log.Printf("[EMAIL] Loading template from: %s", templatePath)
	
	// Check if file exists
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		log.Printf("[EMAIL] Template file not found at %s", templatePath)
		return nil, fmt.Errorf("template file not found at %s", templatePath)
	}
	
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		log.Printf("[EMAIL] Failed to parse template: %v", err)
		return nil, fmt.Errorf("failed to parse email template: %w", err)
	}
	log.Printf("[EMAIL] Template loaded successfully")
	return tmpl, nil
}

// GenerateReminderEmail generates an HTML email for a promise reminder
func GenerateReminderEmail(promise models.Promise, appURL string) (string, error) {
	tmpl, err := LoadEmailTemplate()
	if err != nil {
		return "", err
	}

	// Format due date nicely
	dueDateStr := "No due date set"
	if promise.DueDate != nil {
		dueDateStr = formatEmailDate(*promise.DueDate)
	}

	data := EmailReminderData{
		RecipientName: "there", // Could be extracted from user data if available
		Title:         promise.Description,
		Description:   promise.Description,
		DueDate:       dueDateStr,
		DueDateRaw:    promise.DueDate,
		PromiseTo:     promise.Recipient,
		AppURL:        appURL,
		PromiseID:     promise.ID,
		Year:          time.Now().Year(),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Printf("[EMAIL] Failed to execute template: %v", err)
		return "", fmt.Errorf("failed to execute email template: %w", err)
	}

	htmlContent := buf.String()
	log.Printf("[EMAIL] Generated HTML content, length: %d bytes, first 200 chars: %.200s", len(htmlContent), htmlContent)
	return htmlContent, nil
}

// formatEmailDate formats a time.Time for email display
func formatEmailDate(t time.Time) string {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	thatDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())

	diffDays := int(thatDay.Sub(today).Hours() / 24)

	switch {
	case diffDays == 0:
		return fmt.Sprintf("Today at %s", t.Format("3:04 PM"))
	case diffDays == 1:
		return fmt.Sprintf("Tomorrow at %s", t.Format("3:04 PM"))
	case diffDays == -1:
		return fmt.Sprintf("Yesterday at %s", t.Format("3:04 PM"))
	case diffDays > 1 && diffDays < 7:
		return fmt.Sprintf("In %d days (%s)", diffDays, t.Format("Jan 2"))
	case diffDays < -1 && diffDays > -7:
		return fmt.Sprintf("%d days ago (%s)", -diffDays, t.Format("Jan 2"))
	default:
		if t.Year() == now.Year() {
			return t.Format("January 2")
		}
		return t.Format("January 2, 2006")
	}
}

// SMTPConfig holds SMTP server configuration
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	UseTLS   bool
}

// GetSMTPConfig reads SMTP configuration from environment variables
func GetSMTPConfig() (*SMTPConfig, error) {
	host := os.Getenv("SMTP_HOST")
	portStr := os.Getenv("SMTP_PORT")
	username := os.Getenv("SMTP_USER")
	password := os.Getenv("SMTP_PASS")
	from := os.Getenv("SMTP_FROM")
	useTLSStr := os.Getenv("SMTP_USE_TLS")

	if host == "" {
		return nil, fmt.Errorf("SMTP_HOST not configured")
	}

	port := 587 // Default SMTP port
	if portStr != "" {
		var err error
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid SMTP_PORT: %w", err)
		}
	}

	if from == "" {
		from = "noreply@kept.app"
	}

	useTLS := true
	if useTLSStr != "" {
		useTLS = strings.ToLower(useTLSStr) != "false"
	}

	return &SMTPConfig{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
		UseTLS:   useTLS,
	}, nil
}

// SendReminderEmail sends an email reminder using configured SMTP server
func SendReminderEmail(db *sql.DB, promise models.Promise, userEmail string) error {
	config, err := GetSMTPConfig()
	if err != nil {
		log.Printf("SMTP not configured, skipping email: %v", err)
		return nil // Don't fail if email isn't configured
	}

	appURL := getAppURL()
	htmlContent, err := GenerateReminderEmail(promise, appURL)
	if err != nil {
		return fmt.Errorf("failed to generate email: %w", err)
	}

	return sendSMTPEmail(config, userEmail, "Promise Reminder from Kept", htmlContent)
}

// sendSMTPEmail sends an email via SMTP
func sendSMTPEmail(config *SMTPConfig, to, subject, htmlBody string) error {
	log.Printf("[EMAIL] Sending email to %s, subject: %s, HTML body length: %d", to, subject, len(htmlBody))
	
	// Build email message with proper MIME multipart format
	boundary := "----=_Part_0_1234567890.1234567890"
	
	message := fmt.Sprintf("From: %s\r\n", config.From)
	message += fmt.Sprintf("To: %s\r\n", to)
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "MIME-Version: 1.0\r\n"
	message += fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary)
	message += "\r\n"
	
	// Plain text version
	message += fmt.Sprintf("--%s\r\n", boundary)
	message += "Content-Type: text/plain; charset=UTF-8\r\n"
	message += "Content-Transfer-Encoding: 7bit\r\n"
	message += "\r\n"
	message += "Please view this email in an HTML-capable email client.\r\n"
	message += "\r\n"
	
	// HTML version
	message += fmt.Sprintf("--%s\r\n", boundary)
	message += "Content-Type: text/html; charset=UTF-8\r\n"
	message += "Content-Transfer-Encoding: 7bit\r\n"
	message += "\r\n"
	message += htmlBody
	message += "\r\n"
	message += fmt.Sprintf("--%s--\r\n", boundary)

	log.Printf("[EMAIL] Full message length: %d bytes, first 500 chars: %.500s", len(message), message)

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	var auth smtp.Auth
	if config.Username != "" && config.Password != "" {
		auth = smtp.PlainAuth("", config.Username, config.Password, config.Host)
	}

	// Use TLS if configured
	if config.UseTLS {
		return sendMailTLS(addr, auth, config.From, []string{to}, []byte(message), config.Host)
	}

	// Standard SMTP without TLS
	return smtp.SendMail(addr, auth, config.From, []string{to}, []byte(message))
}

// sendMailTLS sends email with TLS encryption
func sendMailTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte, host string) error {
	// Connect to server
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	// Start TLS
	tlsConfig := &tls.Config{
		ServerName: host,
	}
	if err = client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// Authenticate
	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	// Set sender
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipient
	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient: %w", err)
		}
	}

	// Send message
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}
	defer w.Close()

	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	// Future: Add support for other email providers:
	// - SendGrid API
	// - AWS SES
	// - Mailgun
	// - Postmark
	// etc.

	return nil
}

// getAppURL returns the application URL from environment or default
func getAppURL() string {
	url := os.Getenv("APP_URL")
	if url == "" {
		url = "http://localhost:3000"
	}
	return url
}

// Example: Extend reminder sending to include email option
func SendReminderWithEmail(db *sql.DB, promise models.Promise, userID int, sendEmail bool) error {
	// Send push notification (existing behavior)
	payload := PushPayload{
		Title: fmt.Sprintf("Reminder about your promise to: %s", promise.Recipient),
		Body:  promise.Description,
		Icon:  "/Static/logos/Kept Mascot Colored.svg",
		Badge: "/Static/logos/Kept Mascot Colored.svg",
		Tag:   fmt.Sprintf("kept-reminder-%d", promise.ID),
		Data:  map[string]interface{}{"promise_id": promise.ID},
	}

	if err := SendPushToUser(db, userID, payload); err != nil {
		log.Printf("Failed to send push notification: %v", err)
	}

	// Optionally send email if enabled
	if sendEmail {
		// Get user email from database
		var email sql.NullString
		err := db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&email)
		if err == nil && email.Valid && email.String != "" {
			if err := SendReminderEmail(db, promise, email.String); err != nil {
				log.Printf("Failed to send email reminder: %v", err)
			}
		}
	}

	return nil
}
