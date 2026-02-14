package middleware

import (
	"net/http"

	pkgMiddleware "github.com/fekuna/omnipos-pkg/middleware"
	"github.com/google/uuid"
)

// RequestIDMiddleware is a HTTP middleware that generates a request ID if missing
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get Request ID from header
		reqID := r.Header.Get(pkgMiddleware.RequestIDHeader)

		// Generate new ID if missing
		if reqID == "" {
			reqID = uuid.New().String()
		}

		// Set header in response
		w.Header().Set(pkgMiddleware.RequestIDHeader, reqID)

		// Add to context
		ctx := pkgMiddleware.WithRequestID(r.Context(), reqID)

		// Call next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
