# Ticket System Service

A REST backend service and embedded single-page UI built in Go and SQLite for managing support tickets with JWT authentication, ownership isolation, and strict state machine rules.

## 🔗 Live URLs
- **Deployment URL**: `[https://<your-service-name>.onrender.com](https://ticket-system-issue-app.onrender.com)`
- **Health Endpoint**: `https://<your-service-name>.onrender.com/health`

## 🛠️ Tech Stack
- **Language**: Go 1.22+
- **Database**: SQLite (`modernc.org/sqlite` pure-Go driver)
- **Auth**: JWT (`github.com/golang-jwt/jwt/v5`) and Bcrypt password hashing
- **Container**: Multi-stage Dockerfile

## 📋 API Endpoints
- `GET /health` - System health check (Public)
- `POST /auth/register` - User registration (Public)
- `POST /auth/login` - User login returning JWT (Public)
- `POST /tickets` - Create a ticket (Protected)
- `GET /tickets` - List own tickets (Protected)
- `GET /tickets/{id}` - Fetch own ticket by ID (Protected)
- `PATCH /tickets/{id}/status` - Advance ticket status (Protected)

## 🔄 Status Rules
- Valid flow: `open` -> `in_progress` -> `closed`
- Tickets marked `closed` cannot be edited or reopened.
- Users can only view and update their own tickets.

## 📐 Architecture & Lifecycle Design

### Ticket Status State Machine
```text
  [ Create Ticket ]
         │
         ▼
     ┌────────┐
     │  open  │ ──(PATCH /tickets/{id}/status)──► ┌─────────────┐
     └────────┘                                    │ in_progress │
                                                   └─────────────┘
                                                          │
                                            (PATCH /tickets/{id}/status)
                                                          │
                                                          ▼
                                                   ┌─────────────┐
                                                   │   closed    │ ──► [ Terminal State: Immutable ]
                                                   └─────────────┘

## 💻 Local Commands
```bash
# Direct Run
go run main.go

# Docker Run
docker build -t ticket-system .
docker run -p 8080:8080 ticket-system

# Request Flow & Security Pipeline
Incoming HTTP Request
       │
       ▼
[ Router (net/http ServeMux) ]
       │
       ▼
[ Middleware: Authenticate ]
       ├── Missing / Invalid Header ──► 401 Unauthorized
       └── Valid Bearer JWT
                 │
                 ▼
         Inject UserID into Context
                 │
                 ▼
          [ HTTP Handler ]
                 │
                 ├── Validate Input JSON
                 ├── Enforce Ownership Check (ticket.user_id == ctx.user_id) ──► 403 Forbidden
                 └── Database Transaction (SQLite)
