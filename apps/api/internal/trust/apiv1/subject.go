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
)

var ErrSubjectGone = errors.New("trust subject no longer exists")

// Subject is what a resolver reads for one piece of content, hidden or not.
type Subject struct {
	Hidden      bool
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

func (s *Service) subject(ctx context.Context, kind, rawID string) (*ReviewSubject, *ReviewAuthor) {
	resolve, ok := s.subjects[kind]
	if !ok {
		return nil, nil
	}
	id, err := strconv.Atoi(rawID)
	if err != nil || id < 1 {
		return nil, nil
	}
	out := &ReviewSubject{Object: "review_subject", State: "gone", Content: content.NewDocument(content.Blocks{})}
	sub, err := resolve(ctx, id)
	if errors.Is(err, ErrSubjectGone) {
		return out, nil
	}
	if err != nil {
		slog.Warn("trust v1: review subject not read", "kind", kind, "id", id, "error", err)
		return nil, nil
	}
	out.State = "visible"
	if sub.Hidden {
		out.State = "hidden"
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
	if sub.AuthorID < 1 || s.profiles == nil {
		return out, nil
	}
	profile, active, err := s.profiles.ModerationProfile(ctx, sub.AuthorID)
	if err != nil {
		slog.Warn("trust v1: review subject author not read", "kind", kind, "id", id, "author_id", sub.AuthorID, "error", err)
		return out, nil
	}
	if profile == nil {
		return out, nil
	}
	return out, &ReviewAuthor{Object: "review_author", IsAccountActive: active, Profile: *profile}
}

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
