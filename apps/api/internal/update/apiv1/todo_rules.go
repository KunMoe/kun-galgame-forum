package apiv1

import (
	adminModel "kun-galgame-api/internal/admin/model"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/perm"
)

var stateTokens = [...]string{
	adminModel.TodoStatusPending:   "pending",
	adminModel.TodoStatusClaimed:   "in_progress",
	adminModel.TodoStatusDone:      "done",
	adminModel.TodoStatusDiscarded: "discarded",
}

func stateToken(status int) (string, bool) {
	if status < 0 || status >= len(stateTokens) {
		return "", false
	}
	return stateTokens[status], true
}

func stateFromToken(token string) (int, bool) {
	for status, t := range stateTokens {
		if t == token {
			return status, true
		}
	}
	return 0, false
}

type transition struct{ from, to int }

// transitions is the whole state machine. A pair that is not here is refused
// as INVALID_STATE_TRANSITION before any permission is looked at, including
// "move to the state it is already in".
var transitions = map[transition]func(*adminModel.Todo, *middleware.UserInfo) bool{
	{adminModel.TodoStatusPending, adminModel.TodoStatusClaimed}:   canClaim,
	{adminModel.TodoStatusPending, adminModel.TodoStatusDiscarded}: isAuthor,
	{adminModel.TodoStatusClaimed, adminModel.TodoStatusDone}:      claimerOrEditor,
	{adminModel.TodoStatusClaimed, adminModel.TodoStatusDiscarded}: claimerOrEditor,
	{adminModel.TodoStatusClaimed, adminModel.TodoStatusPending}:   isClaimer,
	{adminModel.TodoStatusDiscarded, adminModel.TodoStatusPending}: canReopen,
}

func canClaim(_ *adminModel.Todo, u *middleware.UserInfo) bool {
	return u.Can(perm.UpdateLogEdit)
}

func canReopen(_ *adminModel.Todo, u *middleware.UserInfo) bool {
	return u.Can(perm.UpdateLogReopen)
}

func isAuthor(t *adminModel.Todo, u *middleware.UserInfo) bool {
	return t.UserID == u.ID
}

func isClaimer(t *adminModel.Todo, u *middleware.UserInfo) bool {
	return t.ClaimedUserID != nil && *t.ClaimedUserID == u.ID
}

func claimerOrEditor(t *adminModel.Todo, u *middleware.UserInfo) bool {
	return isClaimer(t, u) || u.Can(perm.UpdateLogEdit)
}

func isEditable(status int) bool {
	return status == adminModel.TodoStatusPending || status == adminModel.TodoStatusClaimed
}

// allowedTargets lists the states the machine offers from `from`, whoever asks,
// in token order; it goes into the 409 detail.
func allowedTargets(from int) []string {
	var out []string
	for status := range stateTokens {
		if _, ok := transitions[transition{from, status}]; ok {
			out = append(out, stateTokens[status])
		}
	}
	return out
}

func mayTransition(t *adminModel.Todo, u *middleware.UserInfo, to int) bool {
	gate, ok := transitions[transition{t.Status, to}]
	return ok && gate(t, u)
}

func todoViewer(t *adminModel.Todo, u *middleware.UserInfo) *TodoViewer {
	if u == nil {
		return nil
	}
	return &TodoViewer{
		CanEdit:     isEditable(t.Status) && isAuthor(t, u),
		CanDelete:   u.Can(perm.UpdateLogDelete),
		CanClaim:    mayTransition(t, u, adminModel.TodoStatusClaimed),
		CanComplete: mayTransition(t, u, adminModel.TodoStatusDone),
		CanDiscard:  mayTransition(t, u, adminModel.TodoStatusDiscarded),
		CanRelease:  t.Status == adminModel.TodoStatusClaimed && mayTransition(t, u, adminModel.TodoStatusPending),
		CanReopen:   t.Status == adminModel.TodoStatusDiscarded && mayTransition(t, u, adminModel.TodoStatusPending),
	}
}
