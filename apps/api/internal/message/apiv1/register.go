package apiv1

import (
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"

	"github.com/danielgtaylor/huma/v2"
)

func Register(svc *Service) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "listNotifications",
			Method:      http.MethodGet,
			Path:        "/me/notifications",
			Summary:     "List the caller's notifications",
			Description: "Lists the caller's notifications as a cursor page, newest first, ties broken by descending id. " +
				"There is one sort and no sort parameter. " +
				"is_muted=false is the types the caller has not muted; is_muted=true is only the muted types. The partitions are disjoint and together are every row. " +
				"An empty muted list makes the muted partition empty and the default partition everything. " +
				"notification_type further intersects the chosen partition; is_muted=false with a muted notification_type is a legal empty list. " +
				"The cursor is bound to (caller, is_muted, notification_type); reusing it with another combination is INVALID_CURSOR. " +
				"Rows whose actor is banned are dropped; the server keeps fetching limit+1 batches until it has limit+1 renderable rows or the source is exhausted, scanning at most 10 times limit rows, and emits next_cursor at the last scanned row if that cap is hit. " +
				"A missing actor is emitted as a user with name null. Unknown stored types are dropped. " +
				"unread_count lives on the summary, not here: the SQL count includes banned actors that this list drops.",
			Tags: []string{"messages"},
			Responses: problemResponses(map[int]string{
				400: "INVALID_CURSOR, LIMIT_TOO_LARGE, or UNKNOWN_ENUM_VALUE.",
			}),
		}), svc.listNotifications)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getNotificationSummary",
			Method:      http.MethodGet,
			Path:        "/me/notifications/summary",
			Summary:     "Get notification unread counts",
			Description: "Returns unread_count for the unmuted partition, muted_unread_count for the muted partition, and latest, the first renderable unmuted row in listNotifications order. " +
				"The two counts are SQL counts and include rows whose actor is banned, which listNotifications drops, so they are not the length of that list. " +
				"latest is null when the unmuted partition has no renderable row. Any query failure is INTERNAL_ERROR.",
			Tags: []string{"messages"},
		}), svc.getNotificationSummary)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "markNotificationsRead",
			Method:      http.MethodPut,
			Path:        "/me/notifications/read-marker",
			Summary:     "Mark notifications read up to an id",
			Description: "Marks unread notifications in one partition whose id is at most up_to_id as read. " +
				"is_muted selects the partition; the other partition is untouched, as are rows with id greater than up_to_id. " +
				"up_to_id need not name a row the caller owns. " +
				"Banned actors' rows are marked, so a red dot that only those rows held can go out. " +
				"Mirrored rows that were marked are forwarded to community by id. " +
				"Replaying the same request returns marked_count 0 and is still 200.",
			Tags: []string{"messages"},
		}), svc.markNotificationsRead)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID:   "deleteNotification",
			Method:        http.MethodDelete,
			Path:          "/me/notifications/{notification_id}",
			Summary:       "Delete a notification",
			DefaultStatus: http.StatusNoContent,
			Description:   "Deletes one of the caller's notifications. A missing id or another user's notification is NOT_FOUND; the two are indistinguishable. Deleting an unread mirrored row forwards a read to community.",
			Tags:          []string{"messages"},
			Responses: problemResponses(map[int]string{
				404: "NOT_FOUND when the notification does not exist or belongs to another user.",
			}),
		}), svc.deleteNotification)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getNotification",
			Method:      http.MethodGet,
			Path:        "/me/notifications/{notification_id}",
			Summary:     "Get a notification",
			Description: "Returns one of the caller's notifications. " +
				"A missing id, another user's notification, a stored type outside the vocabulary, or an actor that is not renderable is NOT_FOUND; they are indistinguishable.",
			Tags: []string{"messages"},
			Responses: problemResponses(map[int]string{
				404: "NOT_FOUND when the notification does not exist, belongs to another user, has a type outside the vocabulary, or its actor is not renderable.",
			}),
		}), svc.getNotification)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "listConversations",
			Method:      http.MethodGet,
			Path:        "/me/conversations",
			Summary:     "List the caller's direct-message conversations",
			Description: "Lists conversations that have at least one message, newest last message first, ties broken by descending room id, which lives only inside the cursor. " +
				"last_message and last_message_at come from the chat_message row with the greatest id, not from chat_room.last_message_*. " +
				"A conversation whose peer is banned is dropped with the same page-refill rule as listNotifications. " +
				"The cursor is bound to the caller.",
			Tags: []string{"messages"},
			Responses: problemResponses(map[int]string{
				400: "INVALID_CURSOR or LIMIT_TOO_LARGE.",
			}),
		}), svc.listConversations)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getConversation",
			Method:      http.MethodGet,
			Path:        "/me/conversations/{user_id}",
			Summary:     "Get a conversation with one user",
			Description: "Returns the conversation with user_id. A conversation with no messages, including when no room exists, is 200 with message_count 0, unread_count 0, last_message null and last_message_at null. It does not create a room. " +
				"A missing or banned peer is NOT_FOUND; user_id equal to the caller is NOT_FOUND; a failure of /users/batch is SERVICE_UNAVAILABLE.",
			Tags: []string{"messages"},
			Responses: problemResponses(map[int]string{
				404: "NOT_FOUND when the peer does not exist, is not renderable, or is the caller.",
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
			}),
		}), svc.getConversation)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "listDirectMessages",
			Method:      http.MethodGet,
			Path:        "/me/conversations/{user_id}/messages",
			Summary:     "List direct messages with one user",
			Description: "Lists messages in the conversation with user_id as a cursor page, greatest id first. " +
				"A missing room is 200 with an empty list; it does not create a room and does not mark messages read. " +
				"Rows are selected by chat_room_id only. " +
				"A missing or banned peer is NOT_FOUND; user_id equal to the caller is NOT_FOUND; a failure of /users/batch is SERVICE_UNAVAILABLE. " +
				"The cursor is bound to (caller, peer).",
			Tags: []string{"messages"},
			Responses: problemResponses(map[int]string{
				400: "INVALID_CURSOR or LIMIT_TOO_LARGE.",
				404: "NOT_FOUND when the peer does not exist, is not renderable, or is the caller.",
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
			}),
		}), svc.listDirectMessages)

		huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
			OperationID:   "sendDirectMessage",
			Method:        http.MethodPost,
			Path:          "/me/conversations/{user_id}/messages",
			Summary:       "Send a direct message",
			DefaultStatus: http.StatusCreated,
			Description: "Creates a message in the conversation with user_id and returns it. Idempotency-Key is required. " +
				"content_markdown is length-checked on the raw value (max 1000). After NormalizeStoredContent, a value that is empty once whitespace is trimmed is VALIDATION_FAILED REQUIRED at /content_markdown. " +
				"A missing or banned peer is NOT_FOUND; user_id equal to the caller is NOT_FOUND; a failure of /users/batch is SERVICE_UNAVAILABLE and nothing is written. " +
				"A missing room is created by name <smaller id>-<larger id> with INSERT ON CONFLICT (name) DO NOTHING, then selected by name; both participants use ON CONFLICT (chat_room_id, user_id) DO NOTHING, so a concurrent first send is not 500. " +
				"Location is the canonical path of the new message.",
			Tags: []string{"messages"},
			Responses: problemResponses(map[int]string{
				404: "NOT_FOUND when the peer does not exist, is not renderable, or is the caller.",
				422: "VALIDATION_FAILED when content_markdown is blank after trimming or exceeds 1000 characters.",
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached. Nothing is written.",
			}),
		})), svc.sendDirectMessage)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "updateDirectMessage",
			Method:      http.MethodPatch,
			Path:        "/me/conversations/{user_id}/messages/{message_id}",
			Summary:     "Recall a direct message",
			Description: "Recalls the caller's message. state must be recalled; recall is irreversible, so sent is not a target. " +
				"There is no time limit. " +
				"A message that is not in this conversation is NOT_FOUND. A message the peer sent is PERMISSION_REQUIRED; the caller can see it, so it is not NOT_FOUND. " +
				"Already recalled is 200 with no further write. " +
				"A recalled message is still in both histories as a tombstone: state recalled, content an empty document, recalled_at set. " +
				"If it was the latest message in the room, chat_room.last_message_content is set to the empty string; the server never writes a sentence there.",
			Tags: []string{"messages"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the message was sent by the peer.",
				404: "NOT_FOUND when the message is not in this conversation, or user_id is the caller.",
			}),
		}), svc.updateDirectMessage)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getDirectMessage",
			Method:      http.MethodGet,
			Path:        "/me/conversations/{user_id}/messages/{message_id}",
			Summary:     "Get a direct message",
			Description: "Returns one message in the conversation with user_id. " +
				"A missing room is NOT_FOUND. A message that is not in this conversation is NOT_FOUND. " +
				"A missing or banned peer is NOT_FOUND; user_id equal to the caller is NOT_FOUND; a failure of /users/batch is SERVICE_UNAVAILABLE. " +
				"A recalled message is a tombstone: state recalled, content an empty document, recalled_at set.",
			Tags: []string{"messages"},
			Responses: problemResponses(map[int]string{
				404: "NOT_FOUND when the message is not in this conversation, the room is missing, the peer does not exist, is not renderable, or is the caller.",
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
			}),
		}), svc.getDirectMessage)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "markDirectMessagesRead",
			Method:      http.MethodPut,
			Path:        "/me/conversations/{user_id}/read-marker",
			Summary:     "Mark direct messages read up to an id",
			Description: "Marks the peer's messages in this conversation whose id is at most up_to_id as read by the caller, writing chat_message_read_by with ON CONFLICT DO NOTHING. " +
				"A missing room is 200 with marked_count 0 and unread_count 0. " +
				"A missing or banned peer is NOT_FOUND; user_id equal to the caller is NOT_FOUND; a failure of /users/batch is SERVICE_UNAVAILABLE. " +
				"Replaying the same request is 200.",
			Tags: []string{"messages"},
			Responses: problemResponses(map[int]string{
				404: "NOT_FOUND when the peer does not exist, is not renderable, or is the caller.",
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
			}),
		}), svc.markDirectMessagesRead)
	}
}
