package apiv1

import (
	"testing"
	"time"

	"kun-galgame-api/internal/topic/ai"
)

func TestEditDrift(t *testing.T) {
	cases := []struct {
		before, after string
		want          int
	}{
		{"a\nb\nc", "a\nb\nc", 0},
		{"a\nb", "a\nb\nnew", 3},
		{"keep\ngone\nkeep2", "keep\nkeep2", 4},
		{"one\ntwo", "uno\ntwo", 6},
		{"x\nmiddle\ny\nmiddle2\nz", "x\nMIDDLE\ny\nmiddle2\nz", 12},
	}
	for _, c := range cases {
		if got := editDrift(c.before, c.after); got != c.want {
			t.Errorf("editDrift(%q, %q) = %d, want %d", c.before, c.after, got, c.want)
		}
	}
}

func TestClassificationTokenBindsUserTextAndExpiry(t *testing.T) {
	a := NewTopicAI(nil, nil, nil)
	future := time.Now().Add(time.Minute)
	tok := a.Token(7, "title", "body", ai.Verdict{Section: "tech-help"}, future)
	if got, err := a.Verify(tok, 7, "title", "body"); err != nil || got.Section != "tech-help" || got.IsNSFW {
		t.Fatalf("Verify = %+v, %v", got, err)
	}
	adult := a.Token(7, "title", "body", ai.Verdict{Section: "galgame", IsNSFW: true}, future)
	if got, err := a.Verify(adult, 7, "title", "body"); err != nil || got.Section != "galgame" || !got.IsNSFW {
		t.Fatalf("Verify = %+v, %v", got, err)
	}
	for name, check := range map[string]func() error{
		"other user":  func() error { _, err := a.Verify(tok, 8, "title", "body"); return err },
		"edited body": func() error { _, err := a.Verify(tok, 7, "title", "body!"); return err },
		"other key": func() error {
			_, err := NewTopicAI(nil, nil, nil).Verify(tok, 7, "title", "body")
			return err
		},
		"expired": func() error {
			old := a.Token(7, "title", "body", ai.Verdict{Section: "tech-help"}, time.Now().Add(-time.Second))
			_, err := a.Verify(old, 7, "title", "body")
			return err
		},
		"garbage": func() error { _, err := a.Verify("not-a-token", 7, "title", "body"); return err },
	} {
		if check() == nil {
			t.Errorf("%s: token verified", name)
		}
	}
}
