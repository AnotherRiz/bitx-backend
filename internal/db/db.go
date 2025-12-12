package db

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(path string) {
	var err error
	DB, err = sql.Open("sqlite", path)
	if err != nil {
		log.Fatal(err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS transfers (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  code_hash TEXT NOT NULL,
	  payload TEXT NOT NULL,
	  expires_at INTEGER NOT NULL,
	  used INTEGER DEFAULT 0,
	  created_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_expires ON transfers(expires_at);
	`
	if _, err := DB.Exec(schema); err != nil {
		log.Fatal(err)
	}
}
