package app

import (
	"os"
	"strconv"
	"strings"
)

// Config stores runtime configuration loaded from environment variables.
type Config struct {
	AppEnv              string
	HTTPAddr            string
	DBDSN               string
	DefaultExamMinutes  int
	DBMaxOpenConns      int
	DBMaxIdleConns      int
	DBConnMaxLifeMins   int
	CSRFEnforced        bool
	AuthRateLimitPerMin int

	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPass     string
	SMTPFrom     string
	SMTPStartTLS bool

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
		AppEnv:              envOrDefault("APP_ENV", "development"),
		HTTPAddr:            addr,
		DBDSN:               dsn,
		DefaultExamMinutes:  90,
		DBMaxOpenConns:      intOrDefault("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:      intOrDefault("DB_MAX_IDLE_CONNS", 25),
		DBConnMaxLifeMins:   intOrDefault("DB_CONN_MAX_LIFETIME_MINUTES", 30),
		CSRFEnforced:        boolOrDefault("CSRF_ENFORCED", false),
		AuthRateLimitPerMin: intOrDefault("AUTH_RATE_LIMIT_PER_MINUTE", 60),
		SMTPHost:            os.Getenv("SMTP_HOST"),
		SMTPPort:            smtpPort,
		SMTPUser:            os.Getenv("SMTP_USER"),
		SMTPPass:            os.Getenv("SMTP_PASS"),
		SMTPFrom:            envOrDefault("SMTP_FROM", "noreply@cbtlms.local"),
		SMTPStartTLS:        smtpStartTLS,
		BootstrapToken:      os.Getenv("BOOTSTRAP_TOKEN"),
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

func intOrDefault(key string, fallback int) int {
	v := stringsToInt(os.Getenv(key))
	if v <= 0 {
		return fallback
	}
	return v
}

func boolOrDefault(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}
