package moodle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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

type EnrolledUser struct {
	ID    int
	Email string
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

func (c *Client) GetEnrolledUsers(ctx context.Context, courseID int, userIDs []int) ([]EnrolledUser, error) {
	if c == nil {
		return nil, ErrNotConfigured
	}
	payload := url.Values{}
	payload.Set("courseid", strconv.Itoa(courseID))
	for idx, id := range userIDs {
		payload.Set(fmt.Sprintf("options[userids][%d]", idx), strconv.Itoa(id))
	}

	body, err := c.call(ctx, "core_enrol_get_enrolled_users", payload)
	if err != nil {
		return nil, err
	}

	var raw []map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	users := make([]EnrolledUser, 0, len(raw))
	for _, item := range raw {
		idVal, ok := item["id"]
		if !ok {
			continue
		}
		id, err := ToInt(idVal)
		if err != nil {
			continue
		}

		email := ""
		if rawEmail, ok := item["email"]; ok {
			if str, ok := rawEmail.(string); ok {
				email = str
			}
		}

		users = append(users, EnrolledUser{
			ID:    id,
			Email: email,
		})
	}

	return users, nil
}

func (c *Client) IsUserEnrolledInCourse(ctx context.Context, courseID, userID int) (bool, error) {
	if c == nil {
		return false, ErrNotConfigured
	}
	users, err := c.GetEnrolledUsers(ctx, courseID, []int{userID})
	if err != nil {
		return false, err
	}
	return len(users) > 0, nil
}
