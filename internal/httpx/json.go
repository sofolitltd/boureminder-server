package httpx

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// DecodeJSON decodes c's JSON body into v, rejecting unknown fields so
// typos in client payloads fail loudly instead of being silently ignored.
func DecodeJSON(c *gin.Context, v any) error {
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("decode request body: %w", ErrCode(http.StatusBadRequest, "invalid_body", err.Error()))
	}
	return nil
}

// WriteJSON writes v as a JSON response with the given status code.
func WriteJSON(c *gin.Context, status int, v any) {
	c.JSON(status, v)
}
