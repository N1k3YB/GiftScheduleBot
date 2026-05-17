package db

import (
	"database/sql"
	"time"
)

type User struct {
	ID          int64
	TelegramID  int64
	Username    string
	FirstName   string
	LastName    string
	IsBanned    bool
	Notify3Days bool
	Notify1Day  bool
	Notify1Hour bool
	CreatedAt   time.Time
}

func UpsertUser(telegramID int64, username, firstName, lastName string) (*User, error) {
	_, err := DB.Exec(`
		INSERT INTO users (telegram_id, username, first_name, last_name)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(telegram_id) DO UPDATE SET
			username   = excluded.username,
			first_name = excluded.first_name,
			last_name  = excluded.last_name
	`, telegramID, username, firstName, lastName)
	if err != nil {
		return nil, err
	}
	return GetUserByTelegramID(telegramID)
}

func GetUserByTelegramID(telegramID int64) (*User, error) {
	row := DB.QueryRow(`
		SELECT id, telegram_id, username, first_name, last_name,
		       is_banned, notify_3days, notify_1day, notify_1hour, created_at
		FROM users WHERE telegram_id = ?`, telegramID)
	return scanUser(row)
}

func GetUserByID(id int64) (*User, error) {
	row := DB.QueryRow(`
		SELECT id, telegram_id, username, first_name, last_name,
		       is_banned, notify_3days, notify_1day, notify_1hour, created_at
		FROM users WHERE id = ?`, id)
	return scanUser(row)
}

func scanUser(row *sql.Row) (*User, error) {
	u := &User{}
	err := row.Scan(
		&u.ID, &u.TelegramID, &u.Username, &u.FirstName, &u.LastName,
		&u.IsBanned, &u.Notify3Days, &u.Notify1Day, &u.Notify1Hour, &u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func SetNotify(userID int64, field string, val bool) error {
	allowed := map[string]bool{"notify_3days": true, "notify_1day": true, "notify_1hour": true}
	if !allowed[field] {
		return nil
	}
	_, err := DB.Exec("UPDATE users SET "+field+" = ? WHERE id = ?", val, userID)
	return err
}

func SetBanned(userID int64, banned bool) error {
	_, err := DB.Exec("UPDATE users SET is_banned = ? WHERE id = ?", banned, userID)
	return err
}

type UserStats struct {
	TotalUsers  int
	NewToday    int
	NewThisWeek int
}

func GetUserStats() (UserStats, error) {
	var s UserStats
	err := DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&s.TotalUsers)
	if err != nil {
		return s, err
	}
	err = DB.QueryRow(`SELECT COUNT(*) FROM users WHERE date(created_at) = date('now')`).Scan(&s.NewToday)
	if err != nil {
		return s, err
	}
	err = DB.QueryRow(`SELECT COUNT(*) FROM users WHERE created_at >= datetime('now', '-7 days')`).Scan(&s.NewThisWeek)
	return s, err
}

func ListUsers(limit, offset int) ([]*User, error) {
	rows, err := DB.Query(`
		SELECT id, telegram_id, username, first_name, last_name,
		       is_banned, notify_3days, notify_1day, notify_1hour, created_at
		FROM users ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(
			&u.ID, &u.TelegramID, &u.Username, &u.FirstName, &u.LastName,
			&u.IsBanned, &u.Notify3Days, &u.Notify1Day, &u.Notify1Hour, &u.CreatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func CountUsers() (int, error) {
	var n int
	err := DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func SearchUser(query string) ([]*User, error) {
	pattern := "%" + query + "%"
	rows, err := DB.Query(`
		SELECT id, telegram_id, username, first_name, last_name,
		       is_banned, notify_3days, notify_1day, notify_1hour, created_at
		FROM users
		WHERE username LIKE ? OR first_name LIKE ? OR CAST(telegram_id AS TEXT) = ?
		LIMIT 20`, pattern, pattern, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(
			&u.ID, &u.TelegramID, &u.Username, &u.FirstName, &u.LastName,
			&u.IsBanned, &u.Notify3Days, &u.Notify1Day, &u.Notify1Hour, &u.CreatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func GetAllUsersForBroadcast() ([]*User, error) {
	rows, err := DB.Query(`
		SELECT id, telegram_id, username, first_name, last_name,
		       is_banned, notify_3days, notify_1day, notify_1hour, created_at
		FROM users WHERE is_banned = 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(
			&u.ID, &u.TelegramID, &u.Username, &u.FirstName, &u.LastName,
			&u.IsBanned, &u.Notify3Days, &u.Notify1Day, &u.Notify1Hour, &u.CreatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
