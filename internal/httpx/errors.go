// Package httpx holds shared HTTP plumbing on top of Gin: the error
// envelope and middleware (recover, request-id, logging, version gate).
package httpx

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIError is the uniform error shape every endpoint returns:
// {"error":{"code","message"}}, per docs/BACKEND_GO.md §7.
type APIError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

// ErrCode builds an APIError with the given HTTP status and stable code.
// message defaults to code when omitted; it's for debugging only, never
// shown to the user (the client maps code -> Bangla copy).
func ErrCode(status int, code string, message ...string) *APIError {
	msg := code
	if len(message) > 0 {
		msg = message[0]
	}
	return &APIError{Status: status, Code: code, Message: msg}
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// WriteErr writes err as a JSON error envelope and aborts the Gin chain.
// Unrecognized errors become a 500 "internal_error"; the original error
// is logged server-side (never in the response) so it stays debuggable.
func WriteErr(c *gin.Context, err error) {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		slog.ErrorContext(c.Request.Context(), "unhandled error",
			"error", err,
			"request_id", RequestID(c),
		)
		apiErr = ErrCode(http.StatusInternalServerError, "internal_error")
	}
	var env errorEnvelope
	env.Error.Code = apiErr.Code
	env.Error.Message = apiErr.Message
	c.AbortWithStatusJSON(apiErr.Status, env)
}
