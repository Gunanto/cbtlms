package app

import (
	"database/sql"
	"html/template"
	"net/http"

	"cbtlms/internal/auth"
	"cbtlms/internal/exam"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(cfg Config, db *sql.DB) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)

	tmpl := template.Must(template.ParseGlob("web/templates/layout/*.html"))
	template.Must(tmpl.ParseGlob("web/templates/pages/*.html"))

	mailer := auth.NewSMTPMailer(auth.SMTPConfig{
		Host: cfg.SMTPHost,
		Port: cfg.SMTPPort,
		User: cfg.SMTPUser,
		Pass: cfg.SMTPPass,
		From: cfg.SMTPFrom,
	})

	authSvc := auth.NewService(db, auth.ServiceConfig{
		BootstrapToken: cfg.BootstrapToken,
		Mailer:         mailer,
	})
	authHandler := auth.NewHandler(authSvc)

	examSvc := exam.NewService(db, cfg.DefaultExamMinutes)
	examHandler := exam.NewHandler(examSvc)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"Title": "CBT LMS",
		}

		if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.Route("/api/v1", func(api chi.Router) {
		api.Post("/bootstrap/init", authHandler.BootstrapInit)
		api.Post("/auth/login-password", authHandler.LoginPassword)
		api.Post("/auth/otp/request", authHandler.RequestOTP)
		api.Post("/auth/otp/verify", authHandler.VerifyOTP)
		api.Post("/registrations", authHandler.CreateRegistration)

		api.Group(func(secure chi.Router) {
			secure.Use(authHandler.RequireAuth)
			secure.Get("/auth/me", authHandler.Me)
			secure.Post("/auth/logout", authHandler.Logout)

			secure.Post("/attempts/start", examHandler.Start)
			secure.Get("/attempts/{id}", examHandler.GetAttempt)
			secure.Get("/attempts/{id}/result", examHandler.Result)
			secure.Put("/attempts/{id}/answers/{questionID}", examHandler.SaveAnswer)
			secure.Post("/attempts/{id}/submit", examHandler.Submit)

			secure.Group(func(admin chi.Router) {
				admin.Use(authHandler.RequireRoles("admin", "proktor"))
				admin.Get("/admin/registrations", authHandler.ListPendingRegistrations)
				admin.Post("/admin/registrations/{id}/approve", authHandler.ApproveRegistration)
				admin.Post("/admin/registrations/{id}/reject", authHandler.RejectRegistration)
			})
		})
	})

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	return r
}
