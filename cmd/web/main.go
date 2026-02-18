package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"cbtlms/internal/app"
	"cbtlms/internal/db"
)

func main() {
	cfg := app.LoadConfig()

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
