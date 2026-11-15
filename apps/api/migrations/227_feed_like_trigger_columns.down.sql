-- Puts back 226's like triggers, which fire on every UPDATE.
BEGIN;

DROP TRIGGER IF EXISTS trg_feed_topic_reaction ON topic_reaction;
CREATE TRIGGER trg_feed_topic_reaction AFTER INSERT OR UPDATE OR DELETE ON topic_reaction
    FOR EACH ROW EXECUTE FUNCTION feed_sync_topic_reaction();
DROP TRIGGER IF EXISTS trg_feed_topic_reply_reaction ON topic_reply_reaction;
CREATE TRIGGER trg_feed_topic_reply_reaction AFTER INSERT OR UPDATE OR DELETE ON topic_reply_reaction
    FOR EACH ROW EXECUTE FUNCTION feed_sync_topic_reply_reaction();
DROP TRIGGER IF EXISTS trg_feed_topic_comment_like ON topic_comment_like;
CREATE TRIGGER trg_feed_topic_comment_like AFTER INSERT OR UPDATE OR DELETE ON topic_comment_like
    FOR EACH ROW EXECUTE FUNCTION feed_sync_topic_comment_like();
DROP TRIGGER IF EXISTS trg_feed_galgame_like ON galgame_like;
CREATE TRIGGER trg_feed_galgame_like AFTER INSERT OR UPDATE OR DELETE ON galgame_like
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_like();
DROP TRIGGER IF EXISTS trg_feed_galgame_resource_like ON galgame_resource_like;
CREATE TRIGGER trg_feed_galgame_resource_like AFTER INSERT OR UPDATE OR DELETE ON galgame_resource_like
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_resource_like();
DROP TRIGGER IF EXISTS trg_feed_galgame_rating_like ON galgame_rating_like;
CREATE TRIGGER trg_feed_galgame_rating_like AFTER INSERT OR UPDATE OR DELETE ON galgame_rating_like
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_rating_like();

COMMIT;
