package apiv1

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/stickerclient"
)

var errUnconfigured = errors.New("apiv1 stickers: service is not configured")

const (
	cacheControl = "private, max-age=3600"
	freshFor     = time.Hour
	retryAfter   = 5 * time.Minute
	buildTimeout = 30 * time.Second
)

type packSource interface {
	OfficialPacks(ctx context.Context) ([]stickerclient.Pack, error)
}

type snapshot struct {
	list    repr.List[StickerPack]
	etag    string
	expires time.Time
}

type Service struct {
	source packSource
	images func([]string) map[string]imageclient.ImageMeta
	cdn    string
	now    func() time.Time

	mu   sync.Mutex
	snap *snapshot
}

func New(source packSource, images func([]string) map[string]imageclient.ImageMeta, cdn string) *Service {
	return &Service{source: source, images: images, cdn: cdn, now: time.Now}
}

func (s *Service) listStickerPacks(ctx context.Context, in *listStickerPacksInput) (*listStickerPacksOutput, error) {
	if s == nil || s.source == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	snap, err := s.current(ctx)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	out := &listStickerPacksOutput{ETag: snap.etag, CacheControl: cacheControl}
	if matchesAny(in.IfNoneMatch, snap.etag) {
		out.Status = http.StatusNotModified
		return out, nil
	}
	out.Body = snap.list
	return out, nil
}

func (s *Service) Refresh() {
	if s == nil || s.source == nil {
		return
	}
	if _, err := s.current(context.Background()); err != nil {
		slog.Warn("sticker packs: refresh failed", "error", err)
	}
}

func (s *Service) current(ctx context.Context) (*snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if s.snap != nil && now.Before(s.snap.expires) {
		return s.snap, nil
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), buildTimeout)
	defer cancel()
	fresh, err := s.build(ctx)
	if err != nil {
		if s.snap == nil {
			return nil, err
		}
		slog.Warn("sticker packs: refresh failed, serving the last good list", "error", err)
		s.snap.expires = now.Add(retryAfter)
		return s.snap, nil
	}
	fresh.expires = now.Add(freshFor)
	s.snap = fresh
	return fresh, nil
}

func (s *Service) build(ctx context.Context) (*snapshot, error) {
	packs, err := s.source.OfficialPacks(ctx)
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(packs, func(a, b stickerclient.Pack) int {
		return cmp.Or(a.CreatedAt.Compare(b.CreatedAt), strings.Compare(a.ID, b.ID))
	})

	var hashes []string
	for _, p := range packs {
		for _, st := range p.Stickers {
			hashes = append(hashes, st.Image.Hash)
		}
	}
	markdown.LearnStickers(hashes)
	metas := map[string]imageclient.ImageMeta{}
	if s.images != nil {
		metas = s.images(hashes)
	}

	items := make([]StickerPack, 0, len(packs))
	for _, p := range packs {
		name := packName(p.Title)
		stickers := make([]Sticker, 0, len(p.Stickers))
		for _, st := range p.Stickers {
			meta, ok := metas[st.Image.Hash]
			if !ok {
				meta = imageclient.ImageMeta{Width: st.Image.Width, Height: st.Image.Height}
			}
			img := repr.NewImage(s.cdn, st.Image.Hash, &meta)
			if img == nil {
				continue
			}
			stickers = append(stickers, Sticker{
				Object:      "sticker",
				ID:          st.ID,
				DisplayName: name + " - " + strconv.Itoa(st.Position),
				Image:       img,
			})
		}
		if len(stickers) == 0 {
			continue
		}
		items = append(items, StickerPack{Object: "sticker_pack", ID: p.ID, DisplayName: name, Stickers: stickers})
	}

	list := repr.NewList(items, nil)
	raw, err := json.Marshal(list)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(raw)
	return &snapshot{list: list, etag: `"` + hex.EncodeToString(sum[:16]) + `"`}, nil
}

// packName is the sticker site's own editor picker naming, so that a sticker
// inserted from the App carries the same alt text as one inserted on the web.
func packName(title map[string]string) string {
	for _, locale := range []string{"zh-cn", "zh-tw", "ja-jp", "en-us", "und"} {
		if v := title[locale]; v != "" {
			return v
		}
	}
	for _, locale := range slices.Sorted(maps.Keys(title)) {
		if v := title[locale]; v != "" {
			return v
		}
	}
	return "Stickers"
}

// matchesAny is If-None-Match's weak comparison, so W/"x" matches "x".
func matchesAny(header, etag string) bool {
	for _, tag := range strings.Split(header, ",") {
		tag = strings.TrimSpace(tag)
		if tag == "*" || strings.TrimPrefix(tag, "W/") == etag {
			return true
		}
	}
	return false
}
