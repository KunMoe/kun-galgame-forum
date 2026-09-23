package apiv1

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/section/repository"
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"github.com/danielgtaylor/huma/v2"
)

const latestCandidates = 10

var errUnconfigured = errors.New("apiv1 sections: service is not configured")

var categoryPrefix = map[string]string{"galgame": "g-", "technique": "t-", "others": "o-"}

type SectionLatestTopic struct {
	Object    string         `json:"object" enum:"topic" maxLength:"5" doc:"Type discriminant. Always topic."`
	ID        repr.DecimalID `json:"id" doc:"Topic id. JSON string of a decimal integer."`
	Title     string         `json:"title" maxLength:"233" doc:"Topic title. Free text; never use it as a decision input."`
	CreatedAt repr.DateTime  `json:"created_at" doc:"Creation time."`
}

type Section struct {
	Object      string                 `json:"object" enum:"section" maxLength:"7" doc:"Type discriminant. Always section."`
	Section     topicapiv1.SectionSlug `json:"section" doc:"The section, as filed on a topic's sections and as the /section/{section} page segment."`
	Category    string                 `json:"category" enum:"galgame,technique,others" maxLength:"9" doc:"The topic category the section belongs to."`
	TopicCount  int64                  `json:"topic_count" minimum:"0" doc:"Published topics filed under the section that anyone may open, NSFW ones included. A size statistic, not the total of any list."`
	ViewCount   int64                  `json:"view_count" minimum:"0" doc:"Views summed over the same topics as topic_count."`
	LatestTopic *SectionLatestTopic    `json:"latest_topic" doc:"The newest of those topics whose author is still shown. null when there is none."`
}

type listSectionsInput struct {
	Category string `query:"category" enum:"galgame,technique,others" maxLength:"9" doc:"When set, only this category's sections. Omitted means all of them."`
}

type listSectionsOutput struct {
	Body repr.List[Section]
}

type Service struct {
	repo  *repository.SectionRepository
	users *userclient.Client
}

func New(repo *repository.SectionRepository, users *userclient.Client) *Service {
	return &Service{repo: repo, users: users}
}

func categoryOf(slug string) (string, error) {
	for category, prefix := range categoryPrefix {
		if strings.HasPrefix(slug, prefix) {
			return category, nil
		}
	}
	return "", fmt.Errorf("section %q has no known category prefix", slug)
}

func (s *Service) listSections(ctx context.Context, in *listSectionsInput) (*listSectionsOutput, error) {
	if s == nil || s.repo == nil || s.users == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	rows, err := s.repo.Stats(categoryPrefix[in.Category])
	if err != nil {
		return nil, problem.Internal(err)
	}
	candidates := make(map[int][]repository.LatestTopicRow, len(rows))
	var uids []int
	for _, row := range rows {
		cands, err := s.repo.LatestTopics(row.SectionID, latestCandidates)
		if err != nil {
			return nil, problem.Internal(err)
		}
		candidates[row.SectionID] = cands
		for _, c := range cands {
			if !slices.Contains(uids, c.UserID) {
				uids = append(uids, c.UserID)
			}
		}
	}
	users, err := s.users.Users(ctx, uids)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	items := make([]Section, 0, len(rows))
	for _, row := range rows {
		if !topicapiv1.IsSectionSlug(row.SectionName) {
			return nil, problem.Internal(fmt.Errorf("section %q is outside the closed vocabulary", row.SectionName))
		}
		category, err := categoryOf(row.SectionName)
		if err != nil {
			return nil, problem.Internal(err)
		}
		item := Section{
			Object:     "section",
			Section:    topicapiv1.SectionSlug(row.SectionName),
			Category:   category,
			TopicCount: row.TopicCount,
			ViewCount:  row.ViewCount,
		}
		for _, c := range candidates[row.SectionID] {
			if u, ok := users[c.UserID]; ok && userclient.IsRenderable(u) {
				item.LatestTopic = &SectionLatestTopic{
					Object:    "topic",
					ID:        repr.ID(c.ID),
					Title:     c.Title,
					CreatedAt: repr.Timestamp(c.Created),
				}
				break
			}
		}
		items = append(items, item)
	}
	return &listSectionsOutput{Body: repr.NewList(items, nil)}, nil
}

func Register(s *Service) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listSections",
			Method:      http.MethodGet,
			Path:        "/sections",
			Summary:     "List topic sections",
			Description: "Every topic section in vocabulary order with its size and newest topic, empty sections included. " +
				"Not paged: the vocabulary is closed and small. The topics of one section are listTopics with section set.",
			Tags: []string{"topics"},
			Responses: map[string]*huma.Response{
				"503": {
					Description: "SERVICE_UNAVAILABLE when the account service cannot tell which newest topics have a shown author.",
					Content: map[string]*huma.MediaType{
						problem.ContentType: {Schema: &huma.Schema{Ref: v1.ProblemRef}},
					},
				},
			},
		}), s.listSections)
	}
}
