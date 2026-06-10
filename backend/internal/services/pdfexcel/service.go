// Package pdfexcel proxies PDF "listas de admitidos" to S3 and invokes the
// AWS Lambda that converts them into a single XLSX (S3-in / S3-out, synchronous).
//
// The job prefix is always derived server-side from the exam id by the caller
// (controllers) as "exam-"+<exam_id>; this package never accepts a free-text
// prefix from clients. AWS credentials and the Lambda api-key stay inside the
// Service and are never surfaced to callers.
package pdfexcel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithy "github.com/aws/smithy-go"

	"github.com/inscripcion-moodle/go-backend/internal/config"
)

// lambdaTimeout must stay below the server WriteTimeout (240s); see server.go.
const lambdaTimeout = 230 * time.Second

// LayoutInfo summarises one REAL layout the Lambda's AI detected: the parsing
// strategy plus the column->index map it chose. Lets the admin spot a new
// format that got mis-mapped without opening CloudWatch.
type LayoutInfo struct {
	Strategy string         `json:"strategy"`
	Rows     int            `json:"rows"`
	Columns  map[string]int `json:"columns"`
}

// LambdaResult is the parsed 200 response from the converter Lambda. It is safe
// to return to clients: it never contains the bucket URL secret or api-key.
type LambdaResult struct {
	TotalRecords  int          `json:"total_records"`
	ExpectedCount *int         `json:"expected_count"`
	Warnings      []string     `json:"warnings"`
	PDFsProcessed []string     `json:"pdfs_processed"`
	Layouts       []LayoutInfo `json:"layouts"`
	// S3Output is never serialized to clients (json:"-") so the bucket/key
	// cannot leak even if a future caller marshals this struct directly.
	S3Output string `json:"-"`
	Ready    bool   `json:"ready"`
	// AlreadyGenerated is true when Process found an existing output and skipped
	// the Lambda invocation; in that case the record/warning counts are unknown
	// (they live only inside the XLSX, not as JSON), so callers must not present
	// TotalRecords=0 as a real conversion count.
	AlreadyGenerated bool `json:"already_generated"`
}

// Service holds the S3 client and Lambda invocation config. Construct with New.
type Service struct {
	s3        *s3.Client
	http      *http.Client
	bucket    string
	lambdaURL string
	apiKey    string
}

// New loads the default AWS config for cfg.AWSRegion and builds an S3 client.
// It returns an error if AWS config cannot be loaded (e.g. no creds in dev);
// callers are expected to treat the Service as optional and degrade gracefully.
func New(ctx context.Context, cfg *config.Config) (*Service, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		return nil, fmt.Errorf("pdfexcel: load aws config: %w", err)
	}
	if cfg.PDFExcelAPIKey == "" {
		// The service still constructs (S3 ops work), but every Process call
		// will hit the Lambda with an empty x-api-key and get a 401. Surface
		// the misconfig at startup instead of at first use.
		log.Print("pdfexcel: warning: API_LAMBDA is empty; PDF->Excel conversion will fail with 401 until it is set")
	}
	return &Service{
		s3:        s3.NewFromConfig(awsCfg),
		http:      &http.Client{Timeout: lambdaTimeout},
		bucket:    cfg.PDFExcelBucket,
		lambdaURL: cfg.PDFExcelLambdaURL,
		apiKey:    cfg.PDFExcelAPIKey,
	}, nil
}

// OutputKey is the deterministic S3 key the Lambda writes its result to.
func (s *Service) OutputKey(prefix string) string {
	return prefix + "/output/" + prefix + ".xlsx"
}

// inputKey is the S3 key a single uploaded PDF lands at.
func (s *Service) inputKey(prefix, filename string) string {
	return prefix + "/input/" + filename
}

// ClearJob deletes every object under <prefix>/ so a fresh batch is not merged
// with stale PDFs from a previous run. It is BEST-EFFORT: an access-denied /
// permissions failure is logged and swallowed (returns nil) so a not-yet-updated
// IAM policy does not block uploads. Only unexpected failures are returned.
func (s *Service) ClearJob(ctx context.Context, prefix string) error {
	listPrefix := prefix + "/"
	paginator := s3.NewListObjectsV2Paginator(s.s3, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(listPrefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if isAccessDenied(err) {
				log.Printf("pdfexcel: ClearJob list denied for prefix=%s (continuing best-effort): %v", prefix, err)
				return nil
			}
			return fmt.Errorf("pdfexcel: list objects: %w", err)
		}
		if len(page.Contents) == 0 {
			continue
		}

		objects := make([]types.ObjectIdentifier, 0, len(page.Contents))
		for _, obj := range page.Contents {
			objects = append(objects, types.ObjectIdentifier{Key: obj.Key})
		}

		_, err = s.s3.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(s.bucket),
			Delete: &types.Delete{Objects: objects, Quiet: aws.Bool(true)},
		})
		if err != nil {
			if isAccessDenied(err) {
				log.Printf("pdfexcel: ClearJob delete denied for prefix=%s (continuing best-effort): %v", prefix, err)
				return nil
			}
			return fmt.Errorf("pdfexcel: delete objects: %w", err)
		}
	}
	return nil
}

// UploadPDF stores one PDF at <prefix>/input/<filename> with content-type
// application/pdf. The caller is responsible for sanitizing filename.
func (s *Service) UploadPDF(ctx context.Context, prefix, filename string, body io.Reader) error {
	// PutObject needs a seekable/length-known body for streaming uploads; buffer it.
	data, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("pdfexcel: read pdf: %w", err)
	}
	_, err = s.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(s.inputKey(prefix, filename)),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/pdf"),
	})
	if err != nil {
		return fmt.Errorf("pdfexcel: put object: %w", err)
	}
	return nil
}

// Process invokes the converter Lambda for prefix. It is idempotent: if the
// output XLSX already exists in S3 it returns Ready=true WITHOUT invoking. The
// Lambda Function URL is never included in any error returned to callers.
//
// calificacion tells the Lambda whether the source PDFs carry a grades/score
// ("Nota") column so it can map it; it is forwarded in the request body.
func (s *Service) Process(ctx context.Context, prefix string, calificacion bool) (*LambdaResult, error) {
	// Idempotency: skip invocation if the output already exists.
	_, err := s.s3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.OutputKey(prefix)),
	})
	if err == nil {
		return &LambdaResult{
			Ready:            true,
			AlreadyGenerated: true,
			S3Output:         "s3://" + s.bucket + "/" + s.OutputKey(prefix),
		}, nil
	}
	if !isNotFound(err) {
		return nil, fmt.Errorf("pdfexcel: head output object: %w", err)
	}

	reqBody, err := json.Marshal(map[string]any{
		"bucket":       s.bucket,
		"prefix":       prefix,
		"calificacion": calificacion,
	})
	if err != nil {
		return nil, fmt.Errorf("pdfexcel: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.lambdaURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("pdfexcel: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)

	resp, err := s.http.Do(req)
	if err != nil {
		// Do not leak the function URL; wrap with a generic message.
		return nil, errors.New("pdfexcel: lambda request failed")
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode != http.StatusOK {
		return nil, mapLambdaStatus(resp.StatusCode)
	}

	var result LambdaResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.New("pdfexcel: invalid lambda response")
	}
	result.Ready = true
	return &result, nil
}

// Download streams the output XLSX for prefix. The returned ReadCloser must be
// closed by the caller. The S3 key/URL is intentionally not exposed.
func (s *Service) Download(ctx context.Context, prefix string) (io.ReadCloser, *int64, error) {
	out, err := s.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.OutputKey(prefix)),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("pdfexcel: get output object: %w", err)
	}
	return out.Body, out.ContentLength, nil
}

// mapLambdaStatus maps a non-200 Lambda status to a descriptive, client-safe
// error. It never embeds the function URL.
func mapLambdaStatus(status int) error {
	switch status {
	case http.StatusUnauthorized:
		return errors.New("pdfexcel: lambda rejected api key")
	case http.StatusBadRequest:
		return errors.New("pdfexcel: lambda rejected request body")
	case http.StatusNotFound:
		return errors.New("pdfexcel: no pdfs found to process")
	case http.StatusInternalServerError:
		return errors.New("pdfexcel: lambda failed to process pdfs")
	default:
		return fmt.Errorf("pdfexcel: lambda returned status %d", status)
	}
}

// isNotFound reports whether err is an S3 NotFound / NoSuchKey error.
func isNotFound(err error) bool {
	var nf *types.NotFound
	if errors.As(err, &nf) {
		return true
	}
	var nsk *types.NoSuchKey
	if errors.As(err, &nsk) {
		return true
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NotFound", "NoSuchKey", "404":
			return true
		}
	}
	// HeadObject on a missing key often surfaces as a generic transport
	// ResponseError with an empty ErrorCode and HTTP 404, not as a typed
	// *types.NotFound. Treat any 404 as not-found so the first-run path (no
	// output yet) correctly proceeds to invoke the Lambda.
	var respErr *awshttp.ResponseError
	if errors.As(err, &respErr) && respErr.HTTPStatusCode() == http.StatusNotFound {
		return true
	}
	return false
}

// isAccessDenied reports whether err is an S3 access-denied / permissions error.
func isAccessDenied(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		if code == "AccessDenied" || code == "AccessDeniedException" || code == "Forbidden" {
			return true
		}
		if strings.Contains(strings.ToLower(code), "accessdenied") {
			return true
		}
	}
	return false
}
