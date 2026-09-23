package service

import (
	"encoding/json"
	"strings"
	"testing"
)

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
