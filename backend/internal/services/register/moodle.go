package register

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/inscripcion-moodle/go-backend/internal/services/moodle"
)

var (
	validCourses = map[string]int{
		"ingesa":      24,
		"sergas":      10,
		"gva":         15,
		"aragon":      3,
		"sescam":      13,
		"osakidetza":  23,
		"sacyl":       26,
		"osasunbidea": 36,
		"seris":       46,
		"cantabria":   35,
		"canarias":    25,
		"sms":         72,
		"sespa":       45,
		"imserso":     18,
		"ses":         12,
		"era":         14,
		"baleares":    83,
		"xunta":       89,
		"sas":         19,
		"sermas":      20,
	}
	extraCourses = []int{16, 11}
)

func (s *Service) createMoodleUser(ctx context.Context, data Data) (bool, error) {
	username := strings.ToLower(strings.TrimSpace(data.Email))
	courseKey := strings.TrimSpace(strings.ToLower(data.Course))
	payload := url.Values{
		"users[0][username]":  {username},
		"users[0][firstname]": {data.Name},
		"users[0][lastname]":  {data.Surname},
		"users[0][email]":     {data.Email},
		"users[0][auth]":      {"manual"},
		"users[0][password]":  {generatePassword(data.DNI)},
	}
	custom := []struct {
		key   string
		value string
	}{
		{"dni", data.DNI},
		{"conocer", data.Discover},
	}
	for i, field := range append(custom, courseCustomField(courseKey)...) {
		payload.Set(fmt.Sprintf("users[0][customfields][%d][type]", i), field.key)
		payload.Set(fmt.Sprintf("users[0][customfields][%d][value]", i), field.value)
	}

	var userID int
	body, err := s.callMoodle(ctx, "core_user_create_users", payload)
	if err != nil {
		if errors.Is(err, moodle.ErrUserAlreadyExists) {
			users, findErr := s.findExistingUser(ctx, data.Email)
			if findErr != nil {
				return true, findErr
			}
			if len(users) == 0 {
				return true, fmt.Errorf("moodle user exists but no user found")
			}
			userID = users[0]["id"]
			return true, s.enrolUserInCourses(ctx, userID, resolveCourses(data.Course))
		}
		return false, err
	}
	userID, err = parseUserID(body)
	if err != nil {
		return false, err
	}

	return false, s.enrolUserInCourses(ctx, userID, resolveCourses(data.Course))
}

func courseCustomField(course string) []struct {
	key, value string
} {
	if course == "" || !isValidCourseKey(course) {
		return nil
	}
	return []struct {
		key, value string
	}{
		{course, "true"},
	}
}

func resolveCourses(course string) []int {
	var result []int
	if id, ok := validCourses[strings.TrimSpace(strings.ToLower(course))]; ok {
		result = append(result, id)
	}
	result = append(result, extraCourses...)
	unique := make([]int, 0, len(result))
	seen := map[int]struct{}{}
	for _, id := range result {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

func isValidCourseKey(course string) bool {
	_, ok := validCourses[course]
	return ok
}

func (s *Service) enrolUserInCourses(ctx context.Context, userID int, courseIDs []int) error {
	for _, courseID := range courseIDs {
		if err := s.enrolUserInCourse(ctx, userID, courseID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) enrolUserInCourse(ctx context.Context, userID, courseID int) error {
	payload := url.Values{
		"enrolments[0][roleid]":   {strconv.Itoa(5)},
		"enrolments[0][userid]":   {strconv.Itoa(userID)},
		"enrolments[0][courseid]": {strconv.Itoa(courseID)},
	}
	_, err := s.callMoodle(ctx, "enrol_manual_enrol_users", payload)
	return err
}

func (s *Service) findExistingUser(ctx context.Context, email string) ([]map[string]int, error) {
	if s.moodleClient == nil {
		return nil, moodle.ErrNotConfigured
	}

	users, err := s.moodleClient.FindUsersByField(ctx, "email", email)
	if err != nil {
		if errors.Is(err, moodle.ErrUserNotFound) {
			return nil, errors.New("usuario no encontrado")
		}
		return nil, err
	}

	result := make([]map[string]int, 0, len(users))
	for _, user := range users {
		result = append(result, map[string]int{"id": user.ID})
	}
	if len(result) == 0 {
		return nil, errors.New("usuario no encontrado")
	}
	return result, nil
}

func parseUserID(body []byte) (int, error) {
	var users []map[string]interface{}
	if err := json.Unmarshal(body, &users); err != nil {
		return 0, err
	}
	if len(users) == 0 {
		return 0, errors.New("moodle response missing user id")
	}
	if id, ok := users[0]["id"]; ok {
		return moodle.ToInt(id)
	}
	return 0, errors.New("invalid moodle id type")
}

func (s *Service) callMoodle(ctx context.Context, function string, extra url.Values) ([]byte, error) {
	return moodle.Call(ctx, s.cfg, s.client, function, extra)
}

func generatePassword(dni string) string {
	return strings.ToUpper(strings.TrimSpace(dni))
}
