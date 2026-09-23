package repository

import (
	"regexp"
	"strings"
	"testing"
)

type builderCase struct {
	name string
	sql  string
	args []any
}

// Every builder concatenates SQL and appends its own binds; a miscount silently
// shifts every later parameter in the statement instead of failing loudly.
func TestBuilders_BindEveryPlaceholder(t *testing.T) {
	one, two := []string{"汉化"}, []string{"汉化", "补丁"}

	topicOne, topicOneArgs := topicRelevance(one)
	topicTwo, topicTwoArgs := topicRelevance(two)
	bodyOne, bodyOneArgs := contentRelevance("r.content", one)
	bodyTwo, bodyTwoArgs := contentRelevance("r.content", two)

	for _, c := range []builderCase{
		{"topic/one", topicOne, topicOneArgs},
		{"topic/two", topicTwo, topicTwoArgs},
		{"content/one", bodyOne, bodyOneArgs},
		{"content/two", bodyTwo, bodyTwoArgs},
	} {
		if got, want := strings.Count(c.sql, "?"), len(c.args); got != want {
			t.Errorf("%s: %d placeholders, %d binds\n%s", c.name, got, want, c.sql)
		}
	}
}

// The query is typed with a space; Chinese prose writes the term without one.
func TestAdjacencyPattern_ToleratesTheSeparatorTheProseUses(t *testing.T) {
	if got := adjacencyPattern([]string{"汉化"}); got != "" {
		t.Errorf("single keyword pattern = %q, want empty", got)
	}
	got := adjacencyPattern([]string{"汉化", "补丁"})
	if got != `汉化\s*补丁` {
		t.Fatalf("pattern = %q, want the keywords joined by an optional-space class", got)
	}
	for _, text := range []string{"发布汉化补丁", "发布汉化 补丁", "发布汉化\n补丁"} {
		if !regexp.MustCompile(got).MatchString(text) {
			t.Errorf("%q should match %q", got, text)
		}
	}
	if regexp.MustCompile(got).MatchString("汉化质量好, 需要补丁") {
		t.Error("scattered keywords must not earn the adjacency bonus")
	}
}

func TestAdjacencyPattern_EscapesRegexMetacharacters(t *testing.T) {
	got := adjacencyPattern([]string{"c++", "(beta)"})
	if _, err := regexp.Compile(got); err != nil {
		t.Fatalf("pattern %q does not compile: %v", got, err)
	}
	if !regexp.MustCompile(got).MatchString("c++ (beta)") {
		t.Errorf("%q should match the literal text it was built from", got)
	}
}

func TestTopicRelevance_RanksTitleAboveCategoryAboveBody(t *testing.T) {
	if weightTitle <= weightCategory || weightCategory <= weightContent {
		t.Fatalf("weights out of order: title=%d category=%d content=%d",
			weightTitle, weightCategory, weightContent)
	}
	sql, _ := topicRelevance([]string{"kun"})
	for _, want := range []string{"t.title ILIKE ?", "t.category ILIKE ?", "t.content ILIKE ?"} {
		if !strings.Contains(sql, want) {
			t.Errorf("missing %q in\n%s", want, sql)
		}
	}
}
