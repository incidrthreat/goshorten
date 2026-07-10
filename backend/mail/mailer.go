// Package mail provides a minimal outbound-email abstraction for Phase 15
// invitations. It sends via SMTP when configured and otherwise logs the message
// (including any invite link), so the feature works out of the box without an
// SMTP server — the UI additionally surfaces a copyable accept link.
package mail

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"github.com/hashicorp/go-hclog"
)

var log = hclog.Default()

// Config holds SMTP settings. A zero Host disables SMTP and enables log mode.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// Mailer sends transactional email.
type Mailer struct {
	cfg Config
}

// New returns a Mailer. It is always usable: with no SMTP host it logs instead
// of sending.
func New(cfg Config) *Mailer {
	if cfg.From == "" {
		cfg.From = "no-reply@goshorten.local"
	}
	if cfg.Port == 0 {
		cfg.Port = 587
	}
	return &Mailer{cfg: cfg}
}

// Enabled reports whether real SMTP delivery is configured.
func (m *Mailer) Enabled() bool {
	return m != nil && strings.TrimSpace(m.cfg.Host) != ""
}

// Send delivers a plain-text email. When SMTP is not configured it logs the
// message and returns nil (non-fatal), so callers can always proceed.
func (m *Mailer) Send(to, subject, body string) error {
	if m == nil || !m.Enabled() {
		log.Info("Mail", "delivery disabled (no SMTP host); logging message",
			"to", to, "subject", subject)
		log.Debug("Mail", "body", body)
		return nil
	}

	addr := net.JoinHostPort(m.cfg.Host, fmt.Sprintf("%d", m.cfg.Port))
	msg := buildMessage(m.cfg.From, to, subject, body)

	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}

	if m.cfg.Port == 465 {
		// Implicit TLS.
		return m.sendTLS(addr, auth, to, msg)
	}
	// STARTTLS (or plain on non-TLS servers) via the stdlib helper, which issues
	// STARTTLS automatically when the server advertises it.
	if err := smtp.SendMail(addr, auth, m.cfg.From, []string{to}, msg); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}

// sendTLS handles implicit-TLS (port 465) delivery.
func (m *Mailer) sendTLS(addr string, auth smtp.Auth, to string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: m.cfg.Host})
	if err != nil {
		return fmt.Errorf("smtp tls dial: %w", err)
	}
	c, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer func() { _ = c.Quit() }()

	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := c.Mail(m.cfg.From); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}
	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := wc.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	return wc.Close()
}

func buildMessage(from, to, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return []byte(b.String())
}

// SendInvitation sends a workspace invite email with the accept link.
func (m *Mailer) SendInvitation(to, workspaceName, inviterEmail, acceptURL string) error {
	inviter := inviterEmail
	if inviter == "" {
		inviter = "A workspace admin"
	}
	subject := fmt.Sprintf("You've been invited to the %q workspace on GoShorten", workspaceName)
	body := fmt.Sprintf(
		"Hi,\n\n%s has invited you to join the %q workspace on GoShorten.\n\n"+
			"Accept the invitation by opening this link:\n\n%s\n\n"+
			"If you don't have an account yet, you'll be able to create one first.\n"+
			"This invitation expires in 7 days.\n\n"+
			"If you weren't expecting this, you can ignore this email.\n",
		inviter, workspaceName, acceptURL)
	return m.Send(to, subject, body)
}
