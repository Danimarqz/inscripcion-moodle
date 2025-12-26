package moodle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/inscripcion-moodle/go-backend/internal/config"
)

var (
	ErrUserAlreadyExists = errors.New("moodle username or email already exists")
)

func Call(ctx context.Context, cfg *config.Config, client *http.Client, function string, extra url.Values) ([]byte, error) {
	if cfg == nil || strings.TrimSpace(cfg.MoodleURL) == "" || strings.TrimSpace(cfg.MoodleToken) == "" {
		return nil, ErrNotConfigured
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	endpoint := fmt.Sprintf("%s/webservice/rest/server.php", strings.TrimRight(cfg.MoodleURL, "/"))
	payload := url.Values{}
	payload.Set("wstoken", cfg.MoodleToken)
	payload.Set("wsfunction", function)
	payload.Set("moodlewsrestformat", "json")
	for key, vals := range extra {
		for _, val := range vals {
			payload.Add(key, val)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(payload.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("moodle error: %s", strings.TrimSpace(string(body)))
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(body, &decoded); err == nil {
		if exception, ok := decoded["exception"]; ok && exception != nil {
			msg := strings.ToLower(fmt.Sprint(decoded["message"]))
			if strings.Contains(msg, "username already exists") || strings.Contains(msg, "email address already exists") {
				return nil, ErrUserAlreadyExists
			}
			return nil, fmt.Errorf("moodle exception: %v", decoded["message"])
		}
	}

	return body, nil
}

func ToInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	case json.Number:
		return strconv.Atoi(v.String())
	default:
		return 0, fmt.Errorf("unexpected id type %T", value)
	}
}
