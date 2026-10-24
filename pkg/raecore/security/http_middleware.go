package security

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ops"
)

var defaultHTTPMiddlewareLoggerOnce sync.Once
var defaultHTTPMiddlewareLogger *slog.Logger

func getHTTPMiddlewareLogger() *slog.Logger {
	defaultHTTPMiddlewareLoggerOnce.Do(func() {
		svcName := os.Getenv("RAE_SERVICE_NAME")
		if svcName == "" {
			svcName = "rae-suite-go"
		}
		defaultHTTPMiddlewareLogger = ops.NewCanonicalLogger(svcName, os.Getenv("RAE_ENV"), "5.3.7", os.Stdout, slog.LevelInfo)
	})
	return defaultHTTPMiddlewareLogger
}

// WrapSecureHandler wraps an http.Handler with canonical W3C traceparent + slog JSON request logging,
// panic recovery, enterprise security headers (ISO 27001 / OWASP), and optional constant-time API key authentication.
func WrapSecureHandler(next http.Handler) http.Handler {
	core := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		path := r.URL.Path
		if isPublicHealthEndpoint(path) {
			next.ServeHTTP(w, r)
			return
		}

		requireAuth := strings.EqualFold(os.Getenv("RAE_REQUIRE_AUTH"), "true") || os.Getenv("RAE_REQUIRE_AUTH") == "1"
		expectedKey := os.Getenv("RAE_API_KEY")
		if expectedKey == "" {
			expectedKey = os.Getenv("RAE_MASTER_API_KEY")
		}

		if requireAuth && expectedKey != "" {
			provided := extractRequestToken(r)
			if len(provided) == 0 || subtle.ConstantTimeCompare([]byte(provided), []byte(expectedKey)) != 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "unauthorized",
					"code":    "RAE-SEC-AUTH-401",
					"message": "Valid X-API-Key or Authorization: Bearer token required",
				})
				return
			}
		}

		next.ServeHTTP(w, r)
	})
	logger := getHTTPMiddlewareLogger()
	return ops.RequestLoggerMiddleware(logger, ops.RecoveryMiddleware(logger, core))
}

func isPublicHealthEndpoint(path string) bool {
	return path == "/health" || path == "/healthz" || path == "/readyz"
}

func extractRequestToken(r *http.Request) string {
	if key := strings.TrimSpace(r.Header.Get("X-API-Key")); key != "" {
		return key
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}
