package database

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite" // Pure-Go SQLite driver without CGO requirement
)

var DB *sql.DB

// InitDB initializes the SQLite file database and auto-provisions tables.
func InitDB(dataSourceName string) *sql.DB {
	var err error
	DB, err = sql.Open("sqlite", dataSourceName)
	if err != nil {
		log.Fatalf("Database connection error: %v", err)
	}

	_, err = DB.Exec(`PRAGMA journal_mode = WAL; PRAGMA busy_timeout = 5000;`)
	if err != nil {
		log.Printf("PRAGMA configuration notice: %v", err)
	}

	createTables()
	return DB
}

func createTables() {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS tickets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		description TEXT NOT NULL,
		status TEXT NOT NULL CHECK(status IN ('open', 'in_progress', 'closed')),
		user_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	`
	if _, err := DB.Exec(schema); err != nil {
		log.Fatalf("Failed to initialize database tables: %v", err)
	}
}