package apiv1

import (
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"

	"github.com/danielgtaylor/huma/v2"
)

const interactionVisibility = "NOT_FOUND when the topic is hidden or getTopic would not return it to the caller."

func RegisterInteractions(x *Interactions) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "setTopicReaction",
			Method:      http.MethodPut,
			Path:        "/topics/{topic_id}/reactions/{reaction}",
			Summary:     "React to a topic",
			Description: "Adds the caller's reaction with this token. Setting one already set changes nothing. " +
				"like and dislike exclude each other: setting one removes the other. " +
				"A like earns the author 1 moemoepoint and notifies them. " + interactionVisibility,
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "SELF_LIKE_FORBIDDEN when the caller likes their own topic; SCOPE_REQUIRED or ACCOUNT_BANNED.",
			}),
		}), x.setTopicReaction)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "removeTopicReaction",
			Method:      http.MethodDelete,
			Path:        "/topics/{topic_id}/reactions/{reaction}",
			Summary:     "Remove a reaction from a topic",
			Description: "Removes the caller's reaction with this token and returns the topic's engagement. Removing one not set changes nothing. " +
				"Removing a like takes back the moemoepoint it earned. " + interactionVisibility,
			Tags: []string{"topics"},
		}), x.removeTopicReaction)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "favoriteTopic",
			Method:      http.MethodPut,
			Path:        "/topics/{topic_id}/favorite",
			Summary:     "Favorite a topic",
			Description: "Adds the topic to the caller's favorites. Favoriting it again changes nothing. " +
				"Another user's favorite earns the author 1 moemoepoint and notifies them. " + interactionVisibility,
			Tags: []string{"topics"},
		}), x.favoriteTopic)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "unfavoriteTopic",
			Method:      http.MethodDelete,
			Path:        "/topics/{topic_id}/favorite",
			Summary:     "Unfavorite a topic",
			Description: "Removes the topic from the caller's favorites and returns its engagement. Removing one not favorited changes nothing. " +
				"It takes back the moemoepoint the favorite earned. " + interactionVisibility,
			Tags: []string{"topics"},
		}), x.unfavoriteTopic)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "likeComment",
			Method:      http.MethodPut,
			Path:        "/comments/{comment_id}/like",
			Summary:     "Like a comment",
			Description: "Adds the caller's like and returns the comment. Liking one already liked changes nothing. " +
				"It earns the comment's author 1 moemoepoint and notifies them once. " +
				"NOT_FOUND when getComment would not return the comment to the caller.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "SELF_LIKE_FORBIDDEN when the caller likes their own comment; SCOPE_REQUIRED or ACCOUNT_BANNED.",
			}),
		}), x.likeComment)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "unlikeComment",
			Method:      http.MethodDelete,
			Path:        "/comments/{comment_id}/like",
			Summary:     "Remove a like from a comment",
			Description: "Removes the caller's like and returns the comment. Removing one not set changes nothing. " +
				"It takes back the moemoepoint the like earned, but not the notification it sent. " +
				"NOT_FOUND when getComment would not return the comment to the caller.",
			Tags: []string{"topics"},
		}), x.unlikeComment)

		huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
			OperationID:   "upvoteTopic",
			Method:        http.MethodPost,
			Path:          "/topics/{topic_id}/upvotes",
			Summary:       "Upvote a topic",
			DefaultStatus: http.StatusCreated,
			Description: "Records an upvote. It costs the caller 10 moemoepoint, earns the author 5, sets upvoted_at and bumps the topic. " +
				"A user may upvote the same topic again; each upvote is charged. It cannot be undone. " + interactionVisibility,
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "SELF_UPVOTE_FORBIDDEN, or MOEMOEPOINT_INSUFFICIENT when the caller's cached balance is below 10; SCOPE_REQUIRED or ACCOUNT_BANNED.",
			}),
		})), x.upvoteTopic)

		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "listTopicUpvotes",
			Method:      http.MethodGet,
			Path:        "/topics/{topic_id}/upvotes",
			Summary:     "List a topic's upvotes",
			Description: "Lists upvotes newest first as a cursor page. Upvotes by banned users are left out; the server reads on to fill the page, " +
				"so continue while next_cursor is present, whatever the page size. NOT_FOUND under the same conditions as getTopic.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				400: "INVALID_PARAMETER, LIMIT_TOO_LARGE, or INVALID_CURSOR.",
			}),
		}), x.listTopicUpvotes)

		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "listTopicReactions",
			Method:      http.MethodGet,
			Path:        "/topics/{topic_id}/reactions",
			Summary:     "List a topic's reactions",
			Description: "Lists who reacted with what, newest first, as a cursor page. Reactions by banned users are left out; the server reads on to fill the page, " +
				"so continue while next_cursor is present, whatever the page size. NOT_FOUND under the same conditions as getTopic.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				400: "INVALID_PARAMETER, LIMIT_TOO_LARGE, or INVALID_CURSOR.",
			}),
		}), x.listTopicReactions)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "setBestAnswer",
			Method:      http.MethodPut,
			Path:        "/topics/{topic_id}/best-answer",
			Summary:     "Set the best answer",
			Description: "Marks a reply of this topic as the best answer and returns the topic. It needs can_set_best_answer. Setting the current one changes nothing. " +
				"It bumps the topic. The reply's author earns 7 moemoepoint and is notified, unless they are the topic's author; " +
				"a replaced best answer's author loses the 7 it earned. " + interactionVisibility,
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks can_set_best_answer; SCOPE_REQUIRED or ACCOUNT_BANNED.",
				422: "VALIDATION_FAILED at /reply_id when the reply is not a visible reply of this topic.",
			}),
		}), x.setBestAnswer)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "clearBestAnswer",
			Method:      http.MethodDelete,
			Path:        "/topics/{topic_id}/best-answer",
			Summary:     "Clear the best answer",
			Description: "Removes the best-answer mark and returns the topic. It needs can_set_best_answer. Clearing when none is set changes nothing. " +
				"The reply's author loses the 7 moemoepoint it earned. " + interactionVisibility,
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks can_set_best_answer; SCOPE_REQUIRED or ACCOUNT_BANNED.",
			}),
		}), x.clearBestAnswer)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "pinReply",
			Method:      http.MethodPut,
			Path:        "/topics/{topic_id}/pinned-reply",
			Summary:     "Pin a reply",
			Description: "Pins a reply of this topic, replacing any pinned one, and returns the topic. It needs can_pin_reply. Pinning the pinned one changes nothing. " +
				"The reply's author is notified unless they pinned it themselves. " + interactionVisibility,
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks can_pin_reply; SCOPE_REQUIRED or ACCOUNT_BANNED.",
				422: "VALIDATION_FAILED at /reply_id when the reply is not a visible reply of this topic.",
			}),
		}), x.pinReply)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "unpinReply",
			Method:      http.MethodDelete,
			Path:        "/topics/{topic_id}/pinned-reply",
			Summary:     "Unpin the pinned reply",
			Description: "Unpins the pinned reply and returns the topic. It needs can_pin_reply. Unpinning when none is pinned changes nothing. " +
				interactionVisibility,
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks can_pin_reply; SCOPE_REQUIRED or ACCOUNT_BANNED.",
			}),
		}), x.unpinReply)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "setReplyReaction",
			Method:      http.MethodPut,
			Path:        "/replies/{reply_id}/reactions/{reaction}",
			Summary:     "React to a reply",
			Description: "Adds the caller's reaction with this token. Setting one already set changes nothing. " +
				"like and dislike exclude each other: setting one removes the other. " +
				"A like earns the reply's author 1 moemoepoint and notifies them. " +
				"NOT_FOUND when getReply would not return the reply to the caller or its topic is hidden.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "SELF_LIKE_FORBIDDEN when the caller likes their own reply; SCOPE_REQUIRED or ACCOUNT_BANNED.",
			}),
		}), x.setReplyReaction)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "removeReplyReaction",
			Method:      http.MethodDelete,
			Path:        "/replies/{reply_id}/reactions/{reaction}",
			Summary:     "Remove a reaction from a reply",
			Description: "Removes the caller's reaction with this token and returns the reply's engagement. Removing one not set changes nothing. " +
				"Removing a like takes back the moemoepoint it earned. " +
				"NOT_FOUND when getReply would not return the reply to the caller or its topic is hidden.",
			Tags: []string{"topics"},
		}), x.removeReplyReaction)

		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "listReplyReactions",
			Method:      http.MethodGet,
			Path:        "/replies/{reply_id}/reactions",
			Summary:     "List a reply's reactions",
			Description: "Lists who reacted with what, newest first, as a cursor page. Reactions by banned users are left out; the server reads on to fill the page, " +
				"so continue while next_cursor is present, whatever the page size. NOT_FOUND under the same conditions as getReply.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				400: "INVALID_PARAMETER, LIMIT_TOO_LARGE, or INVALID_CURSOR.",
			}),
		}), x.listReplyReactions)

		registerSubscriptions(api, x)
	}
}
