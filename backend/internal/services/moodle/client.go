package moodle

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/inscripcion-moodle/go-backend/internal/config"
)

var (
	ErrNotConfigured = errors.New("moodle not configured")
	ErrUserNotFound  = errors.New("moodle user not found")
)

type MoodleUser struct {
	ID   int
	Auth string
}

type Client struct {
	cfg        *config.Config
	httpClient *http.Client
}

func New(cfg *config.Config) *Client {
	if cfg == nil {
		return nil
	}
	if strings.TrimSpace(cfg.MoodleURL) == "" || strings.TrimSpace(cfg.MoodleToken) == "" {
		return nil
	}
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) call(ctx context.Context, function string, extra url.Values) ([]byte, error) {
	return Call(ctx, c.cfg, c.httpClient, function, extra)
}

func (c *Client) FindUsersByField(ctx context.Context, field, value string) ([]MoodleUser, error) {
	if c == nil {
		return nil, ErrNotConfigured
	}
	payload := url.Values{}
	payload.Set("field", field)
	payload.Set("values[0]", value)
	body, err := c.call(ctx, "core_user_get_users_by_field", payload)
	if err != nil {
		return nil, err
	}

	var raw []map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	users := make([]MoodleUser, 0, len(raw))
	for _, item := range raw {
		idVal, ok := item["id"]
		if !ok {
			continue
		}
		id, err := ToInt(idVal)
		if err != nil {
			continue
		}

		auth := ""
		if rawAuth, ok := item["auth"]; ok {
			if str, ok := rawAuth.(string); ok {
				auth = str
			}
		}

		users = append(users, MoodleUser{ID: id, Auth: auth})
	}

	if len(users) == 0 {
		return nil, ErrUserNotFound
	}
	return users, nil
}
