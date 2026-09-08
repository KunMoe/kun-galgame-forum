package service

import (
	"strings"
	"testing"
)

func TestCommentSourcePageLink(t *testing.T) {
	id := 1207
	cases := []struct {
		src  CommentSource
		want string
	}{
		{SourceRating(), "/galgame-rating/1207"},
		{SourceToolset(), "/toolset/1207"},
		{SourceResource(), "/galgame/resource/1207"},
		{SourceQuiz(), "/galgame-quiz/1207"},
		// website source deliberately has no prefix and does not use this method
		{SourceWebsite(), "1207"},
	}
	for _, tc := range cases {
		got := tc.src.pageLink(id)
		if got != tc.want {
			t.Errorf("%s.pageLink(%d) = %q, want %q", tc.src.key, id, got, tc.want)
		}
		if strings.HasPrefix(got, "/galgame-resource/") {
			t.Errorf("%s.pageLink(%d) = %q begins with /galgame-resource/", tc.src.key, id, got)
		}
	}
}
