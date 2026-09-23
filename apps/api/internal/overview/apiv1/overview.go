package apiv1

import (
	"context"
	"errors"
	"strconv"
	"time"

	"kun-galgame-api/internal/admin/repository"
	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/infrastructure/cron"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

var errUnconfigured = errors.New("overview v1 is not configured")

type OverviewCounts struct {
	TopicCount           int64 `json:"topic_count" minimum:"0" doc:"Topics, hidden ones included."`
	ReplyCount           int64 `json:"reply_count" minimum:"0" doc:"Topic replies, hidden ones included."`
	TopicCommentCount    int64 `json:"topic_comment_count" minimum:"0" doc:"Comments on topic replies."`
	WorkCount            int64 `json:"work_count" minimum:"0" doc:"Published works: the same predicate as the browse list. Unpublished local rows are not counted."`
	GalgameResourceCount int64 `json:"galgame_resource_count" minimum:"0" doc:"Galgame resources."`
	GalgameCommentCount  int64 `json:"galgame_comment_count" minimum:"0" doc:"Posts on galgame comment walls, as this forum mirrors them; hidden posts are not counted."`
	WebsiteCount         int64 `json:"website_count" minimum:"0" doc:"Website directory entries."`
	WebsiteCommentCount  int64 `json:"website_comment_count" minimum:"0" doc:"Posts on website comment walls, as this forum mirrors them; hidden posts are not counted."`
	DirectMessageCount   int64 `json:"direct_message_count" minimum:"0" doc:"Direct messages."`
}

type AdminOverview struct {
	Object string `json:"object" enum:"admin_overview" maxLength:"14" doc:"Type discriminant. Always admin_overview."`
	OverviewCounts
}

type OverviewDay struct {
	Object     string            `json:"object" enum:"overview_day" maxLength:"12" doc:"Type discriminant. Always overview_day."`
	BucketDate repr.CalendarDate `json:"bucket_date" doc:"The Asia/Shanghai calendar day this bucket counts."`
	OverviewCounts
}

type Service struct {
	repo *repository.OverviewRepository
	now  func() time.Time
}

func New(repo *repository.OverviewRepository, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, now: now}
}

type overviewOutput struct {
	Body AdminOverview
}

type dailyInput struct {
	Days int `query:"days" minimum:"1" maximum:"365" default:"30" doc:"How many Asia/Shanghai calendar days to return, today included. 1–365, default 30."`
}

type dailyOutput struct {
	Body repr.List[OverviewDay]
}

func (s *Service) require(ctx context.Context) *problem.Problem {
	if s == nil || s.repo == nil {
		return problem.Internal(errUnconfigured)
	}
	if !v1.User(ctx).Can(perm.AdminDashboard) {
		return problem.New(problem.CodePermissionRequired, "The caller lacks the admin.dashboard permission.")
	}
	return nil
}

func countsOf(c repository.OverviewCounts) OverviewCounts {
	return OverviewCounts{
		TopicCount:           c.Topic,
		ReplyCount:           c.Reply,
		TopicCommentCount:    c.TopicComment,
		WorkCount:            c.Work,
		GalgameResourceCount: c.GalgameResource,
		GalgameCommentCount:  c.GalgameComment,
		WebsiteCount:         c.Website,
		WebsiteCommentCount:  c.WebsiteComment,
		DirectMessageCount:   c.DirectMessage,
	}
}

func (s *Service) getOverview(ctx context.Context, _ *struct{}) (*overviewOutput, error) {
	if prob := s.require(ctx); prob != nil {
		return nil, prob
	}
	totals, err := s.repo.Totals(ctx)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &overviewOutput{Body: AdminOverview{Object: "admin_overview", OverviewCounts: countsOf(totals)}}, nil
}

func (s *Service) listOverviewDays(ctx context.Context, in *dailyInput) (*dailyOutput, error) {
	if prob := s.require(ctx); prob != nil {
		return nil, prob
	}
	loc := cron.ScheduleLocation()
	today := s.now().In(loc)
	end := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)
	start := end.AddDate(0, 0, -in.Days)
	byDay, err := s.repo.Daily(ctx, cron.ScheduleTZ, start, end)
	if err != nil {
		return nil, problem.Internal(err)
	}
	items := make([]OverviewDay, 0, in.Days)
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		day := d.Format("2006-01-02")
		items = append(items, OverviewDay{
			Object:         "overview_day",
			BucketDate:     repr.CalendarDate(day),
			OverviewCounts: countsOf(byDay[day]),
		})
	}
	return &dailyOutput{Body: repr.NewList(items, nil)}, nil
}

func problemResponses(byStatus map[int]string) map[string]*huma.Response {
	out := make(map[string]*huma.Response, len(byStatus))
	for status, desc := range byStatus {
		out[strconv.Itoa(status)] = &huma.Response{
			Description: desc,
			Content: map[string]*huma.MediaType{
				problem.ContentType: {Schema: &huma.Schema{Ref: v1.ProblemRef}},
			},
		}
	}
	return out
}
