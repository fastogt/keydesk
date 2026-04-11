package api

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"keydesk/app/api/handlers"
	"keydesk/app/database"
	"keydesk/app/vault"
	"keydesk/app/version"
)

type ServerConfig struct {
	Database  *database.Database
	JWTSecret string
	Vault     *vault.Vault
}

type Server struct {
	auth        *handlers.AuthHandler
	people      *handlers.PeopleHandler
	accounts    *handlers.AccountsHandler
	services    *handlers.ServicesHandler
	credentials *handlers.CredentialsHandler
	assignments *handlers.AssignmentsHandler
	dashboard   *handlers.DashboardHandler
	settings    *handlers.SettingsHandler
	ext         *handlers.ExtHandler

	templates map[string]*template.Template
}

func NewServer(cfg ServerConfig) *Server {
	s := &Server{}
	log.Info("[Server] Initializing...")

	s.auth = handlers.NewAuthHandler(cfg.Database, cfg.JWTSecret)
	s.people = handlers.NewPeopleHandler(cfg.Database, cfg.Vault)
	s.accounts = handlers.NewAccountsHandler(cfg.Database, cfg.Vault)
	s.services = handlers.NewServicesHandler(cfg.Database)
	s.credentials = handlers.NewCredentialsHandler(cfg.Database, cfg.Vault)
	s.assignments = handlers.NewAssignmentsHandler(cfg.Database)
	s.dashboard = handlers.NewDashboardHandler(cfg.Database)
	s.settings = handlers.NewSettingsHandler(cfg.Database)
	s.ext = handlers.NewExtHandler(cfg.Database, cfg.JWTSecret, cfg.Vault)

	s.templates = make(map[string]*template.Template)
	staticDir := getStaticDir()
	htmlFiles, _ := filepath.Glob(filepath.Join(staticDir, "*.html"))
	for _, f := range htmlFiles {
		name := filepath.Base(f)
		tmpl, err := template.ParseFiles(f)
		if err != nil {
			log.Warnf("Failed to parse template %s: %v", name, err)
			continue
		}
		s.templates[name] = tmpl
	}

	log.Info("[Server] Initialized successfully")
	return s
}

func (s *Server) Stop() {
	log.Info("[Server] Stopped")
}

func getStaticDir() string {
	return version.ShareFolderPath + "/install/public"
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link", "X-Refresh-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/login", s.auth.HandleLogin)

		// Extension API (separate auth — employee JWT)
		r.Route("/ext", func(r chi.Router) {
			r.Post("/auth", s.ext.HandleExtLogin)

			r.Group(func(r chi.Router) {
				r.Use(s.ext.ExtAuthMiddleware)
				r.Get("/accounts", s.ext.HandleExtListAccounts)
				r.Post("/credentials/{id}", s.ext.HandleExtGetCredentials)
				r.Get("/match", s.ext.HandleExtMatch)
				r.Post("/totp/{id}", s.ext.HandleExtGetTOTP)
				r.Post("/audit", s.ext.HandleExtAudit)
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(s.auth.AdminAuthMiddleware)

			r.Post("/auth/logout", s.auth.HandleLogout)
			r.Get("/me", s.auth.HandleMe)

			// Dashboard
			r.Get("/dashboard", s.dashboard.HandleGetDashboard)

			// People
			r.Get("/people", s.people.HandleList)
			r.Post("/people", s.people.HandleCreate)
			r.Get("/people/{id}", s.people.HandleGet)
			r.Put("/people/{id}", s.people.HandleUpdate)
			r.Delete("/people/{id}", s.people.HandleDelete)
			r.Post("/people/{id}/offboard", s.people.HandleOffboard)

			// Accounts
			r.Get("/accounts", s.accounts.HandleList)
			r.Post("/accounts", s.accounts.HandleCreate)
			r.Get("/accounts/{id}", s.accounts.HandleGet)
			r.Put("/accounts/{id}", s.accounts.HandleUpdate)
			r.Delete("/accounts/{id}", s.accounts.HandleDelete)
			r.Post("/accounts/{id}/reveal", s.accounts.HandleReveal)
			r.Post("/accounts/{id}/rotate", s.accounts.HandleRotate)
			r.Post("/accounts/{id}/totp", s.accounts.HandleTOTP)

			// Services
			r.Get("/services", s.services.HandleList)
			r.Post("/services", s.services.HandleCreate)
			r.Get("/services/{id}", s.services.HandleGet)
			r.Put("/services/{id}", s.services.HandleUpdate)
			r.Delete("/services/{id}", s.services.HandleDelete)

			// Credentials
			r.Post("/credentials", s.credentials.HandleCreate)
			r.Put("/credentials/{id}", s.credentials.HandleUpdate)
			r.Delete("/credentials/{id}", s.credentials.HandleDelete)
			r.Post("/credentials/{id}/reveal", s.credentials.HandleReveal)
			r.Post("/credentials/{id}/rotate", s.credentials.HandleRotate)

			// Assignments
			r.Post("/assignments", s.assignments.HandleAssign)
			r.Delete("/assignments/{id}", s.assignments.HandleRevoke)

			// Settings
			r.Get("/settings/profile", s.settings.HandleGetProfile)
			r.Put("/settings/profile", s.settings.HandleUpdateProfile)
			r.Put("/settings/password", s.settings.HandleUpdatePassword)
			r.Post("/settings/import", s.settings.HandleImport)
			r.Get("/settings/export", s.settings.HandleExport)
		})
	})

	s.fileServer(r, getStaticDir())

	return r
}

func (s *Server) serveTemplate(w http.ResponseWriter, filename string) {
	tmpl := s.templates[filepath.Base(filename)]
	if tmpl == nil {
		log.Errorf("Template not found: %s", filename)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, nil); err != nil {
		log.Errorf("Failed to execute template %s: %v", filename, err)
	}
}

func (s *Server) fileServer(r chi.Router, staticDir string) {
	root := http.Dir(staticDir)

	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		path := req.URL.Path

		if path != "/" {
			fullPath := filepath.Join(staticDir, path)
			if _, err := os.Stat(fullPath); err == nil {
				if strings.HasSuffix(path, ".html") {
					s.serveTemplate(w, path)
					return
				}
				http.FileServer(root).ServeHTTP(w, req)
				return
			}
		}

		if path == "/" || path == "/setup" {
			s.serveTemplate(w, "setup.html")
			return
		}

		if path == "/dashboard" {
			s.serveTemplate(w, "dashboard.html")
			return
		}

		if path == "/people" {
			s.serveTemplate(w, "people.html")
			return
		}

		if path == "/accounts" {
			s.serveTemplate(w, "accounts.html")
			return
		}

		if path == "/services" {
			s.serveTemplate(w, "services.html")
			return
		}

		if path == "/settings" {
			s.serveTemplate(w, "settings.html")
			return
		}

		http.NotFound(w, req)
	})
}
