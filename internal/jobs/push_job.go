// Package jobs holds River job args and workers: push notifications, deed
// unlock, bazar due-date checks, wrapped reports, exports, and cleanup
// (docs/BACKEND_GO.md §3).
package jobs

import (
	"context"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
)

// PushArgs sends an FCM notification to one or more users.
// NotificationKind selects the notification copy (CONTENT_BN.md §2) and
// Route is the deep-link path the client opens on tap.
type PushArgs struct {
	UserIDs          []uuid.UUID    `json:"user_ids"`
	NotificationKind string         `json:"notification_kind"`
	Route            string         `json:"route,omitempty"`
	Data             map[string]any `json:"data,omitempty"`
}

func (PushArgs) Kind() string { return "push" }

type PushWorker struct {
	river.WorkerDefaults[PushArgs]
}

func (w *PushWorker) Work(ctx context.Context, job *river.Job[PushArgs]) error {
	// TODO(M1): resolve device tokens for job.Args.UserIDs and send via
	// internal/push.FCM, dropping stale tokens on InvalidRegistration.
	return nil
}
