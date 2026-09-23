package anchor_test

import (
	"testing"

	"kun-galgame-api/internal/community/anchor"
	"kun-galgame-api/pkg/communityclient"
)

func TestResolveKnownAnchors(t *testing.T) {
	refs := []anchor.Ref{
		{Kind: communityclient.AnchorSiteGame, ID: "1207"},
		{Kind: communityclient.AnchorSiteResource, ID: "toolset:9"},
		{Kind: communityclient.AnchorSiteResource, ID: "quiz:3"},
		{Kind: communityclient.AnchorSiteResource, ID: "rating:11"},
		{Kind: communityclient.AnchorSiteResource, ID: "resource:42"},
	}
	got := anchor.New(nil, nil).Resolve(refs)

	want := map[anchor.Ref]string{
		refs[0]: "/galgame/1207",
		refs[1]: "/toolset/9",
		refs[2]: "/galgame-quiz/3",
		refs[3]: "/galgame-rating/11",
		refs[4]: "/galgame/resource/42",
	}
	for ref, link := range want {
		if got[ref].Link != link {
			t.Errorf("%v → %q, want %q", ref, got[ref].Link, link)
		}
	}
	if workID := got[refs[0]].WorkID; workID != 1207 {
		t.Errorf("galgame id = %d, want 1207", workID)
	}
	if workID := got[refs[1]].WorkID; workID != 0 {
		t.Errorf("resource wall carried galgame id %d", workID)
	}
}

// The search and unread faces see walls this forum did not open: a board topic,
// a catalog wall another site anchored, a source retired since. Linking those to
// a page that does not exist is worse than leaving the row out.
func TestResolveDropsUnlinkableAnchors(t *testing.T) {
	refs := []anchor.Ref{
		{Kind: communityclient.AnchorBoard, ID: "1"},
		{Kind: communityclient.AnchorCatalogWork, ID: "88"},
		{Kind: communityclient.AnchorCatalogPerson, ID: "88"},
		{Kind: communityclient.AnchorSiteResource, ID: "wiki:5"},
		{Kind: communityclient.AnchorSiteResource, ID: "toolset:"},
		{Kind: communityclient.AnchorSiteResource, ID: "notaprefix"},
		{Kind: communityclient.AnchorSiteGame, ID: "0"},
		{Kind: communityclient.AnchorSiteGame, ID: "abc"},
	}
	got := anchor.New(nil, nil).Resolve(refs)
	if len(got) != 0 {
		t.Errorf("resolved %d unlinkable anchors: %v", len(got), got)
	}
}
