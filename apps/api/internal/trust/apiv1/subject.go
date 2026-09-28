package apiv1

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	userapiv1 "kun-galgame-api/internal/user/apiv1"
	"kun-galgame-api/pkg/trustclient"
)

var ErrSubjectGone = errors.New("trust subject no longer exists")

// Subject is what a resolver reads for one piece of content, hidden or not.
type Subject struct {
	Hidden      bool
	Deleted     bool
	Title       string
	Markdown    string
	Path        string
	ParentTitle string
	ParentPath  string
	AuthorID    int
	CreatedAt   time.Time
}

type SubjectResolver func(ctx context.Context, id int) (Subject, error)

type Subjects map[string]SubjectResolver

type Profiles interface {
	ModerationProfile(ctx context.Context, userID int) (*userapiv1.UserProfile, bool, error)
}

type Converter interface {
	Convert(ctx context.Context, sources []string) ([]content.ContentDocument, error)
}

func (s *Service) WithSubjects(subjects Subjects, profiles Profiles, convert Converter) *Service {
	if s != nil {
		s.subjects, s.profiles, s.convert = subjects, profiles, convert
	}
	return s
}

func (s *Service) subject(ctx context.Context, kind, rawID string) (*ReviewSubject, int) {
	resolve, ok := s.subjects[kind]
	if !ok {
		return nil, 0
	}
	id, err := strconv.Atoi(rawID)
	if err != nil || id < 1 {
		return nil, 0
	}
	out := &ReviewSubject{Object: "review_subject", State: "gone", Content: content.NewDocument(content.Blocks{})}
	sub, err := resolve(ctx, id)
	if errors.Is(err, ErrSubjectGone) {
		return out, 0
	}
	if err != nil {
		slog.Warn("trust v1: review subject not read", "kind", kind, "id", id, "error", err)
		return nil, 0
	}
	switch {
	case sub.Deleted:
		out.State = "deleted"
	case sub.Hidden:
		out.State = "hidden"
	default:
		out.State = "visible"
	}
	if title := truncate(&sub.Title, 512); title != nil {
		out.Title = *title
	}
	out.PagePath = nonEmpty(sub.Path)
	out.ParentTitle = truncate(nonEmpty(sub.ParentTitle), 512)
	out.ParentPath = nonEmpty(sub.ParentPath)
	if !sub.CreatedAt.IsZero() {
		t := repr.Timestamp(sub.CreatedAt)
		out.AuthoredAt = &t
	}
	if sub.Markdown != "" && s.convert != nil {
		docs, err := s.convert.Convert(ctx, []string{sub.Markdown})
		if err != nil {
			slog.Warn("trust v1: review subject body not converted", "kind", kind, "id", id, "error", err)
		} else if len(docs) == 1 {
			out.Content = docs[0]
		}
	}
	return out, max(sub.AuthorID, 0)
}

func (s *Service) reportedAuthor(ctx context.Context, kind, rawID string) int {
	resolve, ok := s.subjects[kind]
	if !ok {
		return 0
	}
	id, err := strconv.Atoi(rawID)
	if err != nil || id < 1 {
		return 0
	}
	sub, err := resolve(ctx, id)
	if err != nil {
		if !errors.Is(err, ErrSubjectGone) {
			slog.Warn("trust v1: reported author not read", "kind", kind, "id", id, "error", err)
		}
		return 0
	}
	return max(sub.AuthorID, 0)
}

func (s *Service) author(ctx context.Context, authorID int, item trustclient.ReviewItem) *ReviewAuthor {
	if authorID < 1 || s.profiles == nil {
		return nil
	}
	profile, active, err := s.profiles.ModerationProfile(ctx, authorID)
	if err != nil {
		slog.Warn("trust v1: review subject author not read", "item", item.ID, "author_id", authorID, "error", err)
		return nil
	}
	if profile == nil {
		return nil
	}
	out := &ReviewAuthor{Object: "review_author", IsAccountActive: active, Profile: *profile}
	out.PastActionedCount = s.pastDecided(ctx, authorID, item, 2)
	out.PastDismissedCount = s.pastDecided(ctx, authorID, item, 3)
	return out
}

// pastDecided counts the author's other items the trust service closed with
// status; the upstream count includes the item under review once it is decided.
func (s *Service) pastDecided(ctx context.Context, authorID int, item trustclient.ReviewItem, status int16) *int {
	page, err := s.trust.ListReviewItems(ctx, accessToken(ctx), trustclient.ReviewQuery{
		Site: s.site, Status: &status, SubjectAuthorID: int64(authorID), Page: 1, Limit: 1,
	})
	if err != nil {
		slog.Warn("trust v1: author history not read", "item", item.ID, "author_id", authorID, "error", err)
		return nil
	}
	n := int(page.Total)
	if item.Status == status && item.SubjectAuthorID != nil && *item.SubjectAuthorID == int64(authorID) {
		n--
	}
	n = max(n, 0)
	return &n
}

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
