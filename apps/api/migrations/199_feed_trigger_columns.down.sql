DROP TRIGGER IF EXISTS trg_feed_galgame ON galgame;
CREATE TRIGGER trg_feed_galgame
    AFTER INSERT OR DELETE OR UPDATE
    ON galgame
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame();

DROP TRIGGER IF EXISTS trg_feed_galgame_activity ON galgame_activity;
CREATE TRIGGER trg_feed_galgame_activity
    AFTER INSERT OR DELETE OR UPDATE
    ON galgame_activity
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_activity();

DROP TRIGGER IF EXISTS trg_feed_galgame_rating ON galgame_rating;
CREATE TRIGGER trg_feed_galgame_rating
    AFTER INSERT OR DELETE OR UPDATE
    ON galgame_rating
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_rating();

DROP TRIGGER IF EXISTS trg_feed_message ON message;
CREATE TRIGGER trg_feed_message
    AFTER INSERT OR DELETE OR UPDATE
    ON message
    FOR EACH ROW EXECUTE FUNCTION feed_sync_message();

DROP TRIGGER IF EXISTS trg_feed_todo ON todo;
CREATE TRIGGER trg_feed_todo
    AFTER INSERT OR DELETE OR UPDATE
    ON todo
    FOR EACH ROW EXECUTE FUNCTION feed_sync_todo();

DROP TRIGGER IF EXISTS trg_feed_topic ON topic;
CREATE TRIGGER trg_feed_topic
    AFTER INSERT OR DELETE OR UPDATE
    ON topic
    FOR EACH ROW EXECUTE FUNCTION feed_sync_topic();

DROP TRIGGER IF EXISTS trg_feed_topic_comment ON topic_comment;
CREATE TRIGGER trg_feed_topic_comment
    AFTER INSERT OR DELETE OR UPDATE
    ON topic_comment
    FOR EACH ROW EXECUTE FUNCTION feed_sync_topic_comment();

DROP TRIGGER IF EXISTS trg_feed_topic_reply ON topic_reply;
CREATE TRIGGER trg_feed_topic_reply
    AFTER INSERT OR DELETE OR UPDATE
    ON topic_reply
    FOR EACH ROW EXECUTE FUNCTION feed_sync_topic_reply();

DROP TRIGGER IF EXISTS trg_feed_topic_upvote ON topic_upvote;
CREATE TRIGGER trg_feed_topic_upvote
    AFTER INSERT OR DELETE OR UPDATE
    ON topic_upvote
    FOR EACH ROW EXECUTE FUNCTION feed_sync_topic_upvote();

DROP TRIGGER IF EXISTS trg_feed_update_log ON update_log;
CREATE TRIGGER trg_feed_update_log
    AFTER INSERT OR DELETE OR UPDATE
    ON update_log
    FOR EACH ROW EXECUTE FUNCTION feed_sync_update_log();
