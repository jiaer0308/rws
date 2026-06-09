package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"backend/internal/database"
	"backend/internal/reserved_fund"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	log.Println("Starting server setup...")

	// Create context with timeout for db connection check
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to database (continues even if database is temporarily down during startup)
	dbPool, err := database.ConnectDB(ctx)
	if err != nil {
		log.Printf("Warning: Database connection failed: %v. Continuing offline.", err)
	} else {
		defer dbPool.Close()
		log.Println("Database connection established successfully.")

		// Run database schema migrations
		if err := database.RunMigrations(ctx, dbPool); err != nil {
			log.Printf("Warning: Database migrations failed: %v", err)
		} else {
			log.Println("Database migrations executed successfully.")
		}
	}

	r := chi.NewRouter()

	// Standard middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS configuration for local development
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// Basic health check route
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := map[string]string{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		}
		if database.Pool != nil {
			status["database"] = "connected"
		} else {
			status["database"] = "disconnected"
		}
		json.NewEncoder(w).Encode(status)
	})

	// Register reserved funds API routes
	if dbPool != nil {
		rfRepo := reserved_fund.NewRepository(dbPool)
		rfService := reserved_fund.NewService(rfRepo)
		rfHandler := reserved_fund.NewHandler(rfService, rfRepo)
		rfHandler.RegisterRoutes(r)
		log.Println("Reserved Fund routes registered successfully.")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
