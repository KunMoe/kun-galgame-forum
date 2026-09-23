package service

import (
	"testing"
	"time"

	"kun-galgame-api/pkg/catalogclient"
)

func revision(id int64, workID *int64, actor int64, amender *int64, at time.Time) catalogclient.WorkRevisionFeedItem {
	item := catalogclient.WorkRevisionFeedItem{
		ID: id, ActorUID: actor, AmenderUID: amender,
		Site: "kungal", ProductWorkID: workID, CreatedAt: at,
	}
	if workID != nil {
		item.WorkID = *workID
	}
	return item
}

func TestContributorTouchesFoldsAPage(t *testing.T) {
	t0 := time.Date(2025, 7, 20, 21, 58, 23, 0, time.UTC)
	t1 := t0.Add(48 * time.Hour)
	workID := ptr(int64(4321))

	touches, workIDs := contributorTouches([]catalogclient.WorkRevisionFeedItem{
		revision(1, workID, 61516, nil, t1),
		revision(2, workID, 61516, nil, t0),
	})

	if len(workIDs) != 1 || workIDs[0] != 4321 {
		t.Fatalf("workIDs = %v, want one entry 4321 (the count refresh's target list)", workIDs)
	}
	if len(touches) != 1 {
		t.Fatalf("touches = %d, want the two revisions folded onto one pair", len(touches))
	}
	got := touches[0]
	if got.Count != 2 {
		t.Errorf("count = %d, want 2", got.Count)
	}
	if !got.FirstAt.Equal(t0) {
		t.Errorf("first_at = %v, want the EARLIER timestamp %v — the feed is ordered by id, not time", got.FirstAt, t0)
	}
	if !got.LastAt.Equal(t1) {
		t.Errorf("last_at = %v, want the later timestamp %v", got.LastAt, t1)
	}
}

func TestContributorTouchesCreditsBothIdentities(t *testing.T) {
	now := time.Now().UTC()
	workID := ptr(int64(77))

	touches, _ := contributorTouches([]catalogclient.WorkRevisionFeedItem{
		revision(1, workID, 100, ptr(int64(200)), now),
	})
	if len(touches) != 2 {
		t.Fatalf("touches = %d, want the actor and the amender credited separately", len(touches))
	}

	self, _ := contributorTouches([]catalogclient.WorkRevisionFeedItem{
		revision(2, workID, 100, ptr(int64(100)), now),
	})
	if len(self) != 1 || self[0].Count != 1 {
		t.Fatalf("self-amendment = %+v, want a single contribution", self)
	}
}

func TestContributorTouchesSkipsUnanchoredRevisions(t *testing.T) {
	now := time.Now().UTC()
	for _, workID := range []*int64{nil, ptr(int64(0))} {
		touches, workIDs := contributorTouches([]catalogclient.WorkRevisionFeedItem{
			revision(1, workID, 100, nil, now),
		})
		if len(touches) != 0 || len(workIDs) != 0 {
			t.Errorf("anchor %v yielded %d touches / %d work ids, want none", workID, len(touches), len(workIDs))
		}
	}
	if touches, _ := contributorTouches([]catalogclient.WorkRevisionFeedItem{
		revision(2, ptr(int64(77)), 0, nil, now),
	}); len(touches) != 0 {
		t.Errorf("actor 0 yielded %d touches, want none", len(touches))
	}
}

func TestMaxRevisionIDNeverRewinds(t *testing.T) {
	now := time.Now().UTC()
	workID := ptr(int64(1))
	items := []catalogclient.WorkRevisionFeedItem{
		revision(30, workID, 1, nil, now),
		revision(12, workID, 1, nil, now),
	}
	if got := maxRevisionID(items, 5); got != 30 {
		t.Errorf("maxRevisionID = %d, want 30", got)
	}
	if got := maxRevisionID(items, 99); got != 99 {
		t.Errorf("maxRevisionID = %d, want the cursor held at 99", got)
	}
	if got := maxRevisionID(nil, 7); got != 7 {
		t.Errorf("maxRevisionID(empty) = %d, want 7", got)
	}
}
