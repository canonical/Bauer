package middleware

import (
	"bauer/cmd/bauer-api/types"
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

func RequestTrace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.NewUUID()
		if err != nil {
			err := types.InternalError(err).Render(w, r)
			if err != nil {
				slog.Error("Failed rendering internal error response due to failed UUID generation: %w",
					slog.String("error", err.Error()),
				)
			}
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, "requestID", id.String())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
