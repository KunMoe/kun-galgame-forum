package apiv1

import (
	"slices"
	"strings"
	"testing"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/topic/model"
)

func TestCoversSkipStickers(t *testing.T) {
	sticker, ok := markdown.LegacyStickerHash(2, 1)
	if !ok {
		t.Fatal("sticker 2/1")
	}
	upload := strings.Repeat("c", 64)
	tokens := model.ImageTokens{"/image/" + sticker, "/image/" + upload, "/image/" + sticker + "_320"}

	hashes := func(imgs []repr.Image) []string {
		out := make([]string, 0, len(imgs))
		for _, img := range imgs {
			out = append(out, img.Hash)
		}
		return out
	}
	for name, got := range map[string][]repr.Image{
		"coverImages":         coverImages("https://cdn.example", tokens),
		"coverImagesWithMeta": coverImagesWithMeta("https://cdn.example", tokens, nil),
	} {
		if h := hashes(got); !slices.Equal(h, []string{upload}) {
			t.Errorf("%s = %v, want only the upload", name, h)
		}
	}

	derived := deriveCovers("![](/image/" + sticker + "_320) then /image/" + upload)
	if !slices.Equal(derived, model.ImageTokens{"/image/" + upload}) {
		t.Errorf("deriveCovers = %v, want only the upload", derived)
	}
}
