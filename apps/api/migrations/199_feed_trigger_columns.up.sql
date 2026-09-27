-- 199: the feed triggers on galgame, galgame_activity, galgame_rating,
-- message, todo, topic, topic_comment, topic_reply, topic_upvote and
-- update_log fired on every UPDATE. A view, a like, a count recompute, a read
-- notification and the catalog mirror poll each rewrote the feed_activity row
-- with the values it already held, and since 198 every rewrite also queued the
-- row for the activity pusher, which sent it to community again under a new
-- revision. On 2026-09-27 community logged 9-16 unchanged kungal writes every
-- 5 s (galgame 953, galgame_rating 55, topic 37 keys in ten minutes), rising
-- with traffic; pg_stat counted 5.92M updates on galgame and 8.18M on
-- feed_activity.
--
-- As 145 did for galgame_resource, each trigger now fires on UPDATE only when
-- a listed column is written: the columns its function reads, plus the columns
-- of the same row that the pusher sends and the feed row does not hold
-- (galgame.creator_user_id is the actor of GALGAME_CREATION; topic.content and
-- topic.cover_images are the excerpt and cover of TOPIC_CREATION). The
-- functions are unchanged. feed_upsert still rewrites unconditionally, so an
-- edit to one of those columns reaches community at once. A WHERE guard on
-- feed_upsert was the other fix, but it would have held those edits back
-- until the nightly reconcile.
--
-- No rows change, and every statement is drop-and-recreate, so running it
-- again changes nothing.

DROP TRIGGER IF EXISTS trg_feed_galgame ON galgame;
CREATE TRIGGER trg_feed_galgame
    AFTER INSERT OR DELETE OR UPDATE OF published, creator_user_id, created
    ON galgame
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame();

DROP TRIGGER IF EXISTS trg_feed_galgame_activity ON galgame_activity;
CREATE TRIGGER trg_feed_galgame_activity
    AFTER INSERT OR DELETE OR UPDATE OF type, user_id, work_id, created
    ON galgame_activity
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_activity();

DROP TRIGGER IF EXISTS trg_feed_galgame_rating ON galgame_rating;
CREATE TRIGGER trg_feed_galgame_rating
    AFTER INSERT OR DELETE OR UPDATE OF user_id, work_id, short_summary, spoiler_level, created
    ON galgame_rating
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_rating();

DROP TRIGGER IF EXISTS trg_feed_message ON message;
CREATE TRIGGER trg_feed_message
    AFTER INSERT OR DELETE OR UPDATE OF type, link, sender_id, content, created
    ON message
    FOR EACH ROW EXECUTE FUNCTION feed_sync_message();

DROP TRIGGER IF EXISTS trg_feed_todo ON todo;
CREATE TRIGGER trg_feed_todo
    AFTER INSERT OR DELETE OR UPDATE OF user_id, content, created
    ON todo
    FOR EACH ROW EXECUTE FUNCTION feed_sync_todo();

DROP TRIGGER IF EXISTS trg_feed_topic ON topic;
CREATE TRIGGER trg_feed_topic
    AFTER INSERT OR DELETE OR UPDATE OF title, content, cover_images, status, access_scope, is_nsfw, user_id, created
    ON topic
    FOR EACH ROW EXECUTE FUNCTION feed_sync_topic();

DROP TRIGGER IF EXISTS trg_feed_topic_comment ON topic_comment;
CREATE TRIGGER trg_feed_topic_comment
    AFTER INSERT OR DELETE OR UPDATE OF status, topic_id, user_id, content, created
    ON topic_comment
    FOR EACH ROW EXECUTE FUNCTION feed_sync_topic_comment();

DROP TRIGGER IF EXISTS trg_feed_topic_reply ON topic_reply;
CREATE TRIGGER trg_feed_topic_reply
    AFTER INSERT OR DELETE OR UPDATE OF status, topic_id, user_id, content, created
    ON topic_reply
    FOR EACH ROW EXECUTE FUNCTION feed_sync_topic_reply();

DROP TRIGGER IF EXISTS trg_feed_topic_upvote ON topic_upvote;
CREATE TRIGGER trg_feed_topic_upvote
    AFTER INSERT OR DELETE OR UPDATE OF topic_id, user_id, description, created
    ON topic_upvote
    FOR EACH ROW EXECUTE FUNCTION feed_sync_topic_upvote();

DROP TRIGGER IF EXISTS trg_feed_update_log ON update_log;
CREATE TRIGGER trg_feed_update_log
    AFTER INSERT OR DELETE OR UPDATE OF user_id, content, created
    ON update_log
    FOR EACH ROW EXECUTE FUNCTION feed_sync_update_log();
