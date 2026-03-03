package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	_, err = db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set journal_mode: %w", err)
	}

	_, err = db.Exec("PRAGMA foreign_keys=ON")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign_keys: %w", err)
	}

	_, err = db.Exec("PRAGMA busy_timeout=5000")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set busy_timeout: %w", err)
	}

	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) GetSession(id string) (*SessionRecord, error) {
	var record SessionRecord
	var createdAtStr, updatedAtStr string

	err := s.db.QueryRow(
		"SELECT id, messages, created_at, updated_at FROM sessions WHERE id = ?",
		id,
	).Scan(&record.ID, &record.Messages, &createdAtStr, &updatedAtStr)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	record.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}

	record.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated_at: %w", err)
	}

	return &record, nil
}

func (s *SQLiteStore) SaveSession(record *SessionRecord) error {
	createdAtStr := record.CreatedAt.Format(time.RFC3339)
	updatedAtStr := record.UpdatedAt.Format(time.RFC3339)

	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO sessions (id, messages, created_at, updated_at) VALUES (?, ?, ?, ?)",
		record.ID, record.Messages, createdAtStr, updatedAtStr,
	)
	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

func (s *SQLiteStore) DeleteSession(id string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

func (s *SQLiteStore) ListSessions() ([]*SessionRecord, error) {
	rows, err := s.db.Query("SELECT id, messages, created_at, updated_at FROM sessions ORDER BY updated_at DESC")
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*SessionRecord
	for rows.Next() {
		var record SessionRecord
		var createdAtStr, updatedAtStr string

		err := rows.Scan(&record.ID, &record.Messages, &createdAtStr, &updatedAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		record.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at: %w", err)
		}

		record.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse updated_at: %w", err)
		}

		sessions = append(sessions, &record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	return sessions, nil
}
