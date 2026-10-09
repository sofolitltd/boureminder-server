// Package auth verifies Firebase ID tokens and resolves them to the
// app's own user/couple identity (docs/BACKEND_GO.md §9, §11.4).
package auth

import (
	"context"
	"net/http"
	"strings"

	fbauth "firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/sofolit/boureminder/api/internal/httpx"
)

// Actor is the authenticated identity attached to every request after
// RequireUser. CoupleID and PartnerID are uuid.Nil when the user hasn't
// paired yet — every couple-scoped query must check for that itself
// rather than assume a non-nil value.
type Actor struct {
	UserID      uuid.UUID
	FirebaseUID string
	CoupleID    uuid.UUID
	PartnerID   uuid.UUID
}

func (a Actor) Paired() bool { return a.CoupleID != uuid.Nil }

type ctxKey int

const (
	firebaseUIDKey ctxKey = iota
	actorKey
)

// UserLookup resolves a verified Firebase UID to this app's user row (and
// partner, if paired). Implemented by internal/features/me so this
// package never imports sqlc directly.
type UserLookup interface {
	ActorByFirebaseUID(ctx context.Context, firebaseUID string) (Actor, error)
}

// ErrUserNotFound means the token verified but no users row exists yet
// — the client should have called POST /v1/me/bootstrap first.
var ErrUserNotFound = errNotFound{}

type errNotFound struct{}

func (errNotFound) Error() string { return "user not bootstrapped" }

// VerifyToken checks the bearer token's signature and attaches the
// verified Firebase UID to the request context. It does NOT require a
// users row to exist, so /v1/me/bootstrap can run it before the user is
// created.
func VerifyToken(fb *fbauth.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || raw == "" {
			httpx.WriteErr(c, httpx.ErrCode(http.StatusUnauthorized, "unauthenticated"))
			return
		}
		tok, err := fb.VerifyIDToken(c.Request.Context(), raw)
		if err != nil {
			httpx.WriteErr(c, httpx.ErrCode(http.StatusUnauthorized, "invalid_token"))
			return
		}
		ctx := context.WithValue(c.Request.Context(), firebaseUIDKey, tok.UID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// FirebaseUID reads the UID attached by VerifyToken.
func FirebaseUID(ctx context.Context) string {
	uid, _ := ctx.Value(firebaseUIDKey).(string)
	return uid
}

// RequireUser builds on VerifyToken: it resolves the verified Firebase
// UID to an Actor and 404s with "user_not_bootstrapped" if no users row
// exists. Every endpoint except /v1/me/bootstrap uses this.
func RequireUser(lookup UserLookup) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := FirebaseUID(c.Request.Context())
		if uid == "" {
			httpx.WriteErr(c, httpx.ErrCode(http.StatusUnauthorized, "unauthenticated"))
			return
		}
		actor, err := lookup.ActorByFirebaseUID(c.Request.Context(), uid)
		if err != nil {
			if err == ErrUserNotFound {
				httpx.WriteErr(c, httpx.ErrCode(http.StatusNotFound, "user_not_bootstrapped"))
				return
			}
			httpx.WriteErr(c, httpx.ErrCode(http.StatusInternalServerError, "internal_error"))
			return
		}
		ctx := context.WithValue(c.Request.Context(), actorKey, actor)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// MustActor reads the Actor attached by RequireUser. It panics if called
// without that middleware, which is a programming error, not a runtime
// condition to handle gracefully.
func MustActor(ctx context.Context) Actor {
	actor, ok := ctx.Value(actorKey).(Actor)
	if !ok {
		panic("auth.MustActor: no actor in context; is RequireUser wired?")
	}
	return actor
}
