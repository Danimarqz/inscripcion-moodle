package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	ExamsKey         = "public:exams"
	QuestionsPrefix  = "public:questions"
	SubmissionPrefix = "public:check"
	scanCount        = 250
)

func QuestionsKey(examID uint) string {
	return fmt.Sprintf("%s:%d", QuestionsPrefix, examID)
}

func SubmissionPattern(examID uint) string {
	return fmt.Sprintf("%s:%d:*", SubmissionPrefix, examID)
}

func InvalidateExams(client *redis.Client) {
	if client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = client.Del(ctx, ExamsKey).Err()
}

func InvalidateQuestions(client *redis.Client, examID uint) {
	if client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = client.Del(ctx, QuestionsKey(examID)).Err()
}

func InvalidateCheckCache(client *redis.Client, examID uint) {
	if client == nil {
		return
	}
	ctx := context.Background()
	iter := client.Scan(ctx, 0, SubmissionPattern(examID), scanCount).Iterator()
	for iter.Next(ctx) {
		_ = client.Del(ctx, iter.Val()).Err()
	}
}
