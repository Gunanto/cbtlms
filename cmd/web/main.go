package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"cbtlms/internal/app"
	"cbtlms/internal/db"
)

func isProductionEnv(v string) bool {
	env := strings.TrimSpace(strings.ToLower(v))
	return env == "prod" || env == "production"
}

func enforceSecurityBaseline(cfg app.Config) {
	dsnLower := strings.ToLower(cfg.DBDSN)
	prod := isProductionEnv(cfg.AppEnv)

	if strings.Contains(dsnLower, "sslmode=disable") {
		msg := "DB_DSN still uses sslmode=disable"
		if prod {
			log.Printf("security error: %s in APP_ENV=%s", msg, cfg.AppEnv)
			os.Exit(1)
		}
		log.Printf("security warning: %s (allowed in non-production)", msg)
	}

	if strings.Contains(cfg.DBDSN, "cbtlms_dev_password") {
		msg := "DB_DSN still uses default development password"
		if prod {
			log.Printf("security error: %s in APP_ENV=%s", msg, cfg.AppEnv)
			os.Exit(1)
		}
		log.Printf("security warning: %s", msg)
	}

	if !cfg.CSRFEnforced {
		msg := "CSRF_ENFORCED is disabled"
		if prod {
			log.Printf("security error: %s in APP_ENV=%s", msg, cfg.AppEnv)
			os.Exit(1)
		}
		log.Printf("security warning: %s", msg)
	}
}

func main() {
	cfg := app.LoadConfig()
	enforceSecurityBaseline(cfg)

	dbConn, err := db.OpenPostgresWithConfig(context.Background(), cfg.DBDSN, db.PostgresConfig{
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxLifetime: time.Duration(cfg.DBConnMaxLifeMins) * time.Minute,
	})
	if err != nil {
		log.Printf("database error: %v", err)
		os.Exit(1)
	}
	defer dbConn.Close()

	r := app.NewRouter(cfg, dbConn)

	log.Printf("cbtlms web listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, r); err != nil {
		log.Printf("server stopped: %v", err)
		os.Exit(1)
	}
}
