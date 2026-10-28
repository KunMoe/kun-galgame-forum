package apiv1

import (
	"context"
	"log/slog"

	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/pkg/imageclient"
)

const nsfwImageScan = 200

// SetImages lets NSFW judgement read the image service's grading of a
// topic's pictures; without it only the model judges.
func (w *Writes) SetImages(images func(hashes []string) map[string]imageclient.ImageMeta) {
	w.images = images
}

func (w *Writes) hasExplicitImages(body string, covers model.ImageTokens) bool {
	if w.images == nil {
		return false
	}
	seen := map[string]bool{}
	var hashes []string
	for _, tok := range append(markdown.ExtractCoverImages(body, nsfwImageScan), covers...) {
		if h, _, ok := markdown.ParseContentImageRef(tok); ok && !seen[h] {
			seen[h] = true
			hashes = append(hashes, h)
		}
	}
	if len(hashes) == 0 {
		return false
	}
	for _, meta := range w.images(hashes) {
		if meta.IsSexuallyExplicit() {
			return true
		}
	}
	return false
}

// judgeNSFW is the system's verdict on a topic's text and pictures. ok is
// false when the model was asked and did not answer.
func (w *Writes) judgeNSFW(ctx context.Context, title, body string, covers model.ImageTokens) (nsfw, ok bool) {
	if w.hasExplicitImages(body, covers) {
		return true, true
	}
	if !w.ai.ClassifyEnabled() {
		return false, true
	}
	verdict, err := w.ai.Classify(ctx, title, body)
	if err != nil {
		slog.Warn("topic nsfw: model failed", "error", err)
		return false, false
	}
	return verdict.IsNSFW, true
}
