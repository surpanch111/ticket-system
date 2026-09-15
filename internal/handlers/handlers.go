package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ticket-system/internal/auth"
	"ticket-system/internal/database"
	"ticket-system/internal/middleware"
	"ticket-system/internal/models"
)

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// HealthHandler returns ok status for deployment monitoring.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// RegisterHandler registers a new unique user.
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	identifier := strings.TrimSpace(req.Email)
	if identifier == "" {
		identifier = strings.TrimSpace(req.Username)
	}

	if identifier == "" || strings.TrimSpace(req.Password) == "" {
		respondError(w, http.StatusBadRequest, "email/username and password are required")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to encrypt password")
		return
	}

	query := `INSERT INTO users (email, password_hash) VALUES (?, ?)`
	res, err := database.DB.Exec(query, identifier, hash)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			respondError(w, http.StatusConflict, "user already exists")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	userID, _ := res.LastInsertId()
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "user registered successfully",
		"user_id": userID,
	})
}

// LoginHandler authenticates and returns a JWT bearer token.
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	identifier := strings.TrimSpace(req.Email)
	if identifier == "" {
		identifier = strings.TrimSpace(req.Username)
	}

	var user models.User
	query := `SELECT id, email, password_hash FROM users WHERE email = ?`
	err := database.DB.QueryRow(query, identifier).Scan(&user.ID, &user.Email, &user.PasswordHash)
	if err != nil || !auth.CheckPasswordHash(req.Password, user.PasswordHash) {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate access token")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"token":      token,
		"token_type": "Bearer",
	})
}

// CreateTicketHandler saves a new ticket assigned to the authenticated user.
func CreateTicketHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(int64)

	var req models.CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Description) == "" {
		respondError(w, http.StatusBadRequest, "title and description are required")
		return
	}

	initialStatus := "open"
	now := time.Now().UTC()

	query := `INSERT INTO tickets (title, description, status, user_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`
	res, err := database.DB.Exec(query, req.Title, req.Description, initialStatus, userID, now, now)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create ticket")
		return
	}

	ticketID, _ := res.LastInsertId()
	ticket := models.Ticket{
		ID:          ticketID,
		Title:       req.Title,
		Description: req.Description,
		Status:      initialStatus,
		UserID:      userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	respondJSON(w, http.StatusCreated, ticket)
}

// ListTicketsHandler lists only the tickets owned by the authenticated caller.
func ListTicketsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(int64)

	query := `SELECT id, title, description, status, user_id, created_at, updated_at FROM tickets WHERE user_id = ? ORDER BY id DESC`
	rows, err := database.DB.Query(query, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch tickets")
		return
	}
	defer rows.Close()

	tickets := make([]models.Ticket, 0)
	for rows.Next() {
		var t models.Ticket
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.UserID, &t.CreatedAt, &t.UpdatedAt); err != nil {
			respondError(w, http.StatusInternalServerError, "failed parsing ticket")
			return
		}
		tickets = append(tickets, t)
	}

	respondJSON(w, http.StatusOK, tickets)
}

// GetTicketByIDHandler retrieves a single ticket while strictly checking ownership.
func GetTicketByIDHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(int64)
	idStr := r.PathValue("id")

	var ticket models.Ticket
	query := `SELECT id, title, description, status, user_id, created_at, updated_at FROM tickets WHERE id = ?`
	err := database.DB.QueryRow(query, idStr).Scan(
		&ticket.ID, &ticket.Title, &ticket.Description, &ticket.Status,
		&ticket.UserID, &ticket.CreatedAt, &ticket.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "ticket not found")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	// Ownership check: user can only see tickets they created
	if ticket.UserID != userID {
		respondError(w, http.StatusForbidden, "forbidden: you cannot access tickets created by other users")
		return
	}

	respondJSON(w, http.StatusOK, ticket)
}

// UpdateTicketStatusHandler handles ticket status updates with lifecycle validation.
func UpdateTicketStatusHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(int64)
	idStr := r.PathValue("id")

	var req models.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	newStatus := strings.ToLower(strings.TrimSpace(req.Status))

	var ticket models.Ticket
	query := `SELECT id, title, description, status, user_id, created_at, updated_at FROM tickets WHERE id = ?`
	err := database.DB.QueryRow(query, idStr).Scan(
		&ticket.ID, &ticket.Title, &ticket.Description, &ticket.Status,
		&ticket.UserID, &ticket.CreatedAt, &ticket.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "ticket not found")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	// Ownership check: user can only update their own tickets
	if ticket.UserID != userID {
		respondError(w, http.StatusForbidden, "forbidden: you cannot update tickets created by other users")
		return
	}

	// Enforce: open -> in_progress -> closed[cite: 1]
	// Enforce: closed cannot move back to open or in_progress[cite: 1]
	isValid, errMsg := validateTransition(ticket.Status, newStatus)
	if !isValid {
		respondError(w, http.StatusBadRequest, errMsg)
		return
	}

	now := time.Now().UTC()
	updateQuery := `UPDATE tickets SET status = ?, updated_at = ? WHERE id = ?`
	_, err = database.DB.Exec(updateQuery, newStatus, now, ticket.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update ticket status")
		return
	}

	ticket.Status = newStatus
	ticket.UpdatedAt = now
	respondJSON(w, http.StatusOK, ticket)
}

func validateTransition(current, next string) (bool, string) {
	if next != "open" && next != "in_progress" && next != "closed" {
		return false, "invalid status; must be 'open', 'in_progress', or 'closed'"
	}
	if current == "closed" {
		return false, "a closed ticket cannot be reopened or updated" //[cite: 1]
	}
	if current == next {
		return true, ""
	}
	if current == "open" && next == "in_progress" {
		return true, ""
	}
	if current == "in_progress" && next == "closed" {
		return true, ""
	}
	if current == "in_progress" && next == "open" {
		return false, "cannot move status backwards from 'in_progress' to 'open'"
	}
	if current == "open" && next == "closed" {
		return false, "ticket must proceed through 'in_progress' before being 'closed'"
	}
	return false, fmt.Sprintf("invalid transition from %s to %s", current, next)
}