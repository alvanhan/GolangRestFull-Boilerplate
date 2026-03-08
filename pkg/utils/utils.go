package utils

import (
	"fmt"
	"mime"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"file-management-service/pkg/crypto"

	"github.com/google/uuid"
)

var (
	// sanitiseRe strips characters that are unsafe or disallowed in filenames.
	sanitiseRe = regexp.MustCompile(`[^\w\s\-.]`)
	// multiSpaceRe collapses consecutive whitespace.
	multiSpaceRe = regexp.MustCompile(`\s+`)
)

// GetFileExtension returns the lowercase extension of filename including the dot,
// e.g. ".pdf". Returns "" if there is no extension.
func GetFileExtension(filename string) string {
	return strings.ToLower(filepath.Ext(filename))
}

// GetMimeType returns the MIME type derived from the filename extension.
// Falls back to "application/octet-stream" for unknown types.
func GetMimeType(filename string) string {
	ext := filepath.Ext(filename)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream"
	}
	// Strip optional params (e.g. "; charset=utf-8")
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}
	return mimeType
}

// SanitizeFilename removes characters that are potentially dangerous in file
// names and trims surrounding whitespace.
func SanitizeFilename(filename string) string {
	// Preserve the extension
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)

	base = sanitiseRe.ReplaceAllString(base, "")
	base = multiSpaceRe.ReplaceAllString(base, "_")
	base = strings.Trim(base, ". ")

	if base == "" {
		base = "file"
	}
	return base + strings.ToLower(ext)
}

// FormatFileSize formats a byte count into a human-readable string like "1.5 MB".
func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func IsAllowedMimeType(mimeType string, allowed []string) bool {
	for _, a := range allowed {
		if strings.EqualFold(a, mimeType) {
			return true
		}
	}
	return false
}

// GenerateStorageKey produces a unique, path-safe storage key for an object
// belonging to ownerID with the given filename.
// Format: <ownerID>/<year>/<month>/<uuid>_<sanitised-filename>
func GenerateStorageKey(ownerID, filename string) string {
	now := time.Now()
	token, err := crypto.GenerateSecureToken(8)
	if err != nil {
		token = uuid.New().String()
	}
	sanitised := SanitizeFilename(filename)
	return fmt.Sprintf("%s/%d/%02d/%s_%s",
		ownerID,
		now.Year(), now.Month(),
		token, sanitised,
	)
}

// ParseDuration is a thin wrapper around time.ParseDuration that also accepts
// day ("d") and week ("w") suffixes.
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "d") {
		n := strings.TrimSuffix(s, "d")
		var days int64
		if _, err := fmt.Sscanf(n, "%d", &days); err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	if strings.HasSuffix(s, "w") {
		n := strings.TrimSuffix(s, "w")
		var weeks int64
		if _, err := fmt.Sscanf(n, "%d", &weeks); err != nil {
			return 0, err
		}
		return time.Duration(weeks) * 7 * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

// TruncateString truncates s to at most max runes, appending "…" if truncated.
func TruncateString(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

func SliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func PtrString(s string) *string   { return &s }
func PtrTime(t time.Time) *time.Time { return &t }
func PtrInt(i int) *int            { return &i }
func PtrInt64(i int64) *int64      { return &i }
