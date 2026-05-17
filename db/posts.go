package db

import (
	"database/sql"
	"time"
)

type Post struct {
	ID                int64
	TelegramMsgID     int64
	ChannelUsername   string
	ChannelID         int64
	Title             string
	Prizes            string
	EndDate           *time.Time
	HasEndTime        bool
	ResultsURL        string
	ResultsInSamePost bool
	IsCompleted       bool
	ResultPollStarted bool
	DumpMsgID         int64
	SourceURL         string
	ContentParsed     bool
	RawText           string
	CreatedAt         time.Time
}

func CreateOrGetPost(p *Post) (*Post, bool, error) {
	var existing Post
	err := DB.QueryRow(`
		SELECT id FROM posts WHERE channel_id = ? AND telegram_msg_id = ?`,
		p.ChannelID, p.TelegramMsgID).Scan(&existing.ID)
	if err == nil {
		full, ferr := GetPostByID(existing.ID)
		return full, false, ferr
	}
	if err != sql.ErrNoRows {
		return nil, false, err
	}

	res, err := DB.Exec(`
		INSERT INTO posts (telegram_msg_id, channel_username, channel_id, title, prizes,
		                   end_date, has_end_time, results_url, results_in_same_post, source_url,
		                   content_parsed, raw_text, dump_msg_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.TelegramMsgID, p.ChannelUsername, p.ChannelID, p.Title, p.Prizes,
		p.EndDate, p.HasEndTime, p.ResultsURL, p.ResultsInSamePost, p.SourceURL,
		p.ContentParsed, p.RawText, p.DumpMsgID,
	)
	if err != nil {
		return nil, false, err
	}
	id, _ := res.LastInsertId()
	created, err := GetPostByID(id)
	return created, true, err
}

func IsBannedPost(channelID, msgID int64) bool {
	var count int
	DB.QueryRow(`SELECT COUNT(*) FROM banned_posts WHERE channel_id=? AND telegram_msg_id=?`,
		channelID, msgID).Scan(&count)
	return count > 0
}

func BanPost(postID int64) error {
	p, err := GetPostByID(postID)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`
		INSERT OR IGNORE INTO banned_posts (telegram_msg_id, channel_id, channel_username, full_text)
		VALUES (?, ?, ?, ?)`,
		p.TelegramMsgID, p.ChannelID, p.ChannelUsername, p.RawText)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`DELETE FROM posts WHERE id = ?`, postID)
	return err
}

func GetPostsForResultPoll() ([]*Post, error) {
	rows, err := DB.Query(`
		SELECT id, telegram_msg_id, channel_username, channel_id, title, prizes,
		       end_date, has_end_time, results_url, results_in_same_post, is_completed,
		       result_poll_started, source_url, content_parsed, raw_text, created_at
		FROM posts
		WHERE is_completed = 0
		  AND end_date IS NOT NULL
		  AND (
		    (has_end_time = 1 AND end_date <= datetime('now'))
		    OR
		    (has_end_time = 0 AND date(end_date) <= date('now'))
		  )`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPostsFull(rows)
}

func GetPostSubscribers(postID int64) ([]int64, error) {
	rows, err := DB.Query(`
		SELECT u.telegram_id FROM user_posts up
		JOIN users u ON u.id = up.user_id
		WHERE up.post_id = ? AND u.is_banned = 0`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}

func MarkResultPollStarted(postID int64) error {
	_, err := DB.Exec(`UPDATE posts SET result_poll_started = 1 WHERE id = ?`, postID)
	return err
}

func GetPostByID(id int64) (*Post, error) {
	row := DB.QueryRow(`
		SELECT id, telegram_msg_id, channel_username, channel_id, title, prizes,
		       end_date, has_end_time, results_url, results_in_same_post, is_completed,
		       result_poll_started, dump_msg_id, source_url, content_parsed, raw_text, created_at
		FROM posts WHERE id = ?`, id)
	return scanPost(row)
}

func SaveDumpMsgID(postID, dumpMsgID int64) error {
	_, err := DB.Exec(`UPDATE posts SET dump_msg_id = ? WHERE id = ?`, dumpMsgID, postID)
	return err
}

func scanPost(row *sql.Row) (*Post, error) {
	p := &Post{}
	err := row.Scan(
		&p.ID, &p.TelegramMsgID, &p.ChannelUsername, &p.ChannelID,
		&p.Title, &p.Prizes, &p.EndDate, &p.HasEndTime, &p.ResultsURL,
		&p.ResultsInSamePost, &p.IsCompleted,
		&p.ResultPollStarted, &p.DumpMsgID, &p.SourceURL, &p.ContentParsed, &p.RawText, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func AddUserPost(userID, postID int64) error {
	_, err := DB.Exec(`
		INSERT OR IGNORE INTO user_posts (user_id, post_id) VALUES (?, ?)`,
		userID, postID)
	return err
}

func RemoveUserPost(userID, userPostID int64) error {
	_, err := DB.Exec(`
		DELETE FROM user_posts WHERE id = ? AND user_id = ?`,
		userPostID, userID)
	return err
}

type UserPost struct {
	UserPostID int64
	Post       Post
}

func GetUserPosts(userID int64, limit, offset int) ([]UserPost, error) {
	rows, err := DB.Query(`
		SELECT up.id,
		       p.id, p.telegram_msg_id, p.channel_username, p.channel_id, p.title, p.prizes,
		       p.end_date, p.has_end_time, p.results_url, p.results_in_same_post, p.is_completed,
		       p.result_poll_started, p.dump_msg_id, p.source_url, p.content_parsed, p.raw_text, p.created_at
		FROM user_posts up
		JOIN posts p ON p.id = up.post_id
		WHERE up.user_id = ?
		ORDER BY up.id DESC
		LIMIT ? OFFSET ?`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUserPosts(rows)
}

func CountUserPosts(userID int64) (int, error) {
	var n int
	err := DB.QueryRow(`SELECT COUNT(*) FROM user_posts WHERE user_id = ?`, userID).Scan(&n)
	return n, err
}

func CountUserActivePosts(userID int64) (int, error) {
	var n int
	err := DB.QueryRow(`
		SELECT COUNT(*) FROM user_posts up
		JOIN posts p ON p.id = up.post_id
		WHERE up.user_id = ? AND p.is_completed = 0`, userID).Scan(&n)
	return n, err
}

func CountUserCompletedPosts(userID int64) (int, error) {
	var n int
	err := DB.QueryRow(`
		SELECT COUNT(*) FROM user_posts up
		JOIN posts p ON p.id = up.post_id
		WHERE up.user_id = ? AND p.is_completed = 1`, userID).Scan(&n)
	return n, err
}

func GetLastUserPosts(userID int64, limit int) ([]UserPost, error) {
	rows, err := DB.Query(`
		SELECT up.id,
		       p.id, p.telegram_msg_id, p.channel_username, p.channel_id, p.title, p.prizes,
		       p.end_date, p.has_end_time, p.results_url, p.results_in_same_post, p.is_completed,
		       p.result_poll_started, p.dump_msg_id, p.source_url, p.content_parsed, p.raw_text, p.created_at
		FROM user_posts up
		JOIN posts p ON p.id = up.post_id
		WHERE up.user_id = ?
		ORDER BY up.id DESC
		LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUserPosts(rows)
}

func scanUserPosts(rows *sql.Rows) ([]UserPost, error) {
	var list []UserPost
	for rows.Next() {
		var up UserPost
		p := &up.Post
		if err := rows.Scan(
			&up.UserPostID,
			&p.ID, &p.TelegramMsgID, &p.ChannelUsername, &p.ChannelID,
			&p.Title, &p.Prizes, &p.EndDate, &p.HasEndTime, &p.ResultsURL,
			&p.ResultsInSamePost, &p.IsCompleted,
			&p.ResultPollStarted, &p.DumpMsgID, &p.SourceURL, &p.ContentParsed, &p.RawText, &p.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, up)
	}
	return list, rows.Err()
}

func GetAllPosts(limit, offset int) ([]*Post, error) {
	rows, err := DB.Query(`
		SELECT id, telegram_msg_id, channel_username, channel_id, title, prizes,
		       end_date, results_url, results_in_same_post, is_completed,
		       source_url, content_parsed, raw_text, created_at
		FROM posts ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPosts(rows)
}

func CountAllPosts() (int, error) {
	var n int
	err := DB.QueryRow(`SELECT COUNT(*) FROM posts`).Scan(&n)
	return n, err
}

func CountActivePosts() (int, error) {
	var n int
	err := DB.QueryRow(`SELECT COUNT(*) FROM posts WHERE is_completed = 0`).Scan(&n)
	return n, err
}

func CountCompletedPosts() (int, error) {
	var n int
	err := DB.QueryRow(`SELECT COUNT(*) FROM posts WHERE is_completed = 1`).Scan(&n)
	return n, err
}

func scanPosts(rows *sql.Rows) ([]*Post, error) {
	var list []*Post
	for rows.Next() {
		p := &Post{}
		if err := rows.Scan(
			&p.ID, &p.TelegramMsgID, &p.ChannelUsername, &p.ChannelID,
			&p.Title, &p.Prizes, &p.EndDate, &p.ResultsURL,
			&p.ResultsInSamePost, &p.IsCompleted,
			&p.SourceURL, &p.ContentParsed, &p.RawText, &p.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func scanPostsFull(rows *sql.Rows) ([]*Post, error) {
	var list []*Post
	for rows.Next() {
		p := &Post{}
		if err := rows.Scan(
			&p.ID, &p.TelegramMsgID, &p.ChannelUsername, &p.ChannelID,
			&p.Title, &p.Prizes, &p.EndDate, &p.HasEndTime, &p.ResultsURL,
			&p.ResultsInSamePost, &p.IsCompleted,
			&p.ResultPollStarted, &p.DumpMsgID, &p.SourceURL, &p.ContentParsed, &p.RawText, &p.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func GetActivePendingNotification(notifyField string) ([]*Post, error) {
	allowed := map[string]bool{
		"notify_3days": true, "notify_1day": true, "notify_1hour": true,
	}
	if !allowed[notifyField] {
		return nil, nil
	}
	var (
		rows *sql.Rows
		err  error
	)
	switch notifyField {
	case "notify_3days":
		rows, err = DB.Query(`
			SELECT p.id, p.telegram_msg_id, p.channel_username, p.channel_id, p.title, p.prizes,
			       p.end_date, p.results_url, p.results_in_same_post, p.is_completed,
			       p.source_url, p.content_parsed, p.raw_text, p.created_at
			FROM posts p
			WHERE p.is_completed = 0
			  AND p.end_date IS NOT NULL
			  AND p.end_date BETWEEN datetime('now', '+71 hours') AND datetime('now', '+73 hours')`)
	case "notify_1day":
		rows, err = DB.Query(`
			SELECT p.id, p.telegram_msg_id, p.channel_username, p.channel_id, p.title, p.prizes,
			       p.end_date, p.results_url, p.results_in_same_post, p.is_completed,
			       p.source_url, p.content_parsed, p.raw_text, p.created_at
			FROM posts p
			WHERE p.is_completed = 0
			  AND p.end_date IS NOT NULL
			  AND p.end_date BETWEEN datetime('now', '+23 hours') AND datetime('now', '+25 hours')`)
	case "notify_1hour":
		rows, err = DB.Query(`
			SELECT p.id, p.telegram_msg_id, p.channel_username, p.channel_id, p.title, p.prizes,
			       p.end_date, p.results_url, p.results_in_same_post, p.is_completed,
			       p.source_url, p.content_parsed, p.raw_text, p.created_at
			FROM posts p
			WHERE p.is_completed = 0
			  AND p.end_date IS NOT NULL
			  AND p.end_date BETWEEN datetime('now') AND datetime('now', '+1 hours')`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPosts(rows)
}

func GetExpiredPosts() ([]*Post, error) {
	rows, err := DB.Query(`
		SELECT id, telegram_msg_id, channel_username, channel_id, title, prizes,
		       end_date, results_url, results_in_same_post, is_completed,
		       source_url, content_parsed, raw_text, created_at
		FROM posts
		WHERE is_completed = 0
		  AND end_date IS NOT NULL
		  AND end_date <= datetime('now')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPosts(rows)
}

func MarkCompleted(postID int64) error {
	_, err := DB.Exec(`UPDATE posts SET is_completed = 1 WHERE id = ?`, postID)
	return err
}

func GetUsersForPost(postID int64, notifyField string) ([]*User, error) {
	allowed := map[string]bool{
		"notify_3days": true, "notify_1day": true, "notify_1hour": true, "all": true,
	}
	if !allowed[notifyField] {
		return nil, nil
	}
	var query string
	if notifyField == "all" {
		query = `
			SELECT u.id, u.telegram_id, u.username, u.first_name, u.last_name,
			       u.is_banned, u.notify_3days, u.notify_1day, u.notify_1hour, u.created_at
			FROM users u
			JOIN user_posts up ON up.user_id = u.id
			WHERE up.post_id = ? AND u.is_banned = 0`
	} else {
		query = `
			SELECT u.id, u.telegram_id, u.username, u.first_name, u.last_name,
			       u.is_banned, u.notify_3days, u.notify_1day, u.notify_1hour, u.created_at
			FROM users u
			JOIN user_posts up ON up.user_id = u.id
			WHERE up.post_id = ? AND u.is_banned = 0 AND u.` + notifyField + ` = 1`
	}
	rows, err := DB.Query(query, postID)
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

func DeletePostFromDB(postID int64) error {
	_, err := DB.Exec(`DELETE FROM posts WHERE id = ?`, postID)
	return err
}

func GetPostStats() (total, active, completed, newToday, newWeek int, err error) {
	DB.QueryRow(`SELECT COUNT(*) FROM posts`).Scan(&total)
	DB.QueryRow(`SELECT COUNT(*) FROM posts WHERE is_completed = 0`).Scan(&active)
	DB.QueryRow(`SELECT COUNT(*) FROM posts WHERE is_completed = 1`).Scan(&completed)
	DB.QueryRow(`SELECT COUNT(*) FROM posts WHERE date(created_at) = date('now')`).Scan(&newToday)
	err = DB.QueryRow(`SELECT COUNT(*) FROM posts WHERE created_at >= datetime('now', '-7 days')`).Scan(&newWeek)
	return
}

func UserAlreadyHasPost(userID, postID int64) (bool, error) {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM user_posts WHERE user_id = ? AND post_id = ?`, userID, postID).Scan(&count)
	return count > 0, err
}
