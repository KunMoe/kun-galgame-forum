package apiv1

import (
	"context"
	"log/slog"
	"time"

	"kun-galgame-api/internal/setting"
	"kun-galgame-api/internal/topic/ai"
	"kun-galgame-api/internal/topic/repository"

	"gorm.io/gorm"
)

const (
	sweepBatch         = 10
	sweepSummaryWindow = 7 * 24 * time.Hour
)

// Sweep retries what publishing could not finish: topics filed under the
// author's pick because the model did not answer, and summaries that are
// missing or out of date. A disagreeing verdict becomes a staff request and
// never moves a topic or its author's moemoepoint by itself.
func (a *TopicAI) Sweep() {
	if a == nil || a.client == nil {
		return
	}
	ctx := context.Background()
	if a.settings.Bool(setting.TopicAIClassifyEnabled) {
		a.recheck(ctx)
	}
	if a.SummaryEnabled() {
		a.backfillSummaries(ctx)
	}
}

func (a *TopicAI) recheck(ctx context.Context) {
	topics, err := a.repo.UnverifiedTopics(time.Now().Add(-time.Minute), sweepBatch)
	if err != nil {
		slog.Warn("topic recheck: read failed", "error", err)
		return
	}
	for _, t := range topics {
		verdict, err := a.Classify(ctx, t.Title, t.Content)
		if err != nil {
			slog.Warn("topic recheck: model failed", "topic_id", t.ID, "error", err)
			return
		}
		err = a.repo.DB().Transaction(func(tx *gorm.DB) error {
			if err := a.repo.SaveClassification(tx, repository.TopicClassification{
				TopicID: t.ID, Section: verdict.Section, Source: "model", Model: a.model(), Reason: verdict.Reason,
			}); err != nil {
				return err
			}
			if verdict.Section == t.Section {
				return nil
			}
			pending, err := a.repo.HasPendingRequest(tx, t.ID)
			if err != nil || pending {
				return err
			}
			return a.repo.CreateRequest(tx, &repository.TopicSectionRequest{
				TopicID: t.ID, FromSection: t.Section, ToSection: verdict.Section,
				Reason: ai.RecheckReason(verdict.Reason),
			})
		})
		if err != nil {
			slog.Warn("topic recheck: write failed", "topic_id", t.ID, "error", err)
		}
	}
}

func (a *TopicAI) backfillSummaries(ctx context.Context) {
	minLen := a.settings.Int(setting.TopicSummaryMinLength)
	topics, err := a.repo.TopicsNeedingSummary(time.Now().Add(-sweepSummaryWindow), minLen,
		a.settings.Int(setting.TopicSummaryRegenerateThreshold), sweepBatch)
	if err != nil {
		slog.Warn("topic summary sweep: read failed", "error", err)
		return
	}
	for _, t := range topics {
		if !a.wantsSummary(t.Content) {
			if err := a.repo.SetSummary(t.ID, nil, a.model()); err != nil {
				slog.Warn("topic summary sweep: write failed", "topic_id", t.ID, "error", err)
			}
			continue
		}
		cctx, cancel := context.WithTimeout(ctx, summaryTimeout)
		err := a.summarizeNow(cctx, t.ID, t.Title, t.Content)
		cancel()
		if err != nil {
			slog.Warn("topic summary sweep: model failed", "topic_id", t.ID, "error", err)
			return
		}
	}
}
