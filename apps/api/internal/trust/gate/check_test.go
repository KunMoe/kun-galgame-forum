package gate

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"kun-galgame-api/pkg/trustclient"
)

type fakeChecker struct {
	calls  int32
	result *trustclient.CheckResult
	err    error
	block  bool
}

func (f *fakeChecker) Check(ctx context.Context, _ trustclient.CheckRequest) (*trustclient.CheckResult, error) {
	atomic.AddInt32(&f.calls, 1)
	if f.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return f.result, f.err
}

func TestCheckDisabledZeroDial(t *testing.T) {
	s := NewCheckService(nil)
	if s.Enabled() {
		t.Fatal("nil checker must be disabled")
	}
	if v := s.Verdict(context.Background(), "任何内容", nil); v.Decision != DecisionAllow || v.Matched != nil || v.Skipped {
		t.Fatalf("disabled gate = %+v, want unskipped allow", v)
	}
	var nilSvc *CheckService
	if v := nilSvc.Verdict(context.Background(), "x", nil); v.Decision != DecisionAllow || v.Skipped {
		t.Fatalf("nil service = %+v, want unskipped allow", v)
	}
}

func TestCheckDecisionMapping(t *testing.T) {
	cases := []struct {
		wire string
		want string
	}{
		{"deny", DecisionDeny},
		{"hold", DecisionHold},
		{"allow", DecisionAllow},
		{"", DecisionAllow},
	}
	for _, tc := range cases {
		fc := &fakeChecker{result: &trustclient.CheckResult{Decision: tc.wire, Matched: []string{"m"}}}
		s := NewCheckService(fc)
		author := int64(42)
		v := s.Verdict(context.Background(), "内容", &author)
		if v.Decision != tc.want {
			t.Fatalf("wire %q → %q, want %q", tc.wire, v.Decision, tc.want)
		}
		if len(v.Matched) != 1 || v.Matched[0] != "m" {
			t.Fatalf("matched not propagated: %v", v.Matched)
		}
		if v.Skipped {
			t.Fatalf("wire %q: an answered check must not be skipped", tc.wire)
		}
		if fc.calls != 1 {
			t.Fatalf("expected exactly one dial, got %d", fc.calls)
		}
	}
}

func TestCheckFailOpenOnError(t *testing.T) {
	s := NewCheckService(&fakeChecker{err: errors.New("trust 503")})
	if v := s.Verdict(context.Background(), "内容", nil); v.Decision != DecisionAllow || v.Matched != nil || !v.Skipped {
		t.Fatalf("error → %+v, want skipped allow", v)
	}
}

func TestCheckFailOpenOnTimeout(t *testing.T) {
	s := NewCheckService(&fakeChecker{block: true})
	start := time.Now()
	v := s.Verdict(context.Background(), "内容", nil)
	elapsed := time.Since(start)
	if v.Decision != DecisionAllow || !v.Skipped {
		t.Fatalf("timeout → %+v, want skipped allow", v)
	}
	if elapsed > time.Second {
		t.Fatalf("Decision stalled %v; the 500ms bound did not fire", elapsed)
	}
}
