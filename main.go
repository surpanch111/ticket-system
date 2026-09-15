package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"

	"ticket-system/internal/database"
	"ticket-system/internal/handlers"
	"ticket-system/internal/middleware"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	// Initialize persistent SQLite store[cite: 1]
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "tickets.db"
	}
	db := database.InitDB(dbPath)
	defer db.Close()
	log.Printf("SQLite initialized at %s", dbPath)

	mux := http.NewServeMux()

	// Public API Endpoints[cite: 1]
	mux.HandleFunc("GET /health", handlers.HealthHandler)
	mux.HandleFunc("POST /auth/register", handlers.RegisterHandler)
	mux.HandleFunc("POST /auth/login", handlers.LoginHandler)

	// Protected API Endpoints[cite: 1]
	mux.HandleFunc("POST /tickets", middleware.Authenticate(handlers.CreateTicketHandler))
	mux.HandleFunc("GET /tickets", middleware.Authenticate(handlers.ListTicketsHandler))
	mux.HandleFunc("GET /tickets/{id}", middleware.Authenticate(handlers.GetTicketByIDHandler))
	mux.HandleFunc("PATCH /tickets/{id}/status", middleware.Authenticate(handlers.UpdateTicketStatusHandler))

	// Embedded Static Frontend
	staticSub, err := fs.Sub(staticFiles, "static")
	if err == nil {
		mux.Handle("GET /", http.FileServer(http.FS(staticSub)))
	}

	// Port configuration: Defaults to 8080[cite: 1]
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running on port %s...", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server shutdown: %v", err)
	}
}