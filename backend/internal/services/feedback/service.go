// Package feedback serves CloudFront-signed HLS feedback-video playlists for a
// question, after verifying the requesting student has a submission for the
// exam. It fetches the raw single-rendition .m3u8 from the CloudFront origin
// (cached in Redis) and rewrites every segment line as a signed URL.
package feedback

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"time"

	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/cache"
	"github.com/inscripcion-moodle/go-backend/internal/repository"
	"github.com/inscripcion-moodle/go-backend/internal/services/cloudfrontsign"
)

// Sentinel errors mapped to HTTP statuses by the controller.
var (
	ErrNotConfigured = errors.New("feedback video not configured")
	ErrForbidden     = errors.New("no submission for this student and exam")
	ErrNotFound      = errors.New("no feedback video for this question")
	ErrUpstream      = errors.New("failed to fetch raw playlist from origin")
)

const (
	segmentTTL   = 600 * time.Second
	rawCacheTTL  = 5 * time.Minute
	fetchTimeout = 10 * time.Second
)

// Service holds the collaborators needed to build a signed playlist.
type Service struct {
	db        *gorm.DB
	cache     *cache.Cache
	signer    *cloudfrontsign.Signer
	domain    string
	signerErr string
	subs      repository.SubmissionRepository
	questions repository.QuestionRepository
	client    *http.Client
}

func New(
	db *gorm.DB,
	c *cache.Cache,
	signer *cloudfrontsign.Signer,
	domain, signerErr string,
	subs repository.SubmissionRepository,
	questions repository.QuestionRepository,
) *Service {
	return &Service{
		db:        db,
		cache:     c,
		signer:    signer,
		domain:    domain,
		signerErr: signerErr,
		subs:      subs,
		questions: questions,
		client:    &http.Client{Timeout: fetchTimeout},
	}
}

// SignedPlaylist verifies the student, loads the question and returns the raw
// playlist with every segment rewritten as a CloudFront signed URL. email and
// dni must already be normalized by the caller.
func (s *Service) SignedPlaylist(ctx context.Context, questionID uint, email, dni string, examID uint) ([]byte, error) {
	if s.signer == nil {
		return nil, fmt.Errorf("%w: %s", ErrNotConfigured, s.signerErr)
	}

	ok, err := s.subs.ExistsByStudentExam(ctx, s.db, email, dni, examID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrForbidden
	}

	question, err := s.questions.FindByID(ctx, s.db, questionID)
	if err != nil {
		return nil, ErrNotFound
	}
	// The question must belong to the exam the student is authorized for, and
	// must actually have a video key.
	if question.FeedbackVideoKey == nil || question.ExamID != examID {
		return nil, ErrNotFound
	}

	key := *question.FeedbackVideoKey
	m3u8URL := fmt.Sprintf("https://%s/%s/%s.m3u8", s.domain, key, path.Base(key))
	expires := time.Now().Add(segmentTTL)

	raw, err := s.rawPlaylist(ctx, key, m3u8URL, expires)
	if err != nil {
		return nil, err
	}

	rewritten, err := rewritePlaylist(string(raw), m3u8URL, s.signer, expires)
	if err != nil {
		return nil, err
	}
	return []byte(rewritten), nil
}

// rawPlaylist returns the unsigned .m3u8 bytes, from Redis if present, otherwise
// fetched from the (signed) origin URL and cached. Signed segment URLs are never
// cached — only the raw playlist is.
func (s *Service) rawPlaylist(ctx context.Context, key, m3u8URL string, expires time.Time) ([]byte, error) {
	cacheKey := "hls:raw:" + key
	if raw, hit := s.cache.Get(ctx, cacheKey); hit {
		return raw, nil
	}

	// The playlist object itself lives behind the same restricted behavior, so
	// the fetch must be signed too (mirrors filter/s3video/playlist.php).
	signedURL, err := s.signer.SignURL(m3u8URL, expires)
	if err != nil {
		return nil, err
	}

	raw, err := s.fetch(ctx, signedURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUpstream, err)
	}
	s.cache.Set(ctx, cacheKey, raw, rawCacheTTL)
	return raw, nil
}

func (s *Service) fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("origin returned %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
