package app

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/internal/community/anchor"
	galgameRepo "kun-galgame-api/internal/galgame/repository"
	quizRepo "kun-galgame-api/internal/quiz/repository"
	toolsetRepo "kun-galgame-api/internal/toolset/repository"
	topicRepo "kun-galgame-api/internal/topic/repository"
	trustapiv1 "kun-galgame-api/internal/trust/apiv1"
	updateRepo "kun-galgame-api/internal/update/repository"
	userapiv1 "kun-galgame-api/internal/user/apiv1"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

func (a *App) newTrustV1(profiles *userapiv1.Users) *trustapiv1.Service {
	if a.TrustV1 == nil || a.UserClient == nil {
		return a.TrustV1
	}
	cdn := ""
	if a.Config != nil {
		cdn = a.Config.NextMoeAPI.ImageCDNBase
	}
	return a.TrustV1.WithSubjects(a.trustSubjects(), profiles, a.contentConverter(cdn))
}

func gone(err error, notFound error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) || (notFound != nil && errors.Is(err, notFound)) {
		return trustapiv1.ErrSubjectGone
	}
	return err
}

func joinMarkdown(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, "\n\n")
}

// trustSubjects reads each kind of content the trust service can put in the
// review queue, hidden rows included: a moderator decides on what readers no
// longer see.
func (a *App) trustSubjects() trustapiv1.Subjects {
	if a.DB == nil {
		return nil
	}
	topics := topicRepo.NewTopicRepository(a.DB)
	replies := topicRepo.NewReplyRepository(a.DB)
	comments := topicRepo.NewCommentRepository(a.DB)
	polls := topicRepo.NewPollRepository(a.DB)
	lotteries := topicRepo.NewLotteryRepository(a.DB)
	todos := updateRepo.NewStore(a.DB)
	ratings := galgameRepo.NewRatingStore(a.DB)
	resources := galgameRepo.NewResourceV1Store(a.DB)
	galgames := galgameRepo.NewGalgameRepository(a.DB)
	commentMap := galgameRepo.NewCommunityPostRepository(a.DB)
	quizzes := quizRepo.NewStore(a.DB)
	toolsets := toolsetRepo.NewStore(a.DB)
	anchors := anchor.New(a.DB, nil)

	topicTitle := func(id int) string {
		if t, err := topics.FindByID(id); err == nil {
			return t.Title
		}
		return ""
	}
	topicPath := func(id int) string { return "/topic/" + strconv.Itoa(id) }

	workName := func(ctx context.Context, id int) string {
		if a.ResourceCatalog == nil {
			return ""
		}
		rows, appErr := a.ResourceCatalog.CatalogRowsByWorkIDs(ctx, []int{id}, "names", "all")
		if appErr != nil {
			return ""
		}
		return rows[id].DisplayName
	}
	workPath := func(id int) string { return "/galgame/" + strconv.Itoa(id) }

	communityPost := func(ctx context.Context, postID int64) (communityclient.AuthorPostView, error) {
		if a.Community == nil || !a.Community.Configured() {
			return communityclient.AuthorPostView{}, errors.New("community is not configured")
		}
		res, err := a.Community.ModerationResolvePosts(ctx, []int64{postID})
		if err != nil {
			return communityclient.AuthorPostView{}, err
		}
		for _, p := range res.Posts {
			// A purged author's posts come back deleted with nothing kept.
			if p.Post.ID == postID && (p.Post.Status < 2 || p.Post.ContentRaw != "") {
				return p, nil
			}
		}
		return communityclient.AuthorPostView{}, trustapiv1.ErrSubjectGone
	}
	postCreated := func(p communityclient.PostView) time.Time {
		t, _ := time.Parse(time.RFC3339, p.CreatedAt)
		return t
	}

	return trustapiv1.Subjects{
		"forum_topic": func(_ context.Context, id int) (trustapiv1.Subject, error) {
			t, err := topics.FindByID(id)
			if err != nil {
				return trustapiv1.Subject{}, gone(err, nil)
			}
			return trustapiv1.Subject{
				Hidden: t.Status != 0, Title: t.Title, Markdown: t.Content, Path: topicPath(t.ID),
				AuthorID: t.UserID, CreatedAt: t.CreatedAt,
			}, nil
		},
		"forum_reply": func(_ context.Context, id int) (trustapiv1.Subject, error) {
			r, err := replies.FindByID(id)
			if err != nil {
				return trustapiv1.Subject{}, gone(err, nil)
			}
			return trustapiv1.Subject{
				Hidden: r.Status != 0, Title: fmt.Sprintf("%d 楼", r.Floor), Markdown: r.Content,
				Path:        fmt.Sprintf("/topic/%d?reply=%d", r.TopicID, r.Floor),
				ParentTitle: topicTitle(r.TopicID), ParentPath: topicPath(r.TopicID),
				AuthorID: r.UserID, CreatedAt: r.CreatedAt,
			}, nil
		},
		"forum_comment": func(_ context.Context, id int) (trustapiv1.Subject, error) {
			c, err := comments.FindCommentByID(id)
			if err != nil {
				return trustapiv1.Subject{}, gone(err, nil)
			}
			return trustapiv1.Subject{
				Hidden: c.Status != 0, Markdown: c.Content,
				Path:        fmt.Sprintf("/topic/%d?comment=%d", c.TopicID, c.ID),
				ParentTitle: topicTitle(c.TopicID), ParentPath: topicPath(c.TopicID),
				AuthorID: c.UserID, CreatedAt: c.CreatedAt,
			}, nil
		},
		"forum_topic_poll": func(_ context.Context, id int) (trustapiv1.Subject, error) {
			p, err := polls.FindByID(id)
			if err != nil {
				return trustapiv1.Subject{}, gone(err, nil)
			}
			return trustapiv1.Subject{
				Title: p.Title, Markdown: p.Description, Path: topicPath(p.TopicID),
				ParentTitle: topicTitle(p.TopicID), ParentPath: topicPath(p.TopicID),
				AuthorID: p.UserID, CreatedAt: p.CreatedAt,
			}, nil
		},
		"forum_topic_lottery": func(_ context.Context, id int) (trustapiv1.Subject, error) {
			l, err := lotteries.FindByID(id)
			if err != nil {
				return trustapiv1.Subject{}, gone(err, nil)
			}
			return trustapiv1.Subject{
				Title: l.Title, Markdown: l.Description, Path: topicPath(l.TopicID),
				ParentTitle: topicTitle(l.TopicID), ParentPath: topicPath(l.TopicID),
				AuthorID: l.UserID, CreatedAt: l.CreatedAt,
			}, nil
		},
		"forum_todo": func(_ context.Context, id int) (trustapiv1.Subject, error) {
			t, err := todos.FindTodo(id)
			if err != nil {
				return trustapiv1.Subject{}, gone(err, updateRepo.ErrNotFound)
			}
			return trustapiv1.Subject{Markdown: t.Content, Path: "/update/todo", AuthorID: t.UserID, CreatedAt: t.CreatedAt}, nil
		},
		"galgame": func(ctx context.Context, id int) (trustapiv1.Subject, error) {
			name := workName(ctx, id)
			local := galgames.FindLocal(id)
			if name == "" && local.ID == 0 {
				return trustapiv1.Subject{}, trustapiv1.ErrSubjectGone
			}
			sub := trustapiv1.Subject{Title: name, Path: workPath(id), CreatedAt: local.CreatedAt}
			if local.CreatorUserID != nil {
				sub.AuthorID = *local.CreatorUserID
			}
			return sub, nil
		},
		"galgame_rating": func(ctx context.Context, id int) (trustapiv1.Subject, error) {
			r, err := ratings.Get(id)
			if err != nil {
				return trustapiv1.Subject{}, gone(err, galgameRepo.ErrRatingNotFound)
			}
			return trustapiv1.Subject{
				Markdown: r.ShortSummary, Path: "/galgame-rating/" + strconv.Itoa(r.ID),
				ParentTitle: workName(ctx, r.WorkID), ParentPath: workPath(r.WorkID),
				AuthorID: r.UserID, CreatedAt: r.Created,
			}, nil
		},
		"galgame_resource": func(ctx context.Context, id int) (trustapiv1.Subject, error) {
			r, err := resources.Find(id)
			if err != nil {
				return trustapiv1.Subject{}, gone(err, galgameRepo.ErrResourceNotFound)
			}
			links, err := resources.Links(r.ID)
			if err != nil {
				return trustapiv1.Subject{}, err
			}
			list := make([]string, len(links))
			for i, l := range links {
				list[i] = "- " + l
			}
			return trustapiv1.Subject{
				Title: r.Title, Markdown: joinMarkdown(r.Note, strings.Join(list, "\n")),
				Path:        "/galgame/resource/" + strconv.Itoa(r.ID),
				ParentTitle: workName(ctx, r.WorkID), ParentPath: workPath(r.WorkID),
				AuthorID: r.UserID, CreatedAt: r.CreatedAt,
			}, nil
		},
		"galgame_quiz": func(_ context.Context, id int) (trustapiv1.Subject, error) {
			q, err := quizzes.Find(id)
			if err != nil {
				return trustapiv1.Subject{}, gone(err, quizRepo.ErrNotFound)
			}
			return trustapiv1.Subject{
				Title: q.Question, Markdown: joinMarkdown(q.Description, q.Explanation),
				Path: "/galgame-quiz/" + strconv.Itoa(q.ID), AuthorID: q.UserID, CreatedAt: q.CreatedAt,
			}, nil
		},
		"galgame_toolset": func(_ context.Context, id int) (trustapiv1.Subject, error) {
			t, err := toolsets.Find(id)
			if err != nil {
				return trustapiv1.Subject{}, gone(err, toolsetRepo.ErrNotFound)
			}
			return trustapiv1.Subject{
				Title: t.Name, Markdown: t.Description, Path: "/toolset/" + strconv.Itoa(t.ID),
				AuthorID: t.UserID, CreatedAt: t.CreatedAt,
			}, nil
		},
		"galgame_toolset_resource": func(_ context.Context, id int) (trustapiv1.Subject, error) {
			r, err := toolsets.FindResource(id)
			if err != nil {
				return trustapiv1.Subject{}, gone(err, toolsetRepo.ErrNotFound)
			}
			parent := ""
			if t, err := toolsets.Find(r.ToolsetID); err == nil {
				parent = t.Name
			}
			path := "/toolset/" + strconv.Itoa(r.ToolsetID)
			return trustapiv1.Subject{
				Markdown: joinMarkdown(r.Content, r.Note), Path: path, ParentTitle: parent, ParentPath: path,
				AuthorID: r.UserID, CreatedAt: r.CreatedAt,
			}, nil
		},
		"galgame_comment": func(ctx context.Context, id int) (trustapiv1.Subject, error) {
			m, err := commentMap.FindMapByLegacyID(id)
			if err != nil {
				return trustapiv1.Subject{}, gone(err, nil)
			}
			p, err := communityPost(ctx, m.PostID)
			if err != nil {
				return trustapiv1.Subject{}, err
			}
			return trustapiv1.Subject{
				Hidden: p.Post.Status == 1, Deleted: p.Post.Status >= 2, Markdown: p.Post.ContentRaw,
				Path:        fmt.Sprintf("/galgame/%d?comment=%d", m.WorkID, m.PostID),
				ParentTitle: workName(ctx, m.WorkID), ParentPath: workPath(m.WorkID),
				AuthorID: int(p.Post.AuthorID), CreatedAt: postCreated(p.Post),
			}, nil
		},
		"community_post": func(ctx context.Context, id int) (trustapiv1.Subject, error) {
			p, err := communityPost(ctx, int64(id))
			if err != nil {
				return trustapiv1.Subject{}, err
			}
			sub := trustapiv1.Subject{
				Hidden: p.Post.Status == 1, Deleted: p.Post.Status >= 2, Markdown: p.Post.ContentRaw, ParentTitle: p.Thread.Title,
				AuthorID: int(p.Post.AuthorID), CreatedAt: postCreated(p.Post),
			}
			ref := anchor.Ref{Kind: p.Thread.AnchorKind, ID: p.Thread.AnchorID}
			if target, ok := anchors.Resolve([]anchor.Ref{ref})[ref]; ok {
				sub.ParentPath = target.Link
				sub.Path = target.Link
				if target.WorkID > 0 {
					sub.ParentTitle = workName(ctx, target.WorkID)
					sub.Path = fmt.Sprintf("%s?comment=%d", target.Link, id)
				}
				if sub.ParentTitle == "" {
					sub.ParentTitle = target.Label
				}
			}
			return sub, nil
		},
		"user": func(ctx context.Context, id int) (trustapiv1.Subject, error) {
			u, ok, err := a.UserClient.User(ctx, id)
			if err != nil {
				return trustapiv1.Subject{}, err
			}
			if !ok {
				return trustapiv1.Subject{}, trustapiv1.ErrSubjectGone
			}
			return trustapiv1.Subject{
				Hidden: !userclient.IsRenderable(u), Title: u.Name, Path: "/user/" + strconv.Itoa(id), AuthorID: id,
			}, nil
		},
	}
}
