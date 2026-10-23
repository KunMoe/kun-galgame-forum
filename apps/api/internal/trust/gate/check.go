package gate

import (
	"context"
	"log/slog"
	"time"

	"kun-galgame-api/pkg/trustclient"
)

const checkTimeout = 500 * time.Millisecond

const (
	DecisionAllow = "allow"
	DecisionDeny  = "deny"
	DecisionHold  = "hold"
)

type Checker interface {
	Check(ctx context.Context, req trustclient.CheckRequest) (*trustclient.CheckResult, error)
}

type CheckService struct {
	ck Checker
}

func NewCheckService(ck Checker) *CheckService {
	return &CheckService{ck: ck}
}

func (s *CheckService) Enabled() bool { return s != nil && s.ck != nil }

// Verdict is the synchronous check's answer. Skipped means the check failed
// and the content was let through unchecked; the follow-up scan must say so,
// or trust never runs the word list over it again.
type Verdict struct {
	Decision string
	Matched  []string
	Skipped  bool
}

var Allowed = Verdict{Decision: DecisionAllow}

func (s *CheckService) Verdict(ctx context.Context, text string, authorID *int64) Verdict {
	if !s.Enabled() {
		return Allowed
	}
	cctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	res, err := s.ck.Check(cctx, trustclient.CheckRequest{Text: text, AuthorID: authorID})
	if err != nil {
		slog.Warn("trust check fail-open", "err", err)
		return Verdict{Decision: DecisionAllow, Skipped: true}
	}
	switch res.Decision {
	case DecisionDeny, DecisionHold:
		return Verdict{Decision: res.Decision, Matched: res.Matched}
	default:
		return Verdict{Decision: DecisionAllow, Matched: res.Matched}
	}
}
