package gate

import (
	"context"
	"errors"
	"testing"
	"time"

	"kun-galgame-api/pkg/trustclient"
)

type fakeScanner struct {
	got  trustclient.ScanRequest
	err  error
	done chan struct{}
}

func newFakeScanner(err error) *fakeScanner {
	return &fakeScanner{err: err, done: make(chan struct{}, 1)}
}

func (f *fakeScanner) Scan(_ context.Context, req trustclient.ScanRequest) (*trustclient.ScanResult, error) {
	f.got = req
	f.done <- struct{}{}
	if f.err != nil {
		return nil, f.err
	}
	return &trustclient.ScanResult{ScanID: 1}, nil
}

func (f *fakeScanner) wait(t *testing.T) {
	t.Helper()
	select {
	case <-f.done:
	case <-time.After(2 * time.Second):
		t.Fatal("scan goroutine never fired")
	}
}

func TestScanDisabledZeroDial(t *testing.T) {
	s := NewScanService(nil)
	if s.Enabled() {
		t.Fatal("nil scanner must be disabled")
	}
	s.ScanBg(SubjectKindTopic, "1", "x", 7, false)
	var nilSvc *ScanService
	nilSvc.ScanBg(SubjectKindReply, "1", "x", 7, false)
}

func TestScanBgPayload(t *testing.T) {
	fs := newFakeScanner(nil)
	s := NewScanService(fs)
	s.ScanBg(SubjectKindTopic, "8841", "标题\n\n正文", 99, false)
	fs.wait(t)
	if fs.got.SubjectKind != "forum_topic" || fs.got.SubjectID != "8841" {
		t.Fatalf("bad subject: %+v", fs.got)
	}
	if fs.got.Text != "标题\n\n正文" {
		t.Fatalf("bad text: %q", fs.got.Text)
	}
	if fs.got.AuthorID == nil || *fs.got.AuthorID != 99 {
		t.Fatalf("bad author: %+v", fs.got.AuthorID)
	}
}

func TestScanBgOmitsZeroAuthor(t *testing.T) {
	fs := newFakeScanner(nil)
	NewScanService(fs).ScanBg(SubjectKindReply, "5", "正文", 0, false)
	fs.wait(t)
	if fs.got.AuthorID != nil {
		t.Fatalf("author 0 should be omitted, got %v", *fs.got.AuthorID)
	}
}

func TestScanBgFailureNoPanic(t *testing.T) {
	fs := newFakeScanner(errors.New("trust down"))
	NewScanService(fs).ScanBg(SubjectKindTopic, "1", "x", 1, false)
	fs.wait(t)
}

func TestScanCarriesCheckSkipped(t *testing.T) {
	cases := []struct {
		name    string
		checker *fakeChecker
		want    bool
	}{
		{"check errored", &fakeChecker{err: errors.New("trust 503")}, true},
		{"check timed out", &fakeChecker{block: true}, true},
		{"check allowed", &fakeChecker{result: &trustclient.CheckResult{Decision: DecisionAllow}}, false},
		{"check held", &fakeChecker{result: &trustclient.CheckResult{Decision: DecisionHold}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := NewCheckService(tc.checker).Verdict(context.Background(), "正文", nil)
			fs := newFakeScanner(nil)
			NewScanService(fs).ScanBg(SubjectKindReply, "5", "正文", 1, v.Skipped)
			fs.wait(t)
			if fs.got.CheckSkipped != tc.want {
				t.Fatalf("CheckSkipped = %v, want %v", fs.got.CheckSkipped, tc.want)
			}
		})
	}
}
