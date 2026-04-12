package ci

import "context"

// botCommentMarker is a hidden HTML comment embedded in PR comments
// to identify and update bot-created comments idempotently.
const botCommentMarker = "<!-- devbox-preview -->"

// StatusState represents a GitHub commit status state.
type StatusState string

const (
	StatusPending StatusState = "pending"
	StatusSuccess StatusState = "success"
	StatusFailure StatusState = "failure"
	StatusError   StatusState = "error"
)

// CIProvider defines the interface for CI/CD platform integration.
type CIProvider interface {
	// CommentWorkspaceURL posts or updates a PR comment with the workspace URL.
	CommentWorkspaceURL(ctx context.Context, prNumber int, url string) error

	// FindBotComment returns the comment ID of an existing bot comment on a PR.
	// Returns 0 if no bot comment exists.
	FindBotComment(ctx context.Context, prNumber int) (int64, error)

	// UpdateComment updates an existing PR comment by ID.
	UpdateComment(ctx context.Context, commentID int64, body string) error

	// DeleteComment deletes a PR comment by ID.
	DeleteComment(ctx context.Context, commentID int64) error

	// SetCommitStatus sets a commit status check on a given SHA.
	SetCommitStatus(ctx context.Context, sha string, state StatusState, targetURL, description string) error
}
