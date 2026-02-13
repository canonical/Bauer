package middleware

import (
	"crypto/subtle"
	"net/http"
)

const apiUser = "bauer"

func APIBasicAuth(next http.Handler, secret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/health" {
			next.ServeHTTP(w, r)
			return
		}

		username, password, ok := r.BasicAuth()
		if !ok || !secureEquals(username, apiUser) || !secureEquals(password, secret) {
			w.Header().Set("WWW-Authenticate", `Basic realm="bauer-api"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func secureEquals(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
