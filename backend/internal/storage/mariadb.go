package storage

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/inscripcion-moodle/go-backend/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func NewMariaDB(cfg *config.Config) (*gorm.DB, error) {
	dsn, err := dsnFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		return nil, fmt.Errorf("connect to mariadb: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db handle: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.DBConnMaxLifetime)

	return db, nil
}

func dsnFromConfig(cfg *config.Config) (string, error) {
	if cfg.DatabaseURL != "" {
		return buildMariaDSN(cfg.DatabaseURL)
	}

	if cfg.DBName == "" {
		return "", fmt.Errorf("DB_NAME is required when DATABASE_URL is not set")
	}

	host := cfg.DBHost
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.DBPort
	if port == "" {
		port = "3306"
	}

	user := cfg.DBUser
	if user == "" {
		user = "root"
	}

	auth := user
	if cfg.DBPassword != "" {
		auth = fmt.Sprintf("%s:%s", user, cfg.DBPassword)
	}

	hostPort := net.JoinHostPort(host, port)
	dsn := fmt.Sprintf("%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", auth, hostPort, cfg.DBName)
	return dsn, nil
}

func buildMariaDSN(raw string) (string, error) {
	if !strings.Contains(raw, "://") {
		return raw, nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse database url: %w", err)
	}

	baseScheme := strings.Split(parsed.Scheme, "+")[0]
	if baseScheme != "mysql" && baseScheme != "mariadb" {
		return "", fmt.Errorf("unsupported database scheme %q", parsed.Scheme)
	}

	host := parsed.Hostname()
	if host == "" {
		host = "127.0.0.1"
	}
	port := parsed.Port()
	if port == "" {
		port = "3306"
	}

	hostPort := net.JoinHostPort(host, port)

	dbName := strings.TrimPrefix(parsed.Path, "/")
	if dbName == "" {
		return "", fmt.Errorf("database name missing in url")
	}

	var authPart string
	if parsed.User != nil {
		authPart = parsed.User.String()
	}
	if authPart != "" {
		authPart += "@"
	}

	values := parsed.Query()
	if values.Get("charset") == "" {
		values.Set("charset", "utf8mb4")
	}
	if values.Get("parseTime") == "" {
		values.Set("parseTime", "True")
	}
	if values.Get("loc") == "" {
		values.Set("loc", "Local")
	}

	encoded := values.Encode()
	if encoded == "" {
		encoded = "charset=utf8mb4&parseTime=True&loc=Local"
	}

	dsn := fmt.Sprintf("%s@tcp(%s)/%s?%s", authPart, hostPort, dbName, encoded)
	return dsn, nil
}
