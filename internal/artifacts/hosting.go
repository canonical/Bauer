package artifacts

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScreenshotHost uploads or serves a screenshot and returns its public URL.
type ScreenshotHost interface {
	Host(ctx context.Context, localPath string) (publicURL string, err error)
}

// LocalFileServer serves screenshots from the artifact directory.
// BaseURL is the externally reachable URL prefix for the server.
// This implementation computes a URL from a local file path — the caller is
// responsible for ensuring the directory is actually being served at BaseURL.
type LocalFileServer struct {
	BaseURL  string // e.g. "https://bauer.example.com/static"
	ServeDir string // path to the artifact root (relative or absolute)
}

func (s *LocalFileServer) Host(_ context.Context, localPath string) (string, error) {
	rel, err := filepath.Rel(s.ServeDir, localPath)
	if err != nil {
		return "", fmt.Errorf("path %q is not under serve directory %q", localPath, s.ServeDir)
	}
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", fmt.Errorf("path %q is not under serve directory %q", localPath, s.ServeDir)
	}
	return strings.TrimRight(s.BaseURL, "/") + "/" + filepath.ToSlash(rel), nil
}

// NopHost is a no-op implementation that returns the local path unchanged.
// Used when no hosting backend is configured (Stage 1 / CLI mode).
type NopHost struct{}

func (n *NopHost) Host(_ context.Context, localPath string) (string, error) {
	return localPath, nil
}

// S3Host is a stub for future S3 screenshot hosting.
// Implement when cloud deployment is ready.
type S3Host struct {
	Bucket string
	Region string
}

func (s *S3Host) Host(_ context.Context, localPath string) (string, error) {
	return "", fmt.Errorf("S3 screenshot hosting is not yet implemented; unset BAUER_S3_BUCKET to use local file serving")
}

// HostFromEnv returns a ScreenshotHost configured from environment variables.
// Priority: BAUER_STATIC_BASE_URL → BAUER_S3_BUCKET → NopHost.
func HostFromEnv(serveDir string) ScreenshotHost {
	if baseURL := os.Getenv("BAUER_STATIC_BASE_URL"); baseURL != "" {
		return &LocalFileServer{BaseURL: baseURL, ServeDir: serveDir}
	}
	// S3 stub: if BAUER_S3_BUCKET is set, return S3Host (not yet functional)
	if bucket := os.Getenv("BAUER_S3_BUCKET"); bucket != "" {
		return &S3Host{Bucket: bucket, Region: os.Getenv("BAUER_S3_REGION")}
	}
	return &NopHost{}
}
