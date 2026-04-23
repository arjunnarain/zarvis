package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/zarvis/internal/auth"
	"github.com/zarvis/internal/badges"
	"github.com/zarvis/internal/chat"
	"github.com/zarvis/internal/mcp"
	"github.com/zarvis/internal/prompt"
	"github.com/zarvis/internal/search"
	"github.com/zarvis/internal/state"
	"github.com/zarvis/internal/tools"
)

func main() {
	apiKey := os.Getenv("ZARVIS_API_KEY")
	if apiKey == "" {
		log.Fatal("ZARVIS_API_KEY is required")
	}

	dbPath := os.Getenv("ZARVIS_DB")
	if dbPath == "" {
		dbPath = "zarvis.db"
	}

	store, err := state.NewSQLiteStore(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer store.Close()

	docsDir := os.Getenv("ZARVIS_DOCS_DIR")
	if docsDir == "" {
		docsDir = "../docs"
	}

	registry, err := mcp.LoadRegistry(docsDir + "/mcp_tools.json")
	if err != nil {
		log.Fatalf("load tool registry: %v", err)
	}

	promptBuilder, err := prompt.NewBuilder(docsDir + "/prompts")
	if err != nil {
		log.Fatalf("load prompts: %v", err)
	}

	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if base := os.Getenv("ZARVIS_BASE_URL"); base != "" {
		opts = append(opts, option.WithBaseURL(base))
	}
	jwtSecret := os.Getenv("ZARVIS_JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "zarvis-dev-secret-change-in-prod"
	}

	anthropicClient := anthropic.NewClient(opts...)
	searchEngine := search.NewEngine(store)

	h := &chat.Handler{
		Anthropic: &anthropicClient,
		Store:     store,
		Registry:  registry,
		Prompt:    promptBuilder,
		Badges:    badges.New(store),
		Tools:     tools.NewExecutor(store, searchEngine),
		Search:    searchEngine,
		JWTSecret: jwtSecret,
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	allowedOrigins := []string{"http://localhost:5173", "http://localhost:8080"}
	if extra := os.Getenv("ZARVIS_CORS_ORIGIN"); extra != "" {
		allowedOrigins = append(allowedOrigins, extra)
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	}))

	// Public routes
	r.Get("/api/health", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	r.Post("/api/auth/register", h.Register)
	r.Post("/api/auth/login", h.Login)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(jwtSecret))
		r.Get("/api/auth/me", h.GetMe)
		r.Post("/api/session", h.CreateSession)
		r.Get("/api/session/{id}", h.GetSession)
		r.Patch("/api/session/{id}", h.UpdateSession)
		r.Get("/api/session/{id}/badges", h.GetBadges)
		r.Get("/api/session/{id}/tabs", h.GetTabs)
		r.Get("/api/session/{id}/search", h.SearchDocument)
		r.Get("/api/session/{id}/quality", h.GetQuality)
		r.Get("/api/session/{id}/document", h.GetDocument)
		r.Get("/api/session/{id}/documents", h.ListDocuments)
		r.Get("/api/session/{id}/export", h.Export)
		r.Get("/api/session/{id}/export-schema", h.ExportSchema)
		r.Post("/api/upload", h.Upload)
		r.Post("/api/sample", h.LoadSample)
		r.Post("/api/forest", h.CreateForest)
		r.Get("/api/session/{id}/forests", h.ListForests)
		r.Post("/api/forest/{id}/documents", h.AddDocToForest)
		r.Get("/api/forest/{id}/documents", h.GetForestDocs)
		r.Delete("/api/forest/{id}/documents", h.ClearForest)
		r.Post("/api/chat", h.Chat)
	})

	// Serve static frontend in production
	if staticDir := os.Getenv("ZARVIS_STATIC_DIR"); staticDir != "" {
		fs := http.FileServer(http.Dir(staticDir))
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			// Try file first, fall back to index.html for SPA routing
			if _, err := os.Stat(staticDir + r.URL.Path); err != nil {
				http.ServeFile(w, r, staticDir+"/index.html")
				return
			}
			fs.ServeHTTP(w, r)
		})
	}

	srv := &http.Server{Addr: ":8080", Handler: r, ReadHeaderTimeout: 5 * time.Second}

	go func() {
		log.Println("Zarvis listening on :8080")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
