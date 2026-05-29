package feedback

import (
	"strings"
	"time"

	"github.com/inscripcion-moodle/go-backend/internal/services/cloudfrontsign"
)

// rewritePlaylist rewrites segment lines in a single-rendition HLS playlist,
// replacing each segment URI with a CloudFront signed URL. Lines starting with
// '#' and blank lines are passed through unchanged.
//
// NOTE: URIs embedded inside tag lines (e.g. #EXT-X-KEY:URI="..." or
// #EXT-X-MAP:URI="...") are NOT signed — this only supports unencrypted,
// TS-segment, single-rendition playlists.
func rewritePlaylist(raw, baseURL string, signer *cloudfrontsign.Signer, expires time.Time) (string, error) {
	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		abs := resolveSegmentURL(baseURL, trimmed)
		signed, err := signer.SignURL(abs, expires)
		if err != nil {
			return "", err
		}
		lines[i] = signed
	}
	return strings.Join(lines, "\n"), nil
}

// resolveSegmentURL returns an absolute URL for a segment. If segment is already
// absolute it is returned as-is; otherwise it is resolved relative to the
// directory of baseURL (the playlist URL).
func resolveSegmentURL(baseURL, segment string) string {
	if strings.HasPrefix(segment, "http://") || strings.HasPrefix(segment, "https://") {
		return segment
	}
	slash := strings.LastIndex(baseURL, "/")
	if slash < 0 {
		return segment
	}
	return baseURL[:slash+1] + segment
}
