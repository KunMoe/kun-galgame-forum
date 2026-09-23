package service

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/testdb"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/trustclient"
	"kun-galgame-api/pkg/userclient"
)

type scriptedChecker struct{ decision string }

func (c scriptedChecker) Check(_ context.Context, _ trustclient.CheckRequest) (*trustclient.CheckResult, error) {
	return &trustclient.CheckResult{Decision: c.decision, Matched: []string{"坏词"}}, nil
}

type captureScanner struct {
	got  trustclient.ScanRequest
	done chan struct{}
}

func (s *captureScanner) Scan(_ context.Context, req trustclient.ScanRequest) (*trustclient.ScanResult, error) {
	s.got = req
	s.done <- struct{}{}
	return &trustclient.ScanResult{ScanID: 1}, nil
}

func TestQuizAnswerModerationText(t *testing.T) {
	cases := []struct {
		name      string
		qtype     string
		submitted string
		want      string
	}{
		{"single is index-only", quizTypeSingle, `{"value":2}`, ""},
		{"multiple is index-only", quizTypeMultiple, `{"values":[0,1]}`, ""},
		{"judge is boolean-only", quizTypeJudge, `{"value":true}`, ""},
		{"fill carries text", quizTypeFill, `{"values":["Fate","stay night"]}`, "Fate\nstay night"},
		{"essay carries text", quizTypeEssay, `{"text":"这是我的作答"}`, "这是我的作答"},
		{"fill blank/whitespace trims to empty", quizTypeFill, `{"values":["  ",""]}`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := quizAnswerModerationText(tc.qtype, json.RawMessage(tc.submitted))
			if got != tc.want {
				t.Fatalf("quizAnswerModerationText(%s) = %q, want %q", tc.qtype, got, tc.want)
			}
		})
	}
}

func TestQuizAuthoringModerationText(t *testing.T) {
	text := quizAuthoringModerationText(
		"题目问题", "题目描述", "答案解析",
		quizTypeSingle, json.RawMessage(`{"options":["选项甲","选项乙"],"answer":1}`),
	)
	for _, want := range []string{"题目问题", "题目描述", "答案解析", "选项甲", "选项乙"} {
		if !strings.Contains(text, want) {
			t.Fatalf("authoring text %q missing %q", text, want)
		}
	}
	judge := quizContentModerationText(quizTypeJudge, json.RawMessage(`{"answer":true}`))
	if judge != "" {
		t.Fatalf("judge content should carry no free text, got %q", judge)
	}
}

func TestRatingCreateDenyAndScan(t *testing.T) {
	db := testdb.Open(t)

	const workID = 2_000_000_777
	const uid = 2_000_000_778
	cleanup := func() {
		db.Exec("DELETE FROM galgame_rating WHERE work_id = ? AND user_id = ?", workID, uid)
		db.Exec("DELETE FROM galgame WHERE id = ?", workID)
	}
	cleanup()
	defer cleanup()

	ratingRepo := repository.NewRatingRepository(db)
	galgameClient := client.New("", "nm_test", "")
	uc := userclient.New(userclient.Config{})

	reqOf := func() *dto.CreateRatingRequest {
		return &dto.CreateRatingRequest{
			WorkID: workID, Recommend: "recommend", Overall: 8,
			GalgameType: []string{"adv"}, PlayStatus: "played",
			ShortSummary: "这是一段用于审核测试的评测正文",
		}
	}

	denySvc := NewRatingService(ratingRepo, galgameClient, uc,
		gate.NewCheckService(scriptedChecker{decision: gate.DecisionDeny}),
		gate.NewScanService(nil))
	if _, appErr := denySvc.CreateRating(context.Background(), uid, reqOf()); appErr == nil || appErr.StatusCode != 422 {
		t.Fatalf("deny: want 422 error, got %v", appErr)
	}
	var cnt int64
	db.Model(&model.GalgameRating{}).Where("work_id = ? AND user_id = ?", workID, uid).Count(&cnt)
	if cnt != 0 {
		t.Fatalf("deny persisted %d rating row(s), want 0", cnt)
	}

	fs := &captureScanner{done: make(chan struct{}, 1)}
	okSvc := NewRatingService(ratingRepo, galgameClient, uc,
		gate.NewCheckService(scriptedChecker{decision: gate.DecisionAllow}),
		gate.NewScanService(fs))
	created, appErr := okSvc.CreateRating(context.Background(), uid, reqOf())
	if appErr != nil {
		t.Fatalf("allow: unexpected error %v", appErr)
	}
	select {
	case <-fs.done:
	case <-time.After(2 * time.Second):
		t.Fatal("scan goroutine never fired after create")
	}
	if fs.got.SubjectKind != gate.SubjectKindGalgameRating {
		t.Fatalf("scan kind = %q, want %q", fs.got.SubjectKind, gate.SubjectKindGalgameRating)
	}
	if fs.got.SubjectID != strconv.Itoa(created.ID) {
		t.Fatalf("scan subject_id = %q, want %q", fs.got.SubjectID, strconv.Itoa(created.ID))
	}
	if fs.got.Text != reqOf().ShortSummary {
		t.Fatalf("scan text = %q, want %q", fs.got.Text, reqOf().ShortSummary)
	}
}
