package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

func unauthorizedHandler(msg string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": msg})
	})
}

// JWTMiddleware returns middleware that validates Bearer tokens using JWKS from the OIDC issuer.
// If BAUER_OIDC_ISSUER is not configured, the handler is returned unchanged (bypass mode).
func JWTMiddleware(next http.Handler) http.Handler {
	issuer := os.Getenv("BAUER_OIDC_ISSUER")
	if issuer == "" {
		slog.Info("BAUER_OIDC_ISSUER not set; JWT validation bypassed")
		return next
	}

	audience := os.Getenv("BAUER_OIDC_AUDIENCE")
	jwksURL, err := resolveJWKSURL(issuer)
	if err != nil {
		slog.Error("Failed to resolve JWKS URL from OIDC discovery",
			slog.String("issuer", issuer),
			slog.String("error", err.Error()),
		)
		return unauthorizedHandler("authentication service unavailable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	keySet, err := jwk.Fetch(ctx, jwksURL)
	if err != nil {
		slog.Error("Failed to fetch JWKS",
			slog.String("jwks_url", jwksURL),
			slog.String("error", err.Error()),
		)
		return unauthorizedHandler("authentication service unavailable")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawToken := extractBearerToken(r)
		if rawToken == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "missing Authorization header"})
			return
		}

		parseOpts := []jwt.ParseOption{
			jwt.WithKeySet(keySet),
			jwt.WithIssuer(issuer),
			jwt.WithValidate(true),
		}
		if audience != "" {
			parseOpts = append(parseOpts, jwt.WithAudience(audience))
		}

		if _, err := jwt.ParseString(rawToken, parseOpts...); err != nil {
			slog.Warn("JWT validation failed", slog.String("error", err.Error()))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid or expired token"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) < 7 || !strings.EqualFold(authHeader[:7], "bearer ") {
		return ""
	}
	return authHeader[7:]
}

func resolveJWKSURL(issuer string) (string, error) {
	discoveryURL := strings.TrimRight(issuer, "/") + "/.well-known/openid-configuration"
	resp, err := http.Get(discoveryURL) //nolint:noctx
	if err != nil {
		return "", fmt.Errorf("fetching OIDC discovery document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OIDC discovery endpoint returned status %d", resp.StatusCode)
	}

	var doc struct {
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return "", fmt.Errorf("parsing OIDC discovery document: %w", err)
	}
	if doc.JWKSURI == "" {
		return "", fmt.Errorf("jwks_uri not found in OIDC discovery document")
	}
	return doc.JWKSURI, nil
}
