package apiv1

import (
	"context"
	"testing"

	"kun-galgame-api/internal/activity/repository"
)

func TestRatingActivityMasksMarkdownSpoilers(t *testing.T) {
	s := &Service{}
	b := &batch{ratings: map[int]repository.RatingActivity{
		7: {SpoilerLevel: "none", ShortSummary: "**音乐很好**，||秘密结局||\n\n:::spoiler\n整段秘密\n:::"},
	}}

	activity := s.build(context.Background(), repository.FeedRow{
		RowID: 1, SourceID: 7, TypeStr: "GALGAME_RATING_CREATION",
	}, b)
	if activity == nil || activity.GalgameRating == nil {
		t.Fatal("rating activity was not built")
	}
	if got := activity.GalgameRating.ShortSummary; got != "音乐很好，███\n███" {
		t.Fatalf("rating excerpt = %q", got)
	}
}
