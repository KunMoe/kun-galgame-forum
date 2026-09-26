package apiv1

import (
	"context"
	"fmt"
	"strconv"
	"time"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

const moemoepointEntrySort = "id_desc"
const defaultMoemoepointLimit = 20

type listMoemoepointEntriesInput struct {
	Cursor string `query:"cursor" pattern:"^cur_[A-Za-z0-9_-]+$" maxLength:"512" doc:"Opaque keyset cursor from a previous page of this collection."`
	Limit  int    `query:"limit" minimum:"1" maximum:"50" default:"20" doc:"Page size. 1–50, default 20. Values above 50 are rejected, not clamped."`
	Reason string `query:"reason" pattern:"^[a-z][a-z0-9_]{0,39}$" maxLength:"40" doc:"When set, only this ledger reason. Omitted means every reason."`
}

type listMoemoepointEntriesOutput struct {
	Body repr.List[MoemoepointEntry]
}

func moemoepointFingerprint(userID int, reason string) string {
	return collect.Fingerprint(strconv.Itoa(userID), reason)
}

func (s *Users) listMoemoepointEntries(ctx context.Context, in *listMoemoepointEntriesInput) (*listMoemoepointEntriesOutput, error) {
	if prob := s.readyAccounts(); prob != nil {
		return nil, prob
	}
	user := v1.User(ctx)
	if user == nil {
		return nil, problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	}
	fp := moemoepointFingerprint(user.ID, in.Reason)
	keys, curErr := collect.DecodeCursor(in.Cursor, moemoepointEntrySort, fp)
	if curErr != nil {
		return nil, curErr
	}
	beforeID := 0
	if keys != nil {
		if len(keys) != 1 {
			return nil, invalidCursor()
		}
		n, err := strconv.Atoi(keys[0])
		if err != nil || n <= 0 {
			return nil, invalidCursor()
		}
		beforeID = n
	}
	limit := in.Limit
	if limit <= 0 {
		limit = defaultMoemoepointLimit
	}

	page, err := s.accounts.MoemoepointLog(ctx, user.ID, limit, beforeID, in.Reason)
	if err != nil {
		return nil, unavailable(err)
	}

	items := make([]MoemoepointEntry, 0, len(page.Items))
	for _, row := range page.Items {
		entry, err := mapMoemoepointEntry(row)
		if err != nil {
			return nil, problem.Internal(err)
		}
		items = append(items, entry)
	}
	if err := s.attachMoemoepointPaths(items, page.Items); err != nil {
		return nil, problem.Internal(err)
	}
	var next *string
	if page.HasMore && len(page.Items) > 0 {
		last := page.Items[len(page.Items)-1]
		cur := collect.EncodeCursor(moemoepointEntrySort, fp, strconv.FormatInt(last.ID, 10))
		next = &cur
	}
	return &listMoemoepointEntriesOutput{Body: repr.NewList(items, next)}, nil
}

const accountCenterSourceApp = "oauth"

func moemoepointSource(row userclient.MoemoepointLogEntry) string {
	switch {
	case row.IsLocal:
		return "this_site"
	case row.SourceApp == accountCenterSourceApp:
		return "account_center"
	default:
		return "other_site"
	}
}

func mapMoemoepointEntry(row userclient.MoemoepointLogEntry) (MoemoepointEntry, error) {
	created, err := time.Parse(time.RFC3339Nano, row.CreatedAt)
	if err != nil {
		return MoemoepointEntry{}, fmt.Errorf("moemoepoint entry %d created_at: %w", row.ID, err)
	}
	return MoemoepointEntry{
		Object:    "moemoepoint_entry",
		ID:        repr.DecimalID(strconv.FormatInt(row.ID, 10)),
		Delta:     row.Delta,
		Reason:    MoemoepointReason(row.Reason),
		Ref:       row.Ref,
		CreatedAt: repr.Timestamp(created),
		Source:    moemoepointSource(row),
	}, nil
}
