package models

import "time"

// User holds registered account credentials.
type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Omitted from API responses
	CreatedAt    time.Time `json:"created_at"`
}

// Ticket holds support ticket data belonging to a single user.
type Ticket struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // Allowed values: "open", "in_progress", "closed"
	UserID      int64     `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RegisterRequest accepts either email or username for sign-up compatibility.
type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginRequest accepts user credentials.
type LoginRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// CreateTicketRequest accepts data to create a ticket.
type CreateTicketRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// UpdateStatusRequest accepts the target status.
type UpdateStatusRequest struct {
	Status string `json:"status"`
}