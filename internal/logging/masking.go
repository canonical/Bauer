package logging

import "path/filepath"

// MaskSecret returns a masked version of a secret string.
// Empty string returns "<unset>". Short strings (≤4 chars) return "****".
// Longer strings return the first 4 chars + "..." (e.g. "ghp_...").
func MaskSecret(s string) string {
	if s == "" {
		return "<unset>"
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:4] + "..."
}

// MaskPath returns a masked filesystem path showing only the filename.
// Avoids leaking directory structure in logs.
func MaskPath(path string) string {
	if path == "" {
		return "<unset>"
	}
	return ".../" + filepath.Base(path)
}
