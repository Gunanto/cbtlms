package app

import (
	"os"
	"strconv"
)

// Config stores runtime configuration loaded from environment variables.
type Config struct {
	HTTPAddr           string
	DBDSN              string
	DefaultExamMinutes int

	SMTPHost      string
	SMTPPort      int
	SMTPUser      string
	SMTPPass      string
	SMTPFrom      string
	SMTPStartTLS  bool

	BootstrapToken string
}

func LoadConfig() Config {
	addr := envOrDefault("HTTP_ADDR", ":8080")
	dsn := envOrDefault("DB_DSN", "postgres://cbtlms:cbtlms_dev_password@localhost:5432/cbtlms?sslmode=disable")

	smtpPort := 587
	if p := stringsToInt(os.Getenv("SMTP_PORT")); p > 0 {
		smtpPort = p
	}

	smtpStartTLS := true
	if v := os.Getenv("SMTP_STARTTLS"); v != "" {
		smtpStartTLS = v == "1" || v == "true" || v == "TRUE"
	}

	return Config{
		HTTPAddr:           addr,
		DBDSN:              dsn,
		DefaultExamMinutes: 90,
		SMTPHost:           os.Getenv("SMTP_HOST"),
		SMTPPort:           smtpPort,
		SMTPUser:           os.Getenv("SMTP_USER"),
		SMTPPass:           os.Getenv("SMTP_PASS"),
		SMTPFrom:           envOrDefault("SMTP_FROM", "noreply@cbtlms.local"),
		SMTPStartTLS:       smtpStartTLS,
		BootstrapToken:     os.Getenv("BOOTSTRAP_TOKEN"),
	}
}

func envOrDefault(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func stringsToInt(v string) int {
	n, _ := strconv.Atoi(v)
	return n
}
