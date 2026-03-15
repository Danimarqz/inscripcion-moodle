package moodle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/inscripcion-moodle/go-backend/internal/config"
)

const defaultMoodleTablePrefix = "mdl_"

func newMoodleDB(cfg *config.Config) (*sql.DB, error) {
	if cfg == nil {
		return nil, ErrNotConfigured
	}

	if strings.TrimSpace(cfg.MoodleDBURL) == "" && strings.TrimSpace(cfg.MoodleDBName) == "" {
		return nil, nil
	}

	dsn, err := moodleDSNFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	log.Printf("moodle: connecting to DB with user='%s' host='%s' name='%s'", cfg.MoodleDBUser, cfg.MoodleDBHost, cfg.MoodleDBName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(cfg.DBConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping moodle db: %w", err)
	}

	return db, nil
}

func moodleDSNFromConfig(cfg *config.Config) (string, error) {
	if cfg.MoodleDBURL != "" {
		return cfg.MoodleDBURL, nil
	}

	if cfg.MoodleDBName == "" {
		return "", fmt.Errorf("MOODLE_DB_NAME is required when MOODLE_DB_URL is empty")
	}
	if cfg.MoodleDBUser == "" {
		return "", fmt.Errorf("MOODLE_DB_USER is required when MOODLE_DB_URL is empty")
	}

	host := cfg.MoodleDBHost
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.MoodleDBPort
	if port == "" {
		port = "3306"
	}

	auth := cfg.MoodleDBUser
	if cfg.MoodleDBPassword != "" {
		auth = fmt.Sprintf("%s:%s", cfg.MoodleDBUser, cfg.MoodleDBPassword)
	}

	hostPort := net.JoinHostPort(host, port)
	return fmt.Sprintf("%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", auth, hostPort, cfg.MoodleDBName), nil
}

func normalizeTablePrefix(prefix string) string {
	trimmed := strings.TrimSpace(prefix)
	if trimmed == "" {
		return defaultMoodleTablePrefix
	}
	if !strings.HasSuffix(trimmed, "_") {
		trimmed += "_"
	}
	return trimmed
}

func (c *Client) findUserByEmailDB(ctx context.Context, email string) (MoodleUser, error) {
	if c.db == nil {
		return MoodleUser{}, ErrMoodleDBNotConfigured
	}

	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return MoodleUser{}, fmt.Errorf("invalid email")
	}

	query := fmt.Sprintf("SELECT id, auth FROM %suser WHERE LOWER(email) = ? LIMIT 1", c.tablePrefix)
	var user MoodleUser
	if err := c.db.QueryRowContext(ctx, query, normalized).Scan(&user.ID, &user.Auth); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MoodleUser{}, ErrUserNotFound
		}
		return MoodleUser{}, fmt.Errorf("query moodle user: %w", err)
	}

	return user, nil
}

type MoodleUserDNI struct {
	ID        int
	Email     string
	Firstname string
	Lastname  string
	DNI       string
}

func (c *Client) GetAllUsersDNI(ctx context.Context, courseID int) ([]MoodleUserDNI, error) {
	if c.db == nil {
		return nil, ErrMoodleDBNotConfigured
	}

	query := fmt.Sprintf(`
		SELECT u.id, u.email, u.firstname, u.lastname, d.data
		FROM %suser_info_data d
		INNER JOIN %suser u ON u.id = d.userid
		INNER JOIN %suser_enrolments ue ON u.id = ue.userid
		INNER JOIN %senrol e ON ue.enrolid = e.id AND e.courseid = ?
		WHERE d.fieldid = 1
	`, c.tablePrefix, c.tablePrefix, c.tablePrefix, c.tablePrefix)

	rows, err := c.db.QueryContext(ctx, query, courseID)
	if err != nil {
		return nil, fmt.Errorf("query moodle user dni: %w", err)
	}
	defer rows.Close()

	var users []MoodleUserDNI
	for rows.Next() {
		var u MoodleUserDNI
		if err := rows.Scan(&u.ID, &u.Email, &u.Firstname, &u.Lastname, &u.DNI); err != nil {
			continue
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
