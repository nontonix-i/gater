package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"

	"github.com/user/gater/internal/provider"
	"github.com/user/gater/internal/task"
)

func NewRouter(
	db *gorm.DB,
	tm *task.Manager,
	reg *provider.Registry,
	authEnabled bool,
	webFS fs.FS,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(Logger)
	r.Use(chimiddleware.Recoverer)

	handler := NewHandler(tm, reg, db)

	// Public auth endpoints (no auth required)
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", handler.Register)
		r.Post("/login", handler.Login)
	})

	// Authenticated API endpoints
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(Auth(db, authEnabled))

		r.Get("/providers", handler.ListProviders)

		r.Post("/upload", handler.UploadFile)
		r.Post("/upload/url", handler.UploadURL)

		r.Get("/task/{id}", handler.GetTask)
		r.Get("/task/{id}/progress", handler.StreamProgress)
		r.Get("/tasks", handler.ListTasks)

		r.Post("/auth/credential", handler.SaveCredential)
		r.Get("/auth/credentials", handler.ListCredentials)
		r.Post("/auth/regenerate-key", handler.RegenerateKey)
		r.Get("/auth/me", handler.Me)

		r.Get("/settings", handler.GetSettings)
		r.Put("/settings", handler.UpdateSettings)

		r.Get("/providers/{name}", handler.GetProvider)
		r.Get("/providers/{name}/credentials", handler.GetProviderCredentials)
		r.Put("/providers/{name}/credentials", handler.UpdateProviderCredentials)
	})

	// Serve embedded SPA
	fileServer := http.FileServer(http.FS(webFS))

	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if strings.HasPrefix(path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// Try to serve the actual file
		if path != "/" {
			f, err := webFS.Open(strings.TrimPrefix(path, "/"))
			if err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// SPA fallback: serve index.html
		index, err := webFS.Open("index.html")
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		defer index.Close()

		stat, err := index.Stat()
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		data := make([]byte, stat.Size())
		_, err = index.Read(data)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	return r
}
