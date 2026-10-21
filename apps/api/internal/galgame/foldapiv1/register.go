package foldapiv1

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

var errUnconfigured = errors.New("work merge fold v1 is not configured")

type Service struct {
	repo *repository.GalgameMergeRepository
}

func New(repo *repository.GalgameMergeRepository) *Service {
	return &Service{repo: repo}
}

type WorkMergeFold struct {
	Object         string                 `json:"object" enum:"work_merge_fold" maxLength:"15" doc:"Type discriminant. Always work_merge_fold."`
	ID             repr.DecimalID         `json:"id" doc:"Fold id."`
	OldWorkID      repr.DecimalID         `json:"old_work_id" doc:"The work catalog merged away, whose forum rows were folded."`
	NewWorkID      repr.DecimalID         `json:"new_work_id" doc:"The survivor the rows were folded onto."`
	FoldedAt       repr.DateTime          `json:"folded_at" doc:"When the forum folded the rows."`
	UndoneAt       *repr.DateTime         `json:"undone_at" doc:"When the fold was reversed because catalog undid or re-pointed the merge. null while it stands."`
	MovedCount     int                    `json:"moved_count" minimum:"0" doc:"Rows moved onto the survivor."`
	DroppedCount   int                    `json:"dropped_count" minimum:"0" doc:"Rows dropped because the survivor already held the same user's or quiz's row; kept for the reversal."`
	SummedCount    int                    `json:"summed_count" minimum:"0" doc:"Contributor rows and daily view buckets added into the survivor's."`
	RepointedCount int                    `json:"repointed_count" minimum:"0" doc:"Feed entries pointed at the survivor."`
	Problems       []WorkMergeFoldProblem `json:"problems" maxItems:"10000" doc:"Rows the reversal could not put back. Empty array, never null, and empty while the fold stands."`
}

type WorkMergeFoldProblem struct {
	TableName     string          `json:"table_name" enum:"galgame,galgame_resource,galgame_activity,galgame_comment_community_map,galgame_resource_relocation,galgame_like,galgame_favorite,galgame_contributor,galgame_quiz_galgame,galgame_view_daily,feed_activity" maxLength:"29" doc:"The table the row is in."`
	RowID         repr.DecimalID  `json:"row_id" doc:"The row's key in that table: its id, or quiz_id, old_comment_id or resource_id."`
	Cause         string          `json:"cause" enum:"gone,elsewhere,conflict,purged_user,survivor_folded" maxLength:"15" doc:"gone: deleted since the fold. elsewhere: now on another work (current_work_id). conflict: the old work already holds that row. purged_user: its author was purged. survivor_folded: the survivor's own row was folded further, so its counters keep the old work's views."`
	CurrentWorkID *repr.DecimalID `json:"current_work_id" doc:"Where the row is now, for elsewhere and conflict. null otherwise."`
}

type listWorkMergeFoldsInput struct {
	collect.PageNumber
}

type listWorkMergeFoldsOutput struct {
	Body repr.PageList[WorkMergeFold]
}

func Register(s *Service) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "listWorkMergeFolds",
			Method:      http.MethodGet,
			Path:        "/admin/work-merge-folds",
			Summary:     "List work merge folds",
			Description: "Every time the forum folded a merged work's rows onto its survivor, newest first, with what an undone merge could not put back. " +
				"A page-number collection: page × limit may not exceed 10000. Admins only.",
			Tags: []string{"admin"},
			Responses: map[string]*huma.Response{
				"403": {
					Description: "PERMISSION_REQUIRED when the caller is not an admin.",
					Content:     map[string]*huma.MediaType{problem.ContentType: {Schema: &huma.Schema{Ref: v1.ProblemRef}}},
				},
			},
		}), s.listWorkMergeFolds)
	}
}

func (s *Service) listWorkMergeFolds(ctx context.Context, in *listWorkMergeFoldsInput) (*listWorkMergeFoldsOutput, error) {
	if u := v1.User(ctx); u == nil || !u.CanAdminister() {
		return nil, problem.New(problem.CodePermissionRequired, "The caller cannot perform this operation.")
	}
	if s == nil || s.repo == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	if prob := in.CheckDepth(); prob != nil {
		return nil, prob
	}
	rows, count, err := s.repo.ListFolds(in.Offset(), in.Limit)
	if err != nil {
		return nil, problem.Internal(err)
	}
	items := make([]WorkMergeFold, 0, len(rows))
	for _, row := range rows {
		items = append(items, foldOf(row))
	}
	total, relation := collect.ClampTotal(count)
	return &listWorkMergeFoldsOutput{Body: repr.NewPageList(items, total, relation)}, nil
}

func foldOf(row repository.FoldSummary) WorkMergeFold {
	out := WorkMergeFold{
		Object: "work_merge_fold", ID: decimal(row.ID),
		OldWorkID: repr.ID(row.OldWorkID), NewWorkID: repr.ID(row.NewWorkID),
		FoldedAt:   repr.Timestamp(row.FoldedAt),
		MovedCount: row.Moved, DroppedCount: row.Dropped, SummedCount: row.Summed, RepointedCount: row.Repointed,
		Problems: make([]WorkMergeFoldProblem, 0, len(row.Problems)),
	}
	if row.UndoneAt != nil {
		at := repr.Timestamp(*row.UndoneAt)
		out.UndoneAt = &at
	}
	for _, p := range row.Problems {
		item := WorkMergeFoldProblem{TableName: p.Table, RowID: decimal(p.Key), Cause: p.Reason}
		if p.WorkID != nil {
			id := repr.ID(*p.WorkID)
			item.CurrentWorkID = &id
		}
		out.Problems = append(out.Problems, item)
	}
	return out
}

func decimal(n int64) repr.DecimalID {
	return repr.DecimalID(strconv.FormatInt(n, 10))
}
