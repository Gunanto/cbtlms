package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"cbtlms/internal/app"
	"cbtlms/internal/db"
)

func main() {
	cfg := app.LoadConfig()

	dbConn, err := db.OpenPostgres(context.Background(), cfg.DBDSN)
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
