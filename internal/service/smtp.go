package service

import (
	"encoding/json"
	"fmt"
	"net/smtp"
	"net/url"
	"strings"
)

// SMTPWebhookClient adapts alert payloads to SMTP delivery. Alert URLs use
// mailto:user@example.com (optionally comma-separated recipients) when the
// alert destination is email.
type SMTPWebhookClient struct {
	Host     string
	Port     string
	From     string
	User     string
	Password string
	Send     func(addr string, auth smtp.Auth, from string, to []string, message []byte) error
}

func (c *SMTPWebhookClient) PostJSON(rawURL string, payload map[string]any) error {
	if c.Host == "" || c.From == "" {
		return fmt.Errorf("SMTP email delivery is not configured")
	}
	raw := strings.TrimPrefix(strings.TrimSpace(rawURL), "mailto:")
	if raw == "" {
		return fmt.Errorf("email destination must be mailto:user@example.com")
	}
	recipients := strings.Split(raw, ",")
	for i := range recipients {
		recipients[i] = strings.TrimSpace(recipients[i])
	}
	if len(recipients) == 0 || recipients[0] == "" {
		return fmt.Errorf("email destination has no recipients")
	}
	body, _ := json.Marshal(payload)
	message := []byte("From: " + c.From + "\r\nTo: " + strings.Join(recipients, ", ") + "\r\nSubject: Eye alert\r\nContent-Type: application/json\r\n\r\n" + string(body) + "\r\n")
	sender := c.Send
	if sender == nil {
		sender = smtp.SendMail
	}
	var auth smtp.Auth
	if c.User != "" {
		auth = smtp.PlainAuth("", c.User, c.Password, strings.Split(c.Host, ":")[0])
	}
	port := c.Port
	if port == "" {
		port = "25"
	}
	return sender(c.Host+":"+port, auth, c.From, recipients, message)
}

func validateAlertDestinationURL(raw, destination string) error {
	if destination == "email" {
		if strings.HasPrefix(strings.TrimSpace(raw), "mailto:") {
			return nil
		}
		return fmt.Errorf("email destination must be a mailto: URL")
	}
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("webhook_url must be an absolute http(s) URL")
	}
	return nil
}
