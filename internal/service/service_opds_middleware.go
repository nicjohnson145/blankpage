package service

import (
	"net/http"

	"github.com/go-logr/logr"
)

func ContextLoggerMiddleware(rootLogger logr.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a shallow copy of our request, but with a logger in the context
			newReq := r.WithContext(logr.NewContext(r.Context(), rootLogger))
			// Forward it on
			next.ServeHTTP(w, newReq)
		})
	}
}

func MethodLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := logr.FromContextOrDiscard(r.Context())
		logger.Info("request recieved", "path", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
