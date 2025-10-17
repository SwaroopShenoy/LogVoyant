package server

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"logvoyant/internal/analyzer"
	"logvoyant/internal/storage"
)

type Config struct {
	Port        int
	Storage     storage.Storage
	StaticFiles embed.FS
	GroqAPIKey  string
}

type Server struct {
	config   *Config
	router   *chi.Mux
	server   *http.Server
	analyzer *analyzer.Analyzer
	hub      *WebSocketHub
}

func New(cfg *Config) *Server {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
	}))

	// Initialize analyzer
	anlz := analyzer.New(&analyzer.Config{
		Storage:    cfg.Storage,
		GroqAPIKey: cfg.GroqAPIKey,
	})

	// Initialize WebSocket hub
	hub := NewWebSocketHub()
	go hub.Run()

	srv := &Server{
		config:   cfg,
		router:   r,
		analyzer: anlz,
		hub:      hub,
	}

	srv.setupRoutes()

	return srv
}

func (s *Server) setupRoutes() {
	// Static files (UI)
	staticFS, err := fs.Sub(s.config.StaticFiles, "internal/web/ui")
	if err != nil {
		panic(err)
	}
	s.router.Handle("/*", http.FileServer(http.FS(staticFS)))

	// API routes
	s.router.Route("/api", func(r chi.Router) {
		r.Get("/streams", s.handleListStreams)
		r.Get("/streams/{id}", s.handleGetStream)
		r.Get("/streams/{id}/logs", s.handleGetLogs)
		r.Post("/streams/{id}/analyze", s.handleAnalyze)
		r.Get("/streams/{id}/context", s.handleGetContext)
		r.Post("/streams/{id}/resolve", s.handleResolve)
	})

	// WebSocket
	s.router.Get("/ws/streams/{id}", s.handleWebSocket)
}

func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Port),
		Handler: s.router,
	}
	return s.server.ListenAndServe()
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

func (s *Server) Hub() *WebSocketHub {
	return s.hub
}