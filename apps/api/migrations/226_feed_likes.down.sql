-- Removes the like triggers and the like rows 226 put in feed_activity. Each
-- deleted row enqueues itself, so the pusher withdraws what it sent.
BEGIN;

DROP TRIGGER IF EXISTS trg_feed_topic_comment_likes ON topic_comment;
DROP TRIGGER IF EXISTS trg_feed_topic_reply_likes ON topic_reply;
DROP TRIGGER IF EXISTS trg_feed_topic_likes ON topic;
DROP TRIGGER IF EXISTS trg_feed_galgame_rating_like ON galgame_rating_like;
DROP TRIGGER IF EXISTS trg_feed_galgame_resource_like ON galgame_resource_like;
DROP TRIGGER IF EXISTS trg_feed_galgame_like ON galgame_like;
DROP TRIGGER IF EXISTS trg_feed_topic_comment_like ON topic_comment_like;
DROP TRIGGER IF EXISTS trg_feed_topic_reply_reaction ON topic_reply_reaction;
DROP TRIGGER IF EXISTS trg_feed_topic_reaction ON topic_reaction;

DROP FUNCTION IF EXISTS feed_sync_topic_comment_likes();
DROP FUNCTION IF EXISTS feed_sync_topic_reply_likes();
DROP FUNCTION IF EXISTS feed_sync_topic_likes();
DROP FUNCTION IF EXISTS feed_rebuild_topic_likes(int);
DROP FUNCTION IF EXISTS feed_sync_galgame_rating_like();
DROP FUNCTION IF EXISTS feed_sync_galgame_resource_like();
DROP FUNCTION IF EXISTS feed_sync_galgame_like();
DROP FUNCTION IF EXISTS feed_sync_topic_comment_like();
DROP FUNCTION IF EXISTS feed_sync_topic_reply_reaction();
DROP FUNCTION IF EXISTS feed_sync_topic_reaction();

DELETE FROM feed_activity WHERE type IN (
    'TOPIC_LIKE', 'TOPIC_REPLY_LIKE', 'TOPIC_COMMENT_LIKE',
    'GALGAME_LIKE', 'GALGAME_RESOURCE_LIKE', 'GALGAME_RATING_LIKE'
);

COMMIT;
