package service

import (
	"net/smtp"
	"strings"
	"testing"
)

func TestSMTPWebhookClientSendsMailtoPayload(t *testing.T) {
	var address, from string
	var recipients []string
	var message string
	client := &SMTPWebhookClient{Host: "smtp.example", Port: "2525", From: "eye@example.com", Send: func(addr string, _ smtp.Auth, sender string, to []string, body []byte) error {
		address, from, recipients, message = addr, sender, to, string(body)
		return nil
	}}
	if err := client.PostJSON("mailto:ops@example.com, oncall@example.com", map[string]any{"rule_id": "r1"}); err != nil {
		t.Fatal(err)
	}
	if address != "smtp.example:2525" || from != "eye@example.com" || len(recipients) != 2 || !strings.Contains(message, `"rule_id":"r1"`) {
		t.Fatalf("unexpected SMTP delivery: %q %q %#v %q", address, from, recipients, message)
	}
}

func TestEmailAlertRequiresMailtoDestination(t *testing.T) {
	if err := validateAlertDestinationURL("https://ops.example/hook", "email"); err == nil {
		t.Fatal("expected email URL validation error")
	}
	if err := validateAlertDestinationURL("mailto:ops@example.com", "email"); err != nil {
		t.Fatal(err)
	}
}
