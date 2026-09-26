package push

import (
	"testing"

	"kun-galgame-api/pkg/communityclient"
)

func TestItemDiffers(t *testing.T) {
	hash := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	work := int64(88)
	want := communityclient.ActivityWriteItem{
		ActorID: 7, Verb: "publish", ObjectKind: "topic", ObjectLabel: "话题",
		Title: "T", Excerpt: "e", URL: "https://www.kungal.com/topic/1",
		CoverImageHash: hash, WorkID: work, ContentLimit: "sfw",
		OccurredAt: "2026-09-26T04:30:00.000000Z", Notify: true,
	}
	have := communityclient.SiteActivityView{
		ActorID: 7, Verb: "publish", ObjectKind: "topic", ObjectLabel: "话题",
		Title: "T", Excerpt: "e", URL: "https://www.kungal.com/topic/1",
		CoverImageHash: &hash, WorkID: &work, ContentLimit: "sfw",
		OccurredAt: "2026-09-26T04:30:00.000000Z", Notify: false, Removed: false,
	}
	if itemDiffers(want, have) {
		t.Fatal("identical fields (notify differs) must not differ")
	}
	have.ContentLimit = "nsfw"
	if !itemDiffers(want, have) {
		t.Fatal("content_limit change must differ")
	}
	have.ContentLimit = "sfw"
	have.Removed = true
	if !itemDiffers(want, have) {
		t.Fatal("stored removed vs desired live must differ")
	}
	have.Removed = false
	other := "2026-09-26T04:30:00Z"
	have.OccurredAt = other
	if itemDiffers(want, have) {
		t.Fatal("occurred_at same instant must match")
	}
}
