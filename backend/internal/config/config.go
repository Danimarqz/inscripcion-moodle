package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               int
	DatabaseURL        string
	RedisURL           string
	AllowedOrigins     []string
	RateLimitRequests  int
	RateLimitWindow    time.Duration
	DBMaxOpenConns     int
	DBMaxIdleConns     int
	DBConnMaxLifetime  time.Duration
	PublicCacheTTL     time.Duration
	DBUser             string
	DBPassword         string
	DBHost             string
	DBPort             string
	DBName             string
	SecretKey          string
	TokenAlgorithm     string
	TokenTTLMinutes    int
	ServerReadTimeout  time.Duration
	ServerWriteTimeout time.Duration
	ServerIdleTimeout  time.Duration
	SMTPUser           string
	SMTPPass           string
	SMTPServer         string
	SMTPPort           int
	AdminEmail         string
	MoodleURL          string
	MoodleToken        string
	MoodleDBURL        string
	MoodleDBUser       string
	MoodleDBPassword   string
	MoodleDBHost       string
	MoodleDBPort       string
	MoodleDBName       string
	MoodleDBPrefix     string
	GSheetAPI          string
	MoodleExamCourseID int
}

func Load() (*Config, error) {
	if err := godotenv.Load(".env"); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load env file: %w", err)
	}

	port := parseEnvInt("PORT", 8080)
	databaseURL := os.Getenv("DATABASE_URL")
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	origins, err := parseOrigins(os.Getenv("ALLOWED_ORIGINS"))
	if err != nil {
		return nil, err
	}
	rateLimitRequests := parseEnvInt("RATE_LIMIT_REQUESTS", 60)
	rateLimitWindowSeconds := parseEnvInt("RATE_LIMIT_WINDOW_SECONDS", 60)
	tokenTTL := parseEnvInt("ACCESS_TOKEN_EXPIRE_MINUTES", 60)

	return &Config{
		Port:               port,
		DatabaseURL:        databaseURL,
		RedisURL:           redisURL,
		AllowedOrigins:     origins,
		RateLimitRequests:  rateLimitRequests,
		RateLimitWindow:    time.Duration(rateLimitWindowSeconds) * time.Second,
		DBMaxOpenConns:     parseEnvInt("DB_MAX_OPEN_CONNS", 15),
		DBMaxIdleConns:     parseEnvInt("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLifetime:  time.Duration(parseEnvInt("DB_CONN_MAX_LIFETIME_SECONDS", 1800)) * time.Second,
		PublicCacheTTL:     time.Duration(parseEnvInt("PUBLIC_CACHE_TTL_SECONDS", 60)) * time.Second,
		DBUser:             os.Getenv("DB_USER"),
		DBPassword:         os.Getenv("DB_PASSWORD"),
		DBHost:             os.Getenv("DB_HOST"),
		DBPort:             os.Getenv("DB_PORT"),
		DBName:             os.Getenv("DB_NAME"),
		SecretKey:          os.Getenv("SECRET_KEY"),
		TokenAlgorithm:     parseEnvString("ALGORITHM", "HS256"),
		TokenTTLMinutes:    tokenTTL,
		SMTPUser:           os.Getenv("SMTP_USER"),
		SMTPPass:           os.Getenv("SMTP_PASS"),
		SMTPServer:         parseEnvString("SMTP_SERVER", "smtp.gmail.com"),
		SMTPPort:           parseEnvInt("SMTP_PORT", 587),
		AdminEmail:         parseEnvString("ADMIN_EMAIL", os.Getenv("SMTP_USER")),
		MoodleURL:          os.Getenv("MOODLE_URL"),
		MoodleToken:        os.Getenv("MOODLE_TOKEN"),
		MoodleDBURL:        os.Getenv("MOODLE_DB_URL"),
		MoodleDBUser:       os.Getenv("MOODLE_DB_USER"),
		MoodleDBPassword:   os.Getenv("MOODLE_DB_PASSWORD"),
		MoodleDBHost:       parseEnvString("MOODLE_DB_HOST", "localhost"),
		MoodleDBPort:       parseEnvString("MOODLE_DB_PORT", "3306"),
		MoodleDBName:       os.Getenv("MOODLE_DB_NAME"),
		MoodleDBPrefix:     parseEnvString("MOODLE_DB_PREFIX", "mdl_"),
		GSheetAPI:          os.Getenv("API_GSHEET"),
		MoodleExamCourseID: parseEnvInt("MOODLE_EXAM_COURSE_ID", 49),
	}, nil
}

func parseEnvString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func parseEnvInt(key string, fallback int) int {
	if raw := os.Getenv(key); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil {
			return value
		}
	}
	return fallback
}

func parseOrigins(env string) ([]string, error) {
	if env == "" {
		return nil, fmt.Errorf("ALLOWED_ORIGINS environment variable is required (comma-separated list of origins)")
	}
	parts := strings.Split(env, ",")
	result := make([]string, 0, len(parts))
	for _, raw := range parts {
		item := strings.TrimSpace(raw)
		if item != "" {
			result = append(result, item)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("ALLOWED_ORIGINS environment variable is required (comma-separated list of origins)")
	}
	return result, nil
}
