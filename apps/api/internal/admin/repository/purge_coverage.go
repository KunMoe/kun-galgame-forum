package repository

type ColumnHandling string

const (
	HandlingDelete   ColumnHandling = "delete"
	HandlingTransfer ColumnHandling = "transfer"
	HandlingRelease  ColumnHandling = "release"
	HandlingKeep     ColumnHandling = "keep"
)

type UserColumn struct {
	Table    string
	Column   string
	Handling ColumnHandling
	Why      string
}

// UserColumns is every column that names a user, and what a purge does to it.
// TestPurgeCoverage fails when the schema holds a user column this list does
// not name: the purge's table list is hand-written, and on 2026-08-30 it
// covered about 30 of the 58 tables that had one.
var UserColumns = []UserColumn{
	{"chat_message", "receiver_id", HandlingDelete, "the private room goes with its sender"},
	{"chat_message", "sender_id", HandlingDelete, ""},
	{"chat_message_reaction", "user_id", HandlingDelete, ""},
	{"chat_message_read_by", "user_id", HandlingDelete, ""},
	{"chat_room", "last_message_sender_id", HandlingRelease, "recomputed from the messages left in the room"},
	{"chat_room_admin", "user_id", HandlingDelete, ""},
	{"chat_room_participant", "user_id", HandlingDelete, ""},
	{"doc_article", "author_id", HandlingKeep, "staff content; staff are never purged"},
	{"feed_activity", "user_id", HandlingDelete, ""},
	{"galgame", "creator_user_id", HandlingKeep, "a shared catalog row; the column is attribution only"},
	{"galgame_activity", "user_id", HandlingDelete, ""},
	{"galgame_collection", "user_id", HandlingDelete, ""},
	{"galgame_collection_item", "user_id", HandlingDelete, ""},
	{"galgame_collection_viewer", "user_id", HandlingDelete, ""},
	{"galgame_contributor", "user_id", HandlingDelete, ""},
	{"galgame_favorite", "user_id", HandlingDelete, ""},
	{"galgame_like", "user_id", HandlingDelete, ""},
	{"galgame_post_like", "user_id", HandlingDelete, ""},
	{"galgame_quiz", "user_id", HandlingDelete, ""},
	{"galgame_quiz_answer", "user_id", HandlingDelete, ""},
	{"galgame_quiz_favorite", "user_id", HandlingDelete, ""},
	{"galgame_rating", "user_id", HandlingDelete, ""},
	{"galgame_rating_like", "user_id", HandlingDelete, ""},
	{"galgame_resource", "user_id", HandlingDelete, ""},
	{"galgame_resource_like", "user_id", HandlingDelete, ""},
	{"galgame_toolset", "user_id", HandlingDelete, ""},
	{"galgame_toolset_contributor", "user_id", HandlingDelete, ""},
	{"galgame_toolset_practicality", "user_id", HandlingDelete, ""},
	{"galgame_toolset_resource", "user_id", HandlingDelete, ""},
	{"galgame_website", "user_id", HandlingTransfer, "a shared directory entry; it goes to the column's default owner"},
	{"galgame_website_favorite", "user_id", HandlingDelete, ""},
	{"galgame_website_like", "user_id", HandlingDelete, ""},
	{"kungal_user_state", "user_id", HandlingDelete, ""},
	{"message", "receiver_id", HandlingDelete, ""},
	{"message", "sender_id", HandlingDelete, ""},
	{"permission_audit_log", "operator_id", HandlingKeep, "audit trail"},
	{"role_permission_override", "updated_by", HandlingKeep, "audit trail"},
	{"system_message", "user_id", HandlingDelete, ""},
	{"system_message_read_state", "user_id", HandlingDelete, ""},
	{"todo", "claimed_user_id", HandlingRelease, "an open claim goes back to the board; a closed one keeps who worked it"},
	{"todo", "user_id", HandlingDelete, ""},
	{"toolset_upload", "user_id", HandlingDelete, ""},
	{"topic", "user_id", HandlingDelete, ""},
	{"topic_comment", "target_user_id", HandlingKeep, "another user's comment"},
	{"topic_comment", "user_id", HandlingDelete, ""},
	{"topic_comment_like", "user_id", HandlingDelete, ""},
	{"topic_dislike", "user_id", HandlingDelete, ""},
	{"topic_draft", "user_id", HandlingDelete, ""},
	{"topic_favorite", "user_id", HandlingDelete, ""},
	{"topic_like", "user_id", HandlingDelete, ""},
	{"topic_lottery", "user_id", HandlingDelete, ""},
	{"topic_lottery_code", "claimed_by", HandlingKeep, "a key already handed out; releasing it would let a second winner claim it"},
	{"topic_lottery_entry", "user_id", HandlingDelete, ""},
	{"topic_poll", "user_id", HandlingDelete, ""},
	{"topic_poll_vote", "user_id", HandlingDelete, ""},
	{"topic_reaction", "user_id", HandlingDelete, ""},
	{"topic_reply", "user_id", HandlingDelete, ""},
	{"topic_reply_dislike", "user_id", HandlingDelete, ""},
	{"topic_reply_like", "user_id", HandlingDelete, ""},
	{"topic_reply_reaction", "user_id", HandlingDelete, ""},
	{"topic_upvote", "user_id", HandlingDelete, ""},
	{"update_log", "user_id", HandlingKeep, "staff content; staff are never purged"},
	{"user_follow", "followed_id", HandlingDelete, ""},
	{"user_follow", "follower_id", HandlingDelete, ""},
	{"user_friend", "friend_id", HandlingDelete, ""},
	{"user_friend", "user_id", HandlingDelete, ""},
	{"user_permission_override", "updated_by", HandlingKeep, "audit trail"},
	{"user_purge_archive", "operator_id", HandlingKeep, "the purge archive itself"},
	{"user_purge_archive", "target_user_id", HandlingKeep, "the purge archive itself"},
	{"user_permission_override", "user_id", HandlingDelete, ""},
}
