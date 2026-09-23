package apiv1

import (
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"

	"github.com/danielgtaylor/huma/v2"
)

const (
	pageNote   = "A page-number collection: page × limit may not exceed 10000, and total counts under the same filters as items. "
	bannedNote = "Posts by banned authors, or under topics by banned authors, are dropped after counting, so a page may hold fewer items than limit and total may count them. "
	blankQ     = "INVALID_PARAMETER when q is blank after trimming, or the page is past the depth limit."
)

func Register(s *Service) func(huma.API) {
	return func(api huma.API) {
		tags := []string{"search"}

		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "searchTopics",
			Method:      http.MethodGet,
			Path:        "/search/topics",
			Summary:     "Search topics",
			Description: "Topics whose title, body or category contains every keyword, most relevant first: title hits outrank category hits outrank body hits, and the keywords sitting next to each other outrank them scattered. " +
				"Ties break on bump time, then id, both descending. Hidden topics never match; login-only topics match for a signed-in caller. " + pageNote +
				"Topics by banned authors are dropped after counting, so a page may hold fewer items than limit.",
			Tags: tags,
			Responses: problemResponses(map[int]string{
				400: blankQ,
				503: "SERVICE_UNAVAILABLE when the account service cannot say who wrote these topics.",
			}),
		}), s.searchTopics)

		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "searchReplies",
			Method:      http.MethodGet,
			Path:        "/search/replies",
			Summary:     "Search replies",
			Description: "Replies whose body contains every keyword, under topics the caller could find with searchTopics, most relevant first; ties break on posting time, then id, both descending. " + pageNote + bannedNote,
			Tags:        tags,
			Responses: problemResponses(map[int]string{
				400: blankQ,
				503: "SERVICE_UNAVAILABLE when the account service cannot say whose posts these are.",
			}),
		}), s.searchReplies)

		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "searchComments",
			Method:      http.MethodGet,
			Path:        "/search/comments",
			Summary:     "Search topic comments",
			Description: "Topic comments whose body contains every keyword, under topics the caller could find with searchTopics, most relevant first; ties break on posting time, then id, both descending. " + pageNote + bannedNote,
			Tags:        tags,
			Responses: problemResponses(map[int]string{
				400: blankQ,
				503: "SERVICE_UNAVAILABLE when the account service cannot say whose posts these are.",
			}),
		}), s.searchComments)

		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "searchUsers",
			Method:      http.MethodGet,
			Path:        "/search/users",
			Summary:     "Search users",
			Description: "Accounts whose name matches, in the account service's order; banned and deleted accounts are left out. " +
				"The account service answers at most 50 matches and cannot page past them, so total stops at 50 with total_relation gte when there may be more. " +
				"topic_count counts login-only topics for a signed-in caller.",
			Tags: tags,
			Responses: problemResponses(map[int]string{
				400: blankQ,
				503: "SERVICE_UNAVAILABLE when the account service is unreachable.",
			}),
		}), s.searchUsers)

		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "searchWorks",
			Method:      http.MethodGet,
			Path:        "/search/works",
			Summary:     "Search works",
			Description: "Works in the catalog search index, in the chosen order. Filters narrow the same request. " +
				"Works this forum displays as adult are left out unless include_nsfw is true. total is the index's count. " + pageNote,
			Tags: tags,
			Responses: problemResponses(map[int]string{
				400: "INVALID_PARAMETER when q is blank after trimming, released_from is after released_to, or the page is past the depth limit.",
				503: "SERVICE_UNAVAILABLE when the catalog is unreachable or refuses the query.",
			}),
		}), s.searchWorks)

		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "searchWallComments",
			Method:      http.MethodGet,
			Path:        "/search/wall-comments",
			Summary:     "Search comment walls",
			Description: "Posts on the comment walls this forum hosts (works, ratings, resources, quizzes, toolsets and websites) that match the keywords, newest first. " +
				"A cursor collection with no total: the comment index answers a cursor and no count. Posts on walls another site opened, and posts by banned authors, are dropped after the fact, so a page may hold fewer items than limit. " +
				"A cursor is bound to q.",
			Tags: tags,
			Responses: problemResponses(map[int]string{
				400: "INVALID_PARAMETER when q has fewer than 2 characters after trimming; INVALID_CURSOR when the cursor is malformed or came from another q.",
				503: "SERVICE_UNAVAILABLE when the community service or the account service is unreachable.",
			}),
		}), s.searchWallComments)
	}
}
