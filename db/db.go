package db

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Open(path string) {
	var err error
	DB, err = sql.Open("sqlite", path)
	if err != nil {
		log.Fatalf("failed to open sqlite: %v", err)
	}
	DB.SetMaxOpenConns(1)
	if err = DB.Ping(); err != nil {
		log.Fatalf("failed to ping sqlite: %v", err)
	}
	migrate()
	log.Println("database opened:", path)
}

func migrate() {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			telegram_id INTEGER UNIQUE NOT NULL,
			username    TEXT,
			first_name  TEXT,
			last_name   TEXT,
			is_banned   BOOLEAN NOT NULL DEFAULT 0,
			notify_3days BOOLEAN NOT NULL DEFAULT 1,
			notify_1day  BOOLEAN NOT NULL DEFAULT 1,
			notify_1hour BOOLEAN NOT NULL DEFAULT 1,
			created_at  DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS posts (
			id                   INTEGER PRIMARY KEY AUTOINCREMENT,
			telegram_msg_id      INTEGER,
			channel_username     TEXT,
			channel_id           INTEGER,
			title                TEXT,
			prizes               TEXT,
			end_date             DATETIME,
			results_url          TEXT,
			results_in_same_post BOOLEAN NOT NULL DEFAULT 0,
			is_completed         BOOLEAN NOT NULL DEFAULT 0,
			source_url           TEXT,
			content_parsed       BOOLEAN NOT NULL DEFAULT 0,
			raw_text             TEXT,
			created_at           DATETIME NOT NULL DEFAULT (datetime('now')),
			UNIQUE(channel_id, telegram_msg_id)
		)`,
		`CREATE TABLE IF NOT EXISTS user_posts (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			post_id    INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			UNIQUE(user_id, post_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_posts_user ON user_posts(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_posts_post ON user_posts(post_id)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_end_date ON posts(end_date)`,
		`PRAGMA foreign_keys = ON`,
	}
	for _, s := range stmts {
		if _, err := DB.Exec(s); err != nil {
			log.Fatalf("migration failed: %v\nSQL: %s", err, s)
		}
	}
}
