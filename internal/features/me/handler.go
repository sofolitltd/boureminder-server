package me

import (
	"net/http"

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

// Bootstrap handles POST /v1/me/bootstrap. It runs after VerifyToken but
// before RequireUser, since the users row may not exist yet.
func (h *Handler) Bootstrap(c *gin.Context) {
	uid := appauth.FirebaseUID(c.Request.Context())
	user, err := h.svc.Bootstrap(c.Request.Context(), uid)
	if err != nil {
		httpx.WriteErr(c, err)
		return
	}
	httpx.WriteJSON(c, http.StatusOK, toProfile(user))
}

// Get handles GET /v1/me.
func (h *Handler) Get(c *gin.Context) {
	actor := appauth.MustActor(c.Request.Context())
	user, err := h.svc.q.GetUser(c.Request.Context(), actor.UserID)
	if err != nil {
		httpx.WriteErr(c, err)
		return
	}
	httpx.WriteJSON(c, http.StatusOK, toProfile(user))
}

type patchRequest struct {
	DisplayName   *string `json:"display_name"`
	Nickname      *string `json:"nickname"`
	Role          *string `json:"role"`
	BirthdayMonth *int16  `json:"birthday_month"`
	BirthdayDay   *int16  `json:"birthday_day"`
	BirthdayYear  *int16  `json:"birthday_year"`
	PhotoKey      *string `json:"photo_key"`
}

// Patch handles PATCH /v1/me.
func (h *Handler) Patch(c *gin.Context) {
	actor := appauth.MustActor(c.Request.Context())
	var req patchRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.WriteErr(c, err)
		return
	}
	if req.Role != nil && *req.Role != "wife" && *req.Role != "husband" {
		httpx.WriteErr(c, httpx.ErrCode(http.StatusBadRequest, "invalid_role"))
		return
	}
	user, err := h.svc.UpdateProfile(c.Request.Context(), actor.UserID, UpdateProfileInput{
		DisplayName:   req.DisplayName,
		Nickname:      req.Nickname,
		Role:          req.Role,
		BirthdayMonth: req.BirthdayMonth,
		BirthdayDay:   req.BirthdayDay,
		BirthdayYear:  req.BirthdayYear,
		PhotoKey:      req.PhotoKey,
	})
	if err != nil {
		httpx.WriteErr(c, err)
		return
	}
	httpx.WriteJSON(c, http.StatusOK, toProfile(user))
}

type registerDeviceRequest struct {
	Token      string `json:"token"`
	Platform   string `json:"platform"`
	AppVersion string `json:"app_version"`
	Locale     string `json:"locale"`
}

// RegisterDevice handles POST /v1/devices.
func (h *Handler) RegisterDevice(c *gin.Context) {
	actor := appauth.MustActor(c.Request.Context())
	var req registerDeviceRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.WriteErr(c, err)
		return
	}
	if req.Token == "" || (req.Platform != "android" && req.Platform != "ios") {
		httpx.WriteErr(c, httpx.ErrCode(http.StatusBadRequest, "invalid_device"))
		return
	}
	if err := h.svc.RegisterDevice(c.Request.Context(), actor.UserID, req.Token, req.Platform, req.AppVersion, req.Locale); err != nil {
		httpx.WriteErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// UnregisterDevice handles DELETE /v1/devices/{token}.
func (h *Handler) UnregisterDevice(c *gin.Context) {
	token := c.Param("token")
	if err := h.svc.UnregisterDevice(c.Request.Context(), token); err != nil {
		httpx.WriteErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
