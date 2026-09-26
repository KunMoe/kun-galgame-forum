package apiv1

import (
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/message/notifytype"
)

type Notification struct {
	Object           string          `json:"object" enum:"notification" maxLength:"12" doc:"Type discriminant. Always notification."`
	ID               repr.DecimalID  `json:"id" doc:"Notification id. JSON string of a decimal integer."`
	NotificationType notifytype.Type `json:"notification_type" doc:"Notification type. Closed vocabulary of v1 tokens."`
	Actor            repr.UserRef    `json:"actor" doc:"The user who triggered this notification. name is null when the account no longer exists."`
	ActorCount       int             `json:"actor_count" minimum:"0" doc:"How many people are folded into this mirrored row. Values stored below 1 are emitted as 1. followed_thread_activity and user_followed can be greater than 1."`
	ItemCount        int             `json:"item_count" minimum:"0" doc:"How many upstream posts are folded into this mirrored row. Values stored below 1 are emitted as 1."`
	Path             string          `json:"path" maxLength:"100" pattern:"^/" doc:"In-site web path of the target, stored as the legacy link. Always starts with a slash."`
	ExcerptMarkdown  string          `json:"excerpt_markdown" maxLength:"1000" doc:"Markdown snapshot stored on the row, truncated to 1000 runes. May be empty. Free text; never use it as a decision input."`
	Origin           string          `json:"origin" enum:"local,community" maxLength:"9" doc:"local for rows written by this forum; community for rows mirrored from the infra community primitive."`
	IsRead           bool            `json:"is_read" doc:"Whether this notification has been marked read."`
	CreatedAt        repr.DateTime   `json:"created_at" doc:"Creation time. For a mirrored row this is the upstream updated_at, and folding may move it forward."`
}

type NotificationSummary struct {
	Object                   string        `json:"object" enum:"notification_summary" maxLength:"21" doc:"Type discriminant. Always notification_summary."`
	UnreadCount              int           `json:"unread_count" minimum:"0" doc:"Unread rows in the unmuted partition, counted in SQL. Includes rows whose actor is banned, which listNotifications drops, so this is not the length of that list."`
	MutedUnreadCount         int           `json:"muted_unread_count" minimum:"0" doc:"Unread rows in the muted partition, counted in SQL. Includes rows whose actor is banned."`
	DirectMessageUnreadCount int           `json:"direct_message_unread_count" minimum:"0" doc:"Direct messages from others that the caller has not marked read: the sum of unread_count over the conversations listConversations returns. Conversations with a banned peer are left out, as that list leaves them out. Counted whether or not direct messages are muted."`
	IsDirectMessageMuted     bool          `json:"is_direct_message_muted" doc:"Whether the caller muted direct messages, that is, muted_types in getNotificationPreferences contains chat. has_unread_messages on getMe ignores direct messages while this is true."`
	Latest                   *Notification `json:"latest" doc:"The first renderable row of the unmuted partition in listNotifications order. null when that partition has none."`
}

type NotificationReadMarker struct {
	Object      string `json:"object" enum:"notification_read_marker" maxLength:"24" doc:"Type discriminant. Always notification_read_marker."`
	MarkedCount int    `json:"marked_count" minimum:"0" doc:"How many unread rows this request marked. Zero on a replay."`
	UnreadCount int    `json:"unread_count" minimum:"0" doc:"Unread rows remaining in this partition after the mark, counted in SQL."`
}

type DirectMessageViewer struct {
	IsMine bool `json:"is_mine" doc:"Whether the caller sent this message."`
}

type DirectMessage struct {
	Object     string                  `json:"object" enum:"direct_message" maxLength:"14" doc:"Type discriminant. Always direct_message."`
	ID         repr.DecimalID          `json:"id" doc:"Message id. JSON string of a decimal integer."`
	Sender     repr.UserRef            `json:"sender" doc:"The user who sent the message."`
	State      string                  `json:"state" enum:"sent,recalled" maxLength:"8" doc:"Lifecycle state. recalled is irreversible."`
	Content    content.ContentDocument `json:"content" doc:"Message body as a node tree. An empty document (children is an empty array) when state is recalled; clients read state, not emptiness."`
	CreatedAt  repr.DateTime           `json:"created_at" doc:"Creation time."`
	RecalledAt *repr.DateTime          `json:"recalled_at" doc:"Time of the recall. null while state is sent."`
	Viewer     DirectMessageViewer     `json:"viewer" doc:"The caller's own state on this message."`
}

type Conversation struct {
	Object        string         `json:"object" enum:"conversation" maxLength:"12" doc:"Type discriminant. Always conversation."`
	ID            repr.DecimalID `json:"id" doc:"Equals peer.id and the user_id path segment. The conversation has no other identity."`
	Peer          repr.UserRef   `json:"peer" doc:"The other participant."`
	MessageCount  int            `json:"message_count" minimum:"0" doc:"Messages in the conversation, including recalled ones."`
	UnreadCount   int            `json:"unread_count" minimum:"0" doc:"Messages the peer sent that the caller has not marked read."`
	LastMessage   *DirectMessage `json:"last_message" doc:"The message with the greatest id in the conversation. null when there are no messages."`
	LastMessageAt *repr.DateTime `json:"last_message_at" doc:"created_at of last_message. null when there are no messages."`
}

type DirectMessageReadMarker struct {
	Object      string `json:"object" enum:"direct_message_read_marker" maxLength:"26" doc:"Type discriminant. Always direct_message_read_marker."`
	MarkedCount int    `json:"marked_count" minimum:"0" doc:"How many of the peer's messages this request marked read. Zero on a replay."`
	UnreadCount int    `json:"unread_count" minimum:"0" doc:"The peer's messages still unmarked after this request."`
}

type NotificationReadMarkerWrite struct {
	UpToID  repr.DecimalID `json:"up_to_id" doc:"Inclusive upper bound. Rows with id greater than this are left unread. Need not name a row the caller owns."`
	IsMuted bool           `json:"is_muted" required:"false" doc:"When true, only the muted partition is marked. When false or omitted, only the unmuted partition is marked. Default false."`
}

type DirectMessageCreate struct {
	ContentMarkdown string `json:"content_markdown" minLength:"1" maxLength:"1000" doc:"Message body as Markdown source. Length is checked on the raw value. After NormalizeStoredContent, a value that is empty once leading and trailing whitespace is removed is refused as REQUIRED. Free text; never use it as a decision input."`
}

type DirectMessagePatch struct {
	State string `json:"state" enum:"recalled" maxLength:"8" doc:"Target state. The only legal value is recalled; recall is irreversible, so sent is not a target."`
}

type DirectMessageReadMarkerWrite struct {
	UpToID repr.DecimalID `json:"up_to_id" doc:"Inclusive upper bound. Only the peer's messages with id at most this are marked. Need not name a row that exists."`
}
