package apiv1

import (
	"context"
	"strconv"
	"strings"
	"time"

	adminModel "kun-galgame-api/internal/admin/model"
	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/update/repository"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
)

const listSort = "created_at_desc"

func parsePos(keys []string) (*repository.Pos, *problem.Problem) {
	if keys == nil {
		return nil, nil
	}
	if len(keys) != 2 {
		return nil, invalidCursor()
	}
	created, err := time.Parse(time.RFC3339Nano, keys[0])
	if err != nil {
		return nil, invalidCursor()
	}
	id, err := strconv.Atoi(keys[1])
	if err != nil || id <= 0 {
		return nil, invalidCursor()
	}
	return &repository.Pos{Created: created, ID: id}, nil
}

func nextCursor(fp string, created time.Time, id int) *string {
	cur := collect.EncodeCursor(listSort, fp, created.UTC().Format(time.RFC3339Nano), strconv.Itoa(id))
	return &cur
}

func limitOf(n int) int {
	if n <= 0 {
		return collect.DefaultLimit
	}
	return n
}

func (s *Service) listUpdateLogs(ctx context.Context, in *listUpdateLogsInput) (*listUpdateLogsOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	const fp = "update_logs"
	keys, p := collect.DecodeCursor(in.Cursor, listSort, fp)
	if p != nil {
		return nil, p
	}
	pos, p := parsePos(keys)
	if p != nil {
		return nil, p
	}
	limit := limitOf(in.Limit)
	rows, err := s.store.ListUpdateLogs(pos, limit+1)
	if err != nil {
		return nil, problem.Internal(err)
	}
	var next *string
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[limit-1]
		next = nextCursor(fp, last.CreatedAt, last.ID)
	}
	viewer := v1.User(ctx)
	items := make([]UpdateLog, 0, len(rows))
	for i := range rows {
		items = append(items, renderUpdateLog(&rows[i], viewer))
	}
	return &listUpdateLogsOutput{Body: repr.NewList(items, next)}, nil
}

func (s *Service) getUpdateLog(ctx context.Context, in *updateLogIDInput) (*updateLogOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	id, ok := repr.ParseID(repr.DecimalID(in.UpdateLogID))
	if !ok {
		return nil, notFound()
	}
	row, err := s.store.FindUpdateLog(id)
	if err != nil {
		return nil, storeProblem(err)
	}
	return &updateLogOutput{Body: renderUpdateLog(row, v1.User(ctx))}, nil
}

func (s *Service) staff(ctx context.Context, p perm.Permission, detail string) (*middleware.UserInfo, *problem.Problem) {
	user, prob := s.requireActive(ctx)
	if prob != nil {
		return nil, prob
	}
	if !user.Can(p) {
		return nil, permissionRequired(detail)
	}
	return user, nil
}

func (s *Service) createUpdateLog(ctx context.Context, in *createUpdateLogInput) (*createUpdateLogOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.staff(ctx, perm.UpdateLogCreate, "Creating an update log needs update_log.create.")
	if p != nil {
		return nil, p
	}
	version := strings.TrimSpace(in.Body.ReleaseVersion)
	if version == "" {
		return nil, tooShort("/release_version")
	}
	text := markdown.NormalizeStoredContent(in.Body.Text)
	if blank(text) {
		return nil, tooShort("/text")
	}
	row := adminModel.UpdateLog{
		Type:    in.Body.ChangeType,
		Version: version,
		Content: text,
		UserID:  user.ID,
	}
	if err := s.store.CreateUpdateLog(&row); err != nil {
		return nil, problem.Internal(err)
	}
	return &createUpdateLogOutput{
		Location: v1.Prefix + "/update-logs/" + strconv.Itoa(row.ID),
		Body:     renderUpdateLog(&row, user),
	}, nil
}

func (s *Service) updateUpdateLog(ctx context.Context, in *patchUpdateLogInput) (*updateLogOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.staff(ctx, perm.UpdateLogEdit, "Editing an update log needs update_log.edit.")
	if p != nil {
		return nil, p
	}
	id, ok := repr.ParseID(repr.DecimalID(in.UpdateLogID))
	if !ok {
		return nil, notFound()
	}
	fields := map[string]any{}
	if in.Body.ChangeType != nil {
		fields["type"] = *in.Body.ChangeType
	}
	if in.Body.ReleaseVersion != nil {
		version := strings.TrimSpace(*in.Body.ReleaseVersion)
		if version == "" {
			return nil, tooShort("/release_version")
		}
		fields["version"] = version
	}
	if in.Body.Text != nil {
		text := markdown.NormalizeStoredContent(*in.Body.Text)
		if blank(text) {
			return nil, tooShort("/text")
		}
		fields["content"] = text
	}
	if len(fields) > 0 {
		if err := s.store.PatchUpdateLog(id, fields); err != nil {
			return nil, storeProblem(err)
		}
	}
	row, err := s.store.FindUpdateLog(id)
	if err != nil {
		return nil, storeProblem(err)
	}
	return &updateLogOutput{Body: renderUpdateLog(row, user)}, nil
}

func (s *Service) deleteUpdateLog(ctx context.Context, in *updateLogIDInput) (*noContentOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	if _, p := s.staff(ctx, perm.UpdateLogDelete, "Deleting an update log needs update_log.delete."); p != nil {
		return nil, p
	}
	id, ok := repr.ParseID(repr.DecimalID(in.UpdateLogID))
	if !ok {
		return nil, notFound()
	}
	if err := s.store.DeleteUpdateLog(id); err != nil {
		return nil, storeProblem(err)
	}
	return &noContentOutput{}, nil
}
