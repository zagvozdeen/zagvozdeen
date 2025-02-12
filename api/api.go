package api

import (
	"errors"
	"fmt"
	"github.com/zagvozdeen/zagvozdeen/config"
	"log/slog"
	"net/http"
	"os"
)

type Application struct {
	config config.Config
	logger *slog.Logger
}

func New(cfg config.Config) *Application {
	return &Application{
		config: cfg,
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}

func (a *Application) Run() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /blog/{slug}/", func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		bv, err := os.ReadFile("dist/version")
		if err != nil {
			a.logger.Error("Failed to read version file", "err", err, "slug", slug)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		version := string(bv)
		html, err := os.ReadFile(fmt.Sprintf("dist/%s/%s/index.html", version, slug))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				a.logger.Warn("File not found", "err", err, "slug", slug, "version", version)
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			a.logger.Error("Failed to read file", "err", err, "slug", slug, "version", version)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Add("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(html)
		if err != nil {
			a.logger.Error("Failed to write response", "err", err, "slug", slug, "version", version)
			return
		}
	})
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("dist/assets"))))

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	a.logger.Info("Starting server", "addr", server.Addr)
	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		a.logger.Error("Failed to start server", "err", err)
	}
}
