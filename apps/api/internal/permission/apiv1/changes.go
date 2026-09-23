package apiv1

import (
	"context"
	"strconv"

	"kun-galgame-api/internal/admin/model"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"
)

type listPermissionChangesInput struct {
	collect.PageNumber
}

type listPermissionChangesOutput struct {
	Body repr.PageList[PermissionChange]
}

func (s *Service) listPermissionChanges(ctx context.Context, in *listPermissionChangesInput) (*listPermissionChangesOutput, error) {
	if _, prob := requireAdmin(ctx); prob != nil {
		return nil, prob
	}
	if s == nil || s.audit == nil || s.people == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	if prob := in.CheckDepth(); prob != nil {
		return nil, prob
	}
	rows, count, err := s.audit.List(ctx, in.Offset(), in.Limit)
	if err != nil {
		return nil, problem.Internal(err)
	}
	ids := make([]int, 0, 2*len(rows))
	for _, row := range rows {
		ids = append(ids, row.OperatorID)
		if uid, ok := targetUserID(row); ok {
			ids = append(ids, uid)
		}
	}
	users, err := s.people.Users(ctx, ids)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	ref := func(id int) repr.UserRef {
		if u, ok := users[id]; ok {
			return repr.NewUserRef(s.cdn, u)
		}
		return repr.DeletedUserRef(id)
	}
	items := make([]PermissionChange, 0, len(rows))
	for _, row := range rows {
		change := PermissionChange{
			Object:    "permission_change",
			ID:        repr.ID(int(row.ID)),
			Actor:     ref(row.OperatorID),
			Before:    auditOverrides(row.BeforeRows),
			After:     auditOverrides(row.AfterRows),
			CreatedAt: repr.Timestamp(row.CreatedAt),
		}
		if uid, ok := targetUserID(row); ok {
			target := ref(uid)
			change.TargetUser = &target
		} else {
			role := row.Subject
			change.TargetRole = &role
		}
		items = append(items, change)
	}
	total, relation := collect.ClampTotal(int(count))
	return &listPermissionChangesOutput{Body: repr.NewPageList(items, total, relation)}, nil
}

func targetUserID(row model.PermissionAuditLog) (int, bool) {
	if row.SubjectKind != "user" {
		return 0, false
	}
	uid, err := strconv.Atoi(row.Subject)
	return uid, err == nil
}

func auditOverrides(deltas []model.AuditDelta) []PermissionOverride {
	out := make([]PermissionOverride, len(deltas))
	for i, d := range deltas {
		out[i] = PermissionOverride{Permission: Permission(d.Permission), Effect: d.Effect}
	}
	return out
}
