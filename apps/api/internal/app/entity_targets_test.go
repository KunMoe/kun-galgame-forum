package app

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"kun-galgame-api/internal/apiv1/content"
)

func TestCachedPreviewsAskOnlyForMisses(t *testing.T) {
	var asked [][]int
	fail := false
	get := func(_ context.Context, ids []int) (map[int]content.EntityTarget, error) {
		asked = append(asked, slices.Clone(ids))
		if fail {
			return nil, errors.New("down")
		}
		out := map[int]content.EntityTarget{}
		for _, id := range ids {
			if id != 404 {
				out[id] = content.UnavailableTarget(content.EntityRef{Type: content.EntityTag, ID: id})
			}
		}
		return out, nil
	}
	lookup := cached(get, time.Minute)
	ctx := context.Background()

	if got, _ := lookup(ctx, []int{1, 404}); len(got) != 1 {
		t.Fatalf("first read %v", got)
	}
	if got, _ := lookup(ctx, []int{1, 404, 2}); len(got) != 2 {
		t.Fatalf("second read %v", got)
	}
	if want := [][]int{{1, 404}, {2}}; !slices.EqualFunc(asked, want, slices.Equal) {
		t.Errorf("asked %v, want %v", asked, want)
	}

	fail = true
	if _, err := lookup(ctx, []int{3}); err == nil {
		t.Fatal("a failed lookup must surface")
	}
	fail = false
	if got, _ := lookup(ctx, []int{3}); len(got) != 1 {
		t.Errorf("a failure was cached: %v", got)
	}
}
