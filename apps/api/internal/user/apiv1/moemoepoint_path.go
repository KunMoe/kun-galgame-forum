package apiv1

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/userclient"
)

func (s *Users) attachMoemoepointPaths(items []MoemoepointEntry, rows []userclient.MoemoepointLogEntry) error {
	if len(items) == 0 {
		return nil
	}
	replyIDs := map[int64]struct{}{}
	upvoteIDs := map[int64]struct{}{}
	workIDs := map[int64]struct{}{}
	for i, row := range rows {
		if items[i].Source != "this_site" {
			continue
		}
		kind, id, ok := parseMoemoepointRef(row.Ref)
		if !ok {
			continue
		}
		switch kind {
		case "topic_reply":
			replyIDs[id] = struct{}{}
		case "topic_upvote":
			upvoteIDs[id] = struct{}{}
		case "galgame", "galgame_pr":
			workIDs[id] = struct{}{}
		}
	}
	needLookup := len(replyIDs)+len(upvoteIDs)+len(workIDs) > 0
	if needLookup && s.paths == nil {
		return errUnconfigured
	}

	var replies map[int64]repository.TopicReplyRef
	var upvotes map[int64]int64
	var renumbers map[int64]repository.GalgameRenumber
	if s.paths != nil {
		var err error
		if len(replyIDs) > 0 {
			if replies, err = s.paths.TopicReplies(idKeys(replyIDs)); err != nil {
				return err
			}
		}
		if len(upvoteIDs) > 0 {
			if upvotes, err = s.paths.TopicUpvoteTopics(idKeys(upvoteIDs)); err != nil {
				return err
			}
		}
		if len(workIDs) > 0 {
			if renumbers, err = s.paths.GalgameRenumbers(idKeys(workIDs)); err != nil {
				return err
			}
		}
	}

	for i, row := range rows {
		items[i].RefPath = resolveMoemoepointPath(items[i].Source, row, replies, upvotes, renumbers)
	}
	return nil
}

func parseMoemoepointRef(ref string) (kind string, id int64, ok bool) {
	kind, rest, found := strings.Cut(ref, ":")
	if !found {
		return "", 0, false
	}
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil || id <= 0 || strconv.FormatInt(id, 10) != rest {
		return kind, 0, false
	}
	return kind, id, true
}

func resolveMoemoepointPath(
	source string,
	row userclient.MoemoepointLogEntry,
	replies map[int64]repository.TopicReplyRef,
	upvotes map[int64]int64,
	renumbers map[int64]repository.GalgameRenumber,
) *string {
	if source != "this_site" {
		return nil
	}
	kind, id, ok := parseMoemoepointRef(row.Ref)
	if !ok {
		return nil
	}
	switch kind {
	case "topic":
		return pathString(fmt.Sprintf("/topic/%d", id))
	case "topic_reply":
		reply, found := replies[id]
		if !found {
			return nil
		}
		return pathString(fmt.Sprintf("/topic/%d?reply=%d", reply.TopicID, reply.Floor))
	case "topic_upvote":
		topicID, found := upvotes[id]
		if !found {
			return nil
		}
		return pathString(fmt.Sprintf("/topic/%d", topicID))
	case "galgame", "galgame_pr":
		workID := id
		if mapped, found := renumbers[id]; found {
			created, err := time.Parse(time.RFC3339Nano, row.CreatedAt)
			// Ledger refs written before the 2026-09-23 renumber still name old_id; the account ledger was never rewritten.
			if err == nil && created.Before(mapped.Created) {
				workID = mapped.NewID
			}
		}
		return pathString(fmt.Sprintf("/galgame/%d", workID))
	case "galgame_quiz":
		return pathString(fmt.Sprintf("/galgame-quiz/%d", id))
	case "toolset":
		return pathString(fmt.Sprintf("/toolset/%d", id))
	default:
		return nil
	}
}

func pathString(p string) *string {
	return &p
}

func idKeys(ids map[int64]struct{}) []int64 {
	out := make([]int64, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	return out
}
