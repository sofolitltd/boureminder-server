package httpx

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const requestIDKey = "request_id"

// RequestID extracts the request id set by RequestIDMiddleware, or "" if
// none is present (e.g. in a unit test that skips middleware).
func RequestID(c *gin.Context) string {
	id, _ := c.Get(requestIDKey)
	s, _ := id.(string)
	return s
}

// RequestIDMiddleware tags every request with a UUID, echoed back as
// X-Request-Id, for correlating logs across the request lifecycle.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-Id")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(requestIDKey, id)
		c.Header("X-Request-Id", id)
		c.Next()
	}
}

// Recover turns a panic in any downstream handler into a 500 response
// instead of crashing the server, and logs the stack trace.
func Recover(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.ErrorContext(c.Request.Context(), "panic recovered",
					"error", rec,
					"request_id", RequestID(c),
				)
				WriteErr(c, ErrCode(http.StatusInternalServerError, "internal_error"))
			}
		}()
		c.Next()
	}
}

// Logging logs one structured line per request: method, path, status,
// duration, and the correlating request id. Never logs headers or bodies,
// which may carry tokens or private text (docs/CODING_GUIDELINES.md §3.1).
func Logging(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.InfoContext(c.Request.Context(), "request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", RequestID(c),
		)
	}
}

// VersionGate rejects clients older than minVersion with 426 Upgrade
// Required, triggering the app's force-update screen (docs/BACKEND_GO.md
// §7). Requests with no X-App-Version header pass through, since the
// account-delete web page and /healthz never send one.
func VersionGate(minVersion string) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientVersion := c.GetHeader("X-App-Version")
		if clientVersion != "" && compareSemver(clientVersion, minVersion) < 0 {
			WriteErr(c, ErrCode(http.StatusUpgradeRequired, "update_required"))
			return
		}
		c.Next()
	}
}

// compareSemver compares two "x.y.z" versions, returning -1, 0, or 1.
// Malformed segments compare as 0, which is permissive by design: a
// version gate should never hard-fail on an unexpected format.
func compareSemver(a, b string) int {
	av, bv := parseSemver(a), parseSemver(b)
	for i := range 3 {
		if av[i] != bv[i] {
			if av[i] < bv[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

func parseSemver(v string) [3]int {
	var out [3]int
	part := 0
	num := 0
	for _, ch := range v {
		switch {
		case ch >= '0' && ch <= '9':
			num = num*10 + int(ch-'0')
		case ch == '.':
			if part < 3 {
				out[part] = num
			}
			part++
			num = 0
		}
	}
	if part < 3 {
		out[part] = num
	}
	return out
}
