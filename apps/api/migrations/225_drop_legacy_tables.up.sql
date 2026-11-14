-- 225: drop the tables and columns nothing reads or writes any more.
--
-- Each went quiet when its feature moved. Last writes on production, read on
-- 2026-11-14:
--
-- * topic_like, topic_reply_like (2026-06-24), topic_dislike (06-04),
--   topic_reply_dislike (06-14): topic and reply reactions moved to
--   topic_reaction / topic_reply_reaction. The *_count columns are recounted
--   from those.
-- * galgame_favorite (2026-07-06): favorites became collections.
-- * galgame_collection_item (2026-09-08), galgame_collection_viewer (empty):
--   collections moved to catalog folders. galgame_collection stays; it maps a
--   frozen forum collection id to its folder for old links.
-- * galgame_resource_provider (2026-06-04): providers live on the resource.
-- * system_message (empty), system_message_read_state (75,799 rows of read
--   marks for a table with no rows).
-- * chat_message_reaction, chat_room_admin, user_follow, user_friend,
--   galgame_toolset_category, galgame_toolset_category_relation: empty.
-- * _user_merge_109310_to_1922: 13 bookkeeping rows of one manual user merge.
-- * chat_message.edit_time (null on all 10,047 rows),
--   galgame_activity.wiki_pr_id (the retired wiki's PR id, 531 rows),
--   kungal_user_state.following_activity_seen_at (one row; the seen mark is
--   community's).
--
-- galgame_merge_fold_row still lists 17 galgame_favorite rows from folds made
-- before this; unfolding skips them (retiredMergeTables), and the fold API
-- keeps the name in its table_name vocabulary so those ledgers still render.
--
-- Deploy the release that carries this migration before running it. The build
-- before it still deletes from topic_reply_like / topic_reply_dislike when a
-- reply is removed, and from several of these tables when a user is purged;
-- with the tables gone those requests fail.
--
-- The rows are not kept anywhere in this database. A full pg_dump of production
-- was taken on 2026-11-14 before this migration.
BEGIN;

DROP TABLE IF EXISTS galgame_toolset_category_relation;
DROP TABLE IF EXISTS galgame_toolset_category;
DROP TABLE IF EXISTS topic_like;
DROP TABLE IF EXISTS topic_dislike;
DROP TABLE IF EXISTS topic_reply_like;
DROP TABLE IF EXISTS topic_reply_dislike;
DROP TABLE IF EXISTS galgame_favorite;
DROP TABLE IF EXISTS galgame_collection_item;
DROP TABLE IF EXISTS galgame_collection_viewer;
DROP TABLE IF EXISTS galgame_resource_provider;
DROP TABLE IF EXISTS system_message_read_state;
DROP TABLE IF EXISTS system_message;
DROP TABLE IF EXISTS chat_message_reaction;
DROP TABLE IF EXISTS chat_room_admin;
DROP TABLE IF EXISTS user_follow;
DROP TABLE IF EXISTS user_friend;
DROP TABLE IF EXISTS _user_merge_109310_to_1922;

ALTER TABLE chat_message DROP COLUMN IF EXISTS edit_time;
ALTER TABLE galgame_activity DROP COLUMN IF EXISTS wiki_pr_id;
ALTER TABLE kungal_user_state DROP COLUMN IF EXISTS following_activity_seen_at;

COMMIT;
