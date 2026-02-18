package auth

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

type OTPMailer interface {
	SendOTP(ctx context.Context, email, code string) error
}

type SMTPMailer struct {
	host string
	port int
	user string
	pass string
	from string
}

type SMTPConfig struct {
	Host string
	Port int
	User string
	Pass string
	From string
}

func NewSMTPMailer(cfg SMTPConfig) OTPMailer {
	if strings.TrimSpace(cfg.Host) == "" || cfg.Port <= 0 || strings.TrimSpace(cfg.From) == "" {
		return nil
	}
	return &SMTPMailer{
		host: strings.TrimSpace(cfg.Host),
		port: cfg.Port,
		user: strings.TrimSpace(cfg.User),
		pass: cfg.Pass,
		from: strings.TrimSpace(cfg.From),
	}
}

func (m *SMTPMailer) SendOTP(ctx context.Context, email, code string) error {
	_ = ctx
	addr := fmt.Sprintf("%s:%d", m.host, m.port)

	subject := "Kode OTP Login CBT LMS"
	body := fmt.Sprintf("Kode OTP Anda: %s\nBerlaku 5 menit. Jangan bagikan kode ini.", code)
	msg := "From: " + m.from + "\r\n" +
		"To: " + email + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		body + "\r\n"

	var auth smtp.Auth
	if m.user != "" {
		auth = smtp.PlainAuth("", m.user, m.pass, m.host)
	}

	if err := smtp.SendMail(addr, auth, m.from, []string{email}, []byte(msg)); err != nil {
		return fmt.Errorf("smtp send otp: %w", err)
	}
	return nil
}
