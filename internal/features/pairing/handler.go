package pairing

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	appauth "github.com/sofolit/boureminder/api/internal/auth"
	"github.com/sofolit/boureminder/api/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type codeResponse struct {
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
}

// CreateCode handles POST /v1/pair/codes.
func (h *Handler) CreateCode(c *gin.Context) {
	actor := appauth.MustActor(c.Request.Context())
	pc, err := h.svc.CreateCode(c.Request.Context(), actor.UserID)
	if err != nil {
		httpx.WriteErr(c, err)
		return
	}
	httpx.WriteJSON(c, http.StatusOK, codeResponse{Code: pc.Code, ExpiresAt: pc.ExpiresAt})
}

type joinRequest struct {
	Code string `json:"code"`
}

// Join handles POST /v1/pair/join.
func (h *Handler) Join(c *gin.Context) {
	actor := appauth.MustActor(c.Request.Context())
	var req joinRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.WriteErr(c, err)
		return
	}
	if req.Code == "" {
		httpx.WriteErr(c, httpx.ErrCode(http.StatusBadRequest, "invalid_code"))
		return
	}
	coupleID, err := h.svc.Join(c.Request.Context(), actor.UserID, req.Code)
	if err != nil {
		httpx.WriteErr(c, err)
		return
	}
	httpx.WriteJSON(c, http.StatusOK, gin.H{"couple_id": coupleID})
}

type weddingDateRequest struct {
	Month int16  `json:"month"`
	Day   int16  `json:"day"`
	Year  *int16 `json:"year"`
}

// SetWeddingDate handles PUT /v1/couple/wedding-date.
func (h *Handler) SetWeddingDate(c *gin.Context) {
	actor := appauth.MustActor(c.Request.Context())
	if !actor.Paired() {
		httpx.WriteErr(c, httpx.ErrCode(http.StatusConflict, "not_paired"))
		return
	}
	var req weddingDateRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.WriteErr(c, err)
		return
	}
	if req.Month < 1 || req.Month > 12 || req.Day < 1 || req.Day > 31 {
		httpx.WriteErr(c, httpx.ErrCode(http.StatusBadRequest, "invalid_date"))
		return
	}
	couple, err := h.svc.SetWeddingDate(c.Request.Context(), actor.UserID, actor.CoupleID, actor.PartnerID, req.Month, req.Day, req.Year)
	if err != nil {
		httpx.WriteErr(c, err)
		return
	}
	httpx.WriteJSON(c, http.StatusOK, gin.H{
		"wedding_month":     couple.WeddingMonth,
		"wedding_day":       couple.WeddingDay,
		"wedding_year":      couple.WeddingYear,
		"wedding_confirmed": couple.WeddingConfirmed,
	})
}
