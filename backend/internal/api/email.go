package api

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"kept/internal/models"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/gomail.v2"
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

	// Check if file exists
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("template file not found at %s", templatePath)
	}

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse email template: %w", err)
	}
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
		return "", fmt.Errorf("failed to execute email template: %w", err)
	}

	return buf.String(), nil
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

// sendSMTPEmail sends an email via SMTP using gomail
func sendSMTPEmail(config *SMTPConfig, to, subject, htmlBody string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", config.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)

	// Plain text fallback
	m.SetBody("text/plain", "Please view this email in an HTML-capable email client.")

	// HTML version (this is the important part for HTML rendering)
	m.AddAlternative("text/html", htmlBody)

	// Create dialer with SMTP config
	d := gomail.NewDialer(config.Host, config.Port, config.Username, config.Password)

	// Send the email
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

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
