package client

import (
	"testing"

	"kun-galgame-api/pkg/imageclient"
)

func TestHydrateRosterArt_OneBatchKeyedByURL(t *testing.T) {
	const bust = "https://cdn.test/aa/bb/aabbhash1.webp"
	const figure = "https://cdn.test/cc/dd/ccddhash2.webp"
	const unknown = "https://cdn.test/ee/ff/eeffhash3.webp"

	calls := 0
	c := New("https://catalog.test", "nm_test_key", "")
	c.SetImageMetaResolver(func(hashes []string) map[string]imageclient.ImageMeta {
		calls++
		return map[string]imageclient.ImageMeta{
			"aabbhash1": {Width: 256, Height: 360},
			"ccddhash2": {Width: 700, Height: 900, Thumbhash: "hash"},
		}
	})

	chars := []catWorkCharacter{
		{ID: 1, DisplayName: "A", Image: bust, Figure: figure},
		{ID: 2, DisplayName: "B", Figure: unknown},
		{ID: 3, DisplayName: "C"},
	}
	c.hydrateRosterArt(chars)

	if calls != 1 {
		t.Errorf("resolver called %d times, want exactly 1 for the whole cast", calls)
	}
	if chars[0].ImageMeta.Width != 256 || chars[0].FigureMeta.Height != 900 {
		t.Errorf("character 1 = %+v / %+v", chars[0].ImageMeta, chars[0].FigureMeta)
	}
	if chars[0].FigureMeta.Thumbhash != "hash" {
		t.Errorf("thumbhash = %q, want it carried for the blur-up", chars[0].FigureMeta.Thumbhash)
	}
	if chars[1].FigureMeta != (ArtMeta{}) || chars[2].ImageMeta != (ArtMeta{}) {
		t.Errorf("unresolved art = %+v / %+v, want zero", chars[1].FigureMeta, chars[2].ImageMeta)
	}
}

func TestHydrateRosterArt_NoResolverIsANoOp(t *testing.T) {
	c := New("https://catalog.test", "nm_test_key", "")
	chars := []catWorkCharacter{{ID: 1, Image: "https://cdn.test/aa/bb/hash.webp"}}
	c.hydrateRosterArt(chars)
	if chars[0].ImageMeta != (ArtMeta{}) {
		t.Errorf("ImageMeta = %+v, want zero", chars[0].ImageMeta)
	}
}
