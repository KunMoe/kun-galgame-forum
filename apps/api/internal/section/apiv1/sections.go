package apiv1

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"sync"
	"time"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/section/repository"
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"github.com/danielgtaylor/huma/v2"
)

const (
	latestCandidates = 10
	listTTL          = time.Minute
)

var errUnconfigured = errors.New("apiv1 sections: service is not configured")

type SectionLatestTopic struct {
	Object    string         `json:"object" enum:"topic" maxLength:"5" doc:"Type discriminant. Always topic."`
	ID        repr.DecimalID `json:"id" doc:"Topic id. JSON string of a decimal integer."`
	Title     string         `json:"title" maxLength:"233" doc:"Topic title. Free text; never use it as a decision input."`
	CreatedAt repr.DateTime  `json:"created_at" doc:"Creation time."`
}

type Section struct {
	Object              string                     `json:"object" enum:"section" maxLength:"7" doc:"Type discriminant. Always section."`
	Section             topicapiv1.TopicSectionKey `json:"section" doc:"The section, as filed on a topic's section and as the /section/{section} page segment."`
	Group               SectionGroupKey            `json:"group" doc:"The group the section is shown under."`
	GroupLabel          string                     `json:"group_label" maxLength:"40" doc:"The group's display name, edited by admins. Free text; never use it as a decision input."`
	Label               string                     `json:"label" maxLength:"40" doc:"Display name, edited by admins. Free text; never use it as a decision input."`
	Description         string                     `json:"description" maxLength:"500" doc:"What belongs in the section, edited by admins. Free text; never use it as a decision input."`
	IconName            SectionIconName            `json:"icon_name" doc:"Icon to show beside the label."`
	Color               SectionColor               `json:"color" doc:"Color to show the section's badge in."`
	Category            string                     `json:"category" enum:"galgame,technique,others" maxLength:"9" doc:"Deprecated: read group instead. The legacy category topics in this section still carry."`
	IsArchived          bool                       `json:"is_archived" doc:"An archived section keeps its topics and takes no new ones."`
	MoemoepointOnCreate int                        `json:"moemoepoint_on_create" minimum:"-10" maximum:"3" doc:"What publishing a topic here does to the author's moemoepoint: negative is a cost the balance must cover, positive a reward."`
	HiddenFromFeeds     bool                       `json:"is_hidden_from_feeds" doc:"Whether its topics are left out of the home activity feed's normal tab."`
	TopicCount          int64                      `json:"topic_count" minimum:"0" doc:"Published topics filed under the section that anyone may open, NSFW ones included. A size statistic, not the total of any list."`
	ViewCount           int64                      `json:"view_count" minimum:"0" doc:"Views summed over the same topics as topic_count."`
	LatestTopic         *SectionLatestTopic        `json:"latest_topic" doc:"The newest of those topics whose author is still shown. null when there is none."`
}

type listSectionsInput struct {
	Category string `query:"category" enum:"galgame,technique,others" maxLength:"9" doc:"Deprecated. When set, only the sections whose legacy category is this one."`
}

type listSectionsOutput struct {
	Body repr.List[Section]
}

type Service struct {
	repo  *repository.SectionRepository
	users *userclient.Client

	mu       sync.Mutex
	cached   []Section
	cachedAt time.Time
}

func New(repo *repository.SectionRepository, users *userclient.Client) *Service {
	return &Service{repo: repo, users: users}
}

func moemoepointOnCreate(costs bool) int {
	if costs {
		return -constants.CostConsumeSection
	}
	return constants.RewardCreateTopic
}

func (s *Service) listSections(ctx context.Context, in *listSectionsInput) (*listSectionsOutput, error) {
	if s == nil || s.repo == nil || s.users == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	items, p := s.allSections(ctx)
	if p != nil {
		return nil, p
	}
	if in.Category != "" {
		items = slices.DeleteFunc(slices.Clone(items), func(it Section) bool { return it.Category != in.Category })
	}
	return &listSectionsOutput{Body: repr.NewList(items, nil)}, nil
}

func (s *Service) invalidate() {
	s.mu.Lock()
	s.cached = nil
	s.mu.Unlock()
}

// allSections is read by every server-rendered page, so it is kept for a
// minute; an admin write drops it.
func (s *Service) allSections(ctx context.Context) ([]Section, *problem.Problem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cached != nil && time.Since(s.cachedAt) < listTTL {
		return s.cached, nil
	}
	rows, err := s.repo.List("")
	if err != nil {
		return nil, problem.Internal(err)
	}
	candidates := make(map[int][]repository.LatestTopicRow, len(rows))
	var uids []int
	for _, row := range rows {
		cands, err := s.repo.LatestTopics(row.ID, latestCandidates)
		if err != nil {
			return nil, problem.Internal(err)
		}
		candidates[row.ID] = cands
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
		item := Section{
			Object:              "section",
			Section:             topicapiv1.TopicSectionKey(row.Name),
			Group:               SectionGroupKey(row.GroupKey),
			GroupLabel:          row.GroupLabel,
			Label:               row.Label,
			Description:         row.Description,
			IconName:            SectionIconName(row.Icon),
			Color:               SectionColor(row.Color),
			Category:            row.LegacyCategory,
			IsArchived:          row.IsArchived,
			MoemoepointOnCreate: moemoepointOnCreate(row.CostsMoemoepoint),
			HiddenFromFeeds:     row.HiddenFromFeeds,
			TopicCount:          row.TopicCount,
			ViewCount:           row.ViewCount,
		}
		for _, c := range candidates[row.ID] {
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
	s.cached, s.cachedAt = items, time.Now()
	return items, nil
}

func Register(s *Service) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listSections",
			Method:      http.MethodGet,
			Path:        "/sections",
			Summary:     "List topic sections",
			Description: "Every topic section in display order (by group, then within the group), archived ones included, with its display text, rules, size and newest topic. " +
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
		registerAdmin(api, s)
	}
}
