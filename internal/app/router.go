package app

import (
	"database/sql"
	"html/template"
	"net/http"
	"time"

	"cbtlms/internal/app/observability"
	"cbtlms/internal/auth"
	"cbtlms/internal/exam"
	"cbtlms/internal/masterdata"
	"cbtlms/internal/question"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(cfg Config, db *sql.DB) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	obs := observability.NewCollector(db)
	r.Use(obs.Middleware)
	authRateLimiter := NewIPRateLimiter(cfg.AuthRateLimitPerMin, time.Minute)

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
	masterSvc := masterdata.NewService(db)
	masterHandler := masterdata.NewHandler(masterSvc)
	questionSvc := question.NewService(db)
	questionHandler := question.NewHandler(questionSvc)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	r.Get("/metrics", obs.MetricsHandler)

	renderPage := func(w http.ResponseWriter, contentTemplate, pageTitle string, extra map[string]any) {
		data := map[string]any{
			"Title":           pageTitle,
			"Page":            contentTemplate,
			"ContentTemplate": contentTemplate,
		}
		for k, v := range extra {
			data[k] = v
		}
		if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		renderPage(w, "home_content", "CBT LMS", map[string]any{})
	})
	r.Get("/login", func(w http.ResponseWriter, r *http.Request) {
		renderPage(w, "login_content", "Login CBT LMS", map[string]any{})
	})
	r.Get("/simulasi", func(w http.ResponseWriter, r *http.Request) {
		renderPage(w, "simulasi_content", "Pilih Simulasi", map[string]any{})
	})
	r.Get("/authoring", func(w http.ResponseWriter, r *http.Request) {
		renderPage(w, "authoring_content", "Authoring Soal", map[string]any{})
	})
	r.Get("/ujian/{id}", func(w http.ResponseWriter, r *http.Request) {
		attemptID := chi.URLParam(r, "id")
		renderPage(w, "attempt_content", "Pengerjaan Ujian", map[string]any{
			"AttemptID": attemptID,
		})
	})
	r.Get("/hasil/{id}", func(w http.ResponseWriter, r *http.Request) {
		attemptID := chi.URLParam(r, "id")
		renderPage(w, "result_content", "Hasil Ujian", map[string]any{
			"AttemptID": attemptID,
		})
	})

	r.Route("/api/v1", func(api chi.Router) {
		api.Post("/bootstrap/init", authHandler.BootstrapInit)
		api.With(RateLimitMiddleware(authRateLimiter)).Post("/auth/login-password", authHandler.LoginPassword)
		api.With(RateLimitMiddleware(authRateLimiter)).Post("/auth/otp/request", authHandler.RequestOTP)
		api.With(RateLimitMiddleware(authRateLimiter)).Post("/auth/otp/verify", authHandler.VerifyOTP)
		api.With(RateLimitMiddleware(authRateLimiter)).Post("/registrations", authHandler.CreateRegistration)

		api.Group(func(secure chi.Router) {
			secure.Use(authHandler.RequireAuth)
			secure.Use(CSRFMiddleware(cfg.CSRFEnforced))
			secure.Get("/auth/me", authHandler.Me)
			secure.Post("/auth/logout", authHandler.Logout)
			secure.Get("/subjects", examHandler.ListSubjects)
			secure.Get("/exams", examHandler.ListExams)

			secure.Post("/attempts/start", examHandler.Start)
			secure.Get("/attempts/{id}", examHandler.GetAttempt)
			secure.Get("/attempts/{id}/questions/{no}", examHandler.GetAttemptQuestion)
			secure.Get("/attempts/{id}/result", examHandler.Result)
			secure.Post("/attempts/{id}/events", examHandler.LogEvent)
			secure.Put("/attempts/{id}/answers/{questionID}", examHandler.SaveAnswer)
			secure.Post("/attempts/{id}/submit", examHandler.Submit)
			secure.With(authHandler.RequireRoles("admin", "proktor")).Get("/attempts/{id}/events", examHandler.ListEvents)
			secure.With(authHandler.RequireRoles("admin", "proktor", "guru")).Post("/stimuli", questionHandler.CreateStimulus)
			secure.With(authHandler.RequireRoles("admin", "proktor", "guru")).Get("/stimuli", questionHandler.ListStimuli)
			secure.With(authHandler.RequireRoles("admin", "proktor", "guru")).Post("/questions/{id}/versions", questionHandler.CreateQuestionVersion)
			secure.With(authHandler.RequireRoles("admin", "proktor", "guru")).Post("/questions/{id}/versions/{version}/finalize", questionHandler.FinalizeQuestionVersion)
			secure.With(authHandler.RequireRoles("admin", "proktor", "guru")).Get("/questions/{id}/versions", questionHandler.ListQuestionVersions)
			secure.With(authHandler.RequireRoles("admin", "proktor", "guru")).Post("/exams/{id}/parallels", questionHandler.CreateQuestionParallel)
			secure.With(authHandler.RequireRoles("admin", "proktor", "guru")).Get("/exams/{id}/parallels", questionHandler.ListQuestionParallels)
			secure.With(authHandler.RequireRoles("admin", "proktor", "guru")).Put("/exams/{id}/parallels/{parallelID}", questionHandler.UpdateQuestionParallel)
			secure.With(authHandler.RequireRoles("admin", "proktor", "guru")).Delete("/exams/{id}/parallels/{parallelID}", questionHandler.DeleteQuestionParallel)
			secure.With(authHandler.RequireRoles("admin", "proktor", "guru")).Post("/reviews/tasks", questionHandler.CreateReviewTask)
			secure.With(authHandler.RequireRoles("admin", "proktor", "guru")).Post("/reviews/tasks/{id}/decision", questionHandler.DecideReviewTask)
			secure.With(authHandler.RequireRoles("admin", "proktor", "guru")).Get("/reviews/tasks", questionHandler.ListReviewTasks)
			secure.With(authHandler.RequireRoles("admin", "proktor", "guru")).Get("/questions/{id}/reviews", questionHandler.GetQuestionReviews)

			secure.Group(func(admin chi.Router) {
				admin.Use(authHandler.RequireRoles("admin", "proktor"))
				admin.Get("/admin/registrations", authHandler.ListPendingRegistrations)
				admin.Post("/admin/registrations/{id}/approve", authHandler.ApproveRegistration)
				admin.Post("/admin/registrations/{id}/reject", authHandler.RejectRegistration)
				admin.Post("/admin/schools", masterHandler.CreateSchool)
				admin.Post("/admin/classes", masterHandler.CreateClass)
				admin.Post("/admin/imports/students", masterHandler.ImportStudentsCSV)
			})
		})
	})

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	return r
}
