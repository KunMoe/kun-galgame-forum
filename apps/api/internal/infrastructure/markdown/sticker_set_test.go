package markdown

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestIsStickerSeededFromTheLegacyTable(t *testing.T) {
	for _, key := range []stickerKey{{1, 1}, {7, 18}} {
		h, ok := LegacyStickerHash(key.pack, key.position)
		if !ok {
			t.Fatalf("sticker %d/%d", key.pack, key.position)
		}
		if !IsSticker(h) {
			t.Errorf("sticker %d/%d (%s) is not a sticker", key.pack, key.position, h)
		}
	}
	if IsSticker(strings.Repeat("0", 64)) {
		t.Error("an unknown hash is a sticker")
	}
}

var learnRuns atomic.Int64

func TestLearnStickers(t *testing.T) {
	learned := fmt.Sprintf("5e%062x", learnRuns.Add(1))
	if IsSticker(learned) {
		t.Fatal("learned before LearnStickers")
	}
	LearnStickers([]string{learned})
	if !IsSticker(learned) {
		t.Error("a learned hash is not a sticker")
	}
	seed, _ := LegacyStickerHash(1, 1)
	if !IsSticker(seed) {
		t.Error("learning dropped the seed")
	}
}

func TestStickerSetConcurrentLearnAndRead(t *testing.T) {
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() {
			LearnStickers([]string{fmt.Sprintf("c0ffee%058d", i)})
		})
		wg.Go(func() { _ = IsSticker(strings.Repeat("9", 64)) })
	}
	wg.Wait()
}

func TestExtractCoverImagesSkipsStickersBeforeTheCap(t *testing.T) {
	var b strings.Builder
	for pos := 1; pos <= 10; pos++ {
		h, ok := LegacyStickerHash(1, pos)
		if !ok {
			t.Fatalf("sticker 1/%d", pos)
		}
		b.WriteString("![](/image/" + h + "_320) ![](/image/" + h + ") ")
	}
	upload := strings.Repeat("e", 64)
	b.WriteString("![](/image/" + upload + ") ![](/image/" + upload + "_320)")

	got := ExtractCoverImages(b.String(), 9)
	if want := []string{"/image/" + upload}; !slices.Equal(got, want) {
		t.Fatalf("covers %v, want %v", got, want)
	}
}
