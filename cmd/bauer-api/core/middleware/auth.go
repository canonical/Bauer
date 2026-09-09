package middleware

import (
	"bauer/cmd/bauer-api/types"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// RequireAPIToken rejects any request that does not present the expected token
// as an "Authorization: Bearer <token>" header. It fails closed: if no token is
// configured on the server, every request is denied rather than left open.
func RequireAPIToken(expected string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if expected == "" {
				slog.Error("API token not configured; denying request")
				if err := types.InternalError(fmt.Errorf("API token not configured")).Render(w, r); err != nil {
					slog.Error("error writing response", "error", err.Error())
				}
				return
			}

			provided := bearerToken(r.Header.Get("Authorization"))
			if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
				if err := types.Unauthorized(fmt.Errorf("invalid or missing API token")).Render(w, r); err != nil {
					slog.Error("error writing response", "error", err.Error())
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if len(header) >= len(prefix) && strings.EqualFold(header[:len(prefix)], prefix) {
		return strings.TrimSpace(header[len(prefix):])
	}
	return ""
}
