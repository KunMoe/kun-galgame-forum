package apiv1

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/pkg/problem"
)

type TopicClassifyRequest struct {
	Title           string `json:"title" minLength:"1" maxLength:"233" doc:"Title as it will be published. Free text; never use it as a decision input."`
	ContentMarkdown string `json:"content_markdown" minLength:"1" maxLength:"100007" doc:"Body as it will be published. Free text; never use it as a decision input."`
}

type TopicClassification struct {
	Object              string          `json:"object" enum:"topic_classification" maxLength:"20" doc:"Type discriminant. Always topic_classification."`
	Section             TopicSectionKey `json:"section" doc:"The section the topic will be published in."`
	Reason              string          `json:"reason" maxLength:"120" doc:"The model's reason, for the author. Free text; never use it as a decision input."`
	ClassificationToken string          `json:"classification_token" maxLength:"256" pattern:"^[A-Za-z0-9_-]+$" doc:"Send as createTopic's classification_token with the same title and body to publish in this section without classifying again."`
	TokenExpiresAt      repr.DateTime   `json:"token_expires_at" doc:"When the token stops being accepted."`
}

type classifyTopicInput struct {
	Body TopicClassifyRequest
}

type classifyTopicOutput struct {
	Body TopicClassification
}

const (
	classifyPerWindow = 10
	classifyWindow    = 10 * time.Minute
)

// classifyLimiter keeps one author from spending the model's per-minute
// quota for everyone else.
type classifyLimiter struct {
	mu   sync.Mutex
	hits map[int][]time.Time
}

func (l *classifyLimiter) allow(userID int, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.hits == nil {
		l.hits = map[int][]time.Time{}
	}
	kept := l.hits[userID][:0]
	for _, t := range l.hits[userID] {
		if now.Sub(t) < classifyWindow {
			kept = append(kept, t)
		}
	}
	if len(kept) >= classifyPerWindow {
		l.hits[userID] = kept
		return false
	}
	l.hits[userID] = append(kept, now)
	return true
}

func classificationUnavailable() *problem.Problem {
	return problem.New(problem.CodeServiceUnavailable, "Automatic section classification is off or the model did not answer. Pick a section by hand.")
}

func (w *Writes) classifyTopic(ctx context.Context, in *classifyTopicInput) (*classifyTopicOutput, error) {
	user := w.caller(ctx)
	if user == nil {
		return nil, problem.New(problem.CodeMissingCredential, "The request has no credentials.")
	}
	if !w.ai.ClassifyEnabled() {
		return nil, classificationUnavailable()
	}
	title := trimTitle(in.Body.Title)
	body := markdown.NormalizeStoredContent(in.Body.ContentMarkdown)
	if title == "" || strings.TrimSpace(body) == "" {
		return nil, validationFailed(tooShort("/title"))
	}
	if !w.classifyLimit.allow(user.ID, time.Now()) {
		return nil, problem.New(problem.CodeRateLimited, "Too many classifications in ten minutes. Pick a section by hand or try again later.")
	}
	verdict, err := w.ai.Classify(ctx, title, body)
	if err != nil {
		slog.Warn("topic classify: model failed", "user_id", user.ID, "error", err)
		return nil, classificationUnavailable()
	}
	expires := time.Now().Add(classificationTTL)
	return &classifyTopicOutput{Body: TopicClassification{
		Object:              "topic_classification",
		Section:             TopicSectionKey(verdict.Section),
		Reason:              verdict.Reason,
		ClassificationToken: w.ai.Token(user.ID, title, body, verdict.Section, expires),
		TokenExpiresAt:      repr.Timestamp(expires),
	}}, nil
}

type sectionDecision struct {
	section string
	source  string
	reason  string
}

// decideSection is createTopic's choice: a token for this exact text, else
// the model now, else the author's pick, recorded as unverified.
func (w *Writes) decideSection(ctx context.Context, userID int, title, body, token, requested string) sectionDecision {
	if !w.ai.ClassifyEnabled() {
		return sectionDecision{section: requested}
	}
	if token != "" {
		if section, err := w.ai.Verify(token, userID, title, body); err == nil {
			return sectionDecision{section: section, source: "model"}
		}
	}
	verdict, err := w.ai.Classify(ctx, title, body)
	if err != nil {
		slog.Warn("topic create: model failed, filing under the author's section", "user_id", userID, "error", err)
		return sectionDecision{section: requested, source: "fallback"}
	}
	return sectionDecision{section: verdict.Section, source: "model", reason: verdict.Reason}
}
