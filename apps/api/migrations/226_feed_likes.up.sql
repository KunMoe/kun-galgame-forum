-- 226: likes enter feed_activity, so the pusher sends them to community and a
-- user's timeline (listUserActivities) shows them, grouped per day there.
--
-- Six push-only types, one per live like table: TOPIC_LIKE (topic_reaction,
-- reaction 'like'), TOPIC_REPLY_LIKE (topic_reply_reaction, 'like'),
-- TOPIC_COMMENT_LIKE (topic_comment_like), GALGAME_LIKE (galgame_like),
-- GALGAME_RESOURCE_LIKE (galgame_resource_like), GALGAME_RATING_LIKE
-- (galgame_rating_like). listActivities names its types explicitly, so these
-- rows never reach the site feed; activityapiv1 knows them only to assemble a
-- push.
--
-- Topic likes follow their topic exactly like TOPIC_UPVOTE does (223): a row
-- exists only while the topic is published with scope public or login, and
-- carries that scope; the pusher sends public rows only. The topic, reply and
-- comment triggers of 223 are not touched. Separate AFTER UPDATE triggers
-- rebuild a topic's, reply's or comment's like rows whenever its status,
-- access_scope or is_nsfw actually changes: delete them, then upsert the ones
-- still visible from the live tables. Without that a like on a topic that went
-- login-only would stay public in community until the daily reconcile.
--
-- Work likes carry work_id and is_nsfw = false like every other work-tied row;
-- whether the work renders is decided at read time (workRendered), and a fold
-- repoints them with the rest of the work's rows.
--
-- The rows of existing likes are built at the bottom. Their queue entries are
-- not notifications: the pusher notifies only publish verbs.
--
-- Additive. Deploy first, then run this: the build before it acks queue rows
-- of a type it does not push, and those would wait for the next 04:30
-- reconcile instead of going out at once.
BEGIN;

CREATE OR REPLACE FUNCTION feed_sync_topic_reaction() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE v_scope text; v_nsfw boolean;
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('TOPIC_LIKE', OLD.id); RETURN OLD; END IF;
    SELECT scope, nsfw INTO v_scope, v_nsfw FROM topic_feed_audience(NEW.topic_id);
    IF NEW.reaction <> 'like' OR v_scope IS NULL THEN
        PERFORM feed_delete('TOPIC_LIKE', NEW.id); RETURN NEW;
    END IF;
    PERFORM feed_upsert('TOPIC_LIKE', NEW.id, NEW.user_id, 0, '', '/topic/' || NEW.topic_id, v_nsfw, NEW.created, v_scope);
    RETURN NEW;
END $$;

CREATE OR REPLACE FUNCTION feed_sync_topic_reply_reaction() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE v_tid int; v_status int; v_scope text; v_nsfw boolean;
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('TOPIC_REPLY_LIKE', OLD.id); RETURN OLD; END IF;
    SELECT topic_id, status INTO v_tid, v_status FROM topic_reply WHERE id = NEW.topic_reply_id;
    SELECT scope, nsfw INTO v_scope, v_nsfw FROM topic_feed_audience(v_tid);
    IF NEW.reaction <> 'like' OR v_status IS DISTINCT FROM 0 OR v_scope IS NULL THEN
        PERFORM feed_delete('TOPIC_REPLY_LIKE', NEW.id); RETURN NEW;
    END IF;
    PERFORM feed_upsert('TOPIC_REPLY_LIKE', NEW.id, NEW.user_id, 0, '', '/topic/' || v_tid, v_nsfw, NEW.created, v_scope);
    RETURN NEW;
END $$;

CREATE OR REPLACE FUNCTION feed_sync_topic_comment_like() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE v_tid int; v_status int; v_scope text; v_nsfw boolean;
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('TOPIC_COMMENT_LIKE', OLD.id); RETURN OLD; END IF;
    SELECT topic_id, status INTO v_tid, v_status FROM topic_comment WHERE id = NEW.topic_comment_id;
    SELECT scope, nsfw INTO v_scope, v_nsfw FROM topic_feed_audience(v_tid);
    IF v_status IS DISTINCT FROM 0 OR v_scope IS NULL THEN
        PERFORM feed_delete('TOPIC_COMMENT_LIKE', NEW.id); RETURN NEW;
    END IF;
    PERFORM feed_upsert('TOPIC_COMMENT_LIKE', NEW.id, NEW.user_id, 0, '', '/topic/' || v_tid, v_nsfw, NEW.created, v_scope);
    RETURN NEW;
END $$;

CREATE OR REPLACE FUNCTION feed_sync_galgame_like() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('GALGAME_LIKE', OLD.id); RETURN OLD; END IF;
    PERFORM feed_upsert('GALGAME_LIKE', NEW.id, NEW.user_id, NEW.work_id, '', '/galgame/' || NEW.work_id, false, NEW.created);
    RETURN NEW;
END $$;

CREATE OR REPLACE FUNCTION feed_sync_galgame_resource_like() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE v_work int;
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('GALGAME_RESOURCE_LIKE', OLD.id); RETURN OLD; END IF;
    SELECT work_id INTO v_work FROM galgame_resource WHERE id = NEW.galgame_resource_id;
    IF v_work IS NULL THEN
        PERFORM feed_delete('GALGAME_RESOURCE_LIKE', NEW.id); RETURN NEW;
    END IF;
    PERFORM feed_upsert('GALGAME_RESOURCE_LIKE', NEW.id, NEW.user_id, v_work, '', '/galgame/' || v_work, false, NEW.created);
    RETURN NEW;
END $$;

CREATE OR REPLACE FUNCTION feed_sync_galgame_rating_like() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE v_work int;
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('GALGAME_RATING_LIKE', OLD.id); RETURN OLD; END IF;
    SELECT work_id INTO v_work FROM galgame_rating WHERE id = NEW.galgame_rating_id;
    IF v_work IS NULL THEN
        PERFORM feed_delete('GALGAME_RATING_LIKE', NEW.id); RETURN NEW;
    END IF;
    PERFORM feed_upsert('GALGAME_RATING_LIKE', NEW.id, NEW.user_id, v_work, '', '/galgame/' || v_work, false, NEW.created);
    RETURN NEW;
END $$;

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

-- The like rows of one topic, rebuilt from the live tables.
CREATE OR REPLACE FUNCTION feed_rebuild_topic_likes(p_tid int) RETURNS void
LANGUAGE plpgsql AS $$
DECLARE v_scope text; v_nsfw boolean;
BEGIN
    DELETE FROM feed_activity fa USING topic_reaction x
      WHERE fa.type = 'TOPIC_LIKE' AND fa.source_id = x.id AND x.topic_id = p_tid;
    DELETE FROM feed_activity fa USING topic_reply_reaction x, topic_reply r
      WHERE fa.type = 'TOPIC_REPLY_LIKE' AND fa.source_id = x.id AND r.id = x.topic_reply_id AND r.topic_id = p_tid;
    DELETE FROM feed_activity fa USING topic_comment_like x, topic_comment c
      WHERE fa.type = 'TOPIC_COMMENT_LIKE' AND fa.source_id = x.id AND c.id = x.topic_comment_id AND c.topic_id = p_tid;
    SELECT scope, nsfw INTO v_scope, v_nsfw FROM topic_feed_audience(p_tid);
    IF v_scope IS NULL THEN RETURN; END IF;
    PERFORM feed_upsert('TOPIC_LIKE', x.id, x.user_id, 0, '', '/topic/' || p_tid, v_nsfw, x.created, v_scope)
      FROM topic_reaction x WHERE x.topic_id = p_tid AND x.reaction = 'like';
    PERFORM feed_upsert('TOPIC_REPLY_LIKE', x.id, x.user_id, 0, '', '/topic/' || p_tid, v_nsfw, x.created, v_scope)
      FROM topic_reply_reaction x JOIN topic_reply r ON r.id = x.topic_reply_id
      WHERE r.topic_id = p_tid AND r.status = 0 AND x.reaction = 'like';
    PERFORM feed_upsert('TOPIC_COMMENT_LIKE', x.id, x.user_id, 0, '', '/topic/' || p_tid, v_nsfw, x.created, v_scope)
      FROM topic_comment_like x JOIN topic_comment c ON c.id = x.topic_comment_id
      WHERE c.topic_id = p_tid AND c.status = 0;
END $$;

CREATE OR REPLACE FUNCTION feed_sync_topic_likes() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM feed_rebuild_topic_likes(NEW.id);
    RETURN NEW;
END $$;

CREATE OR REPLACE FUNCTION feed_sync_topic_reply_likes() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE v_scope text; v_nsfw boolean;
BEGIN
    DELETE FROM feed_activity fa USING topic_reply_reaction x
      WHERE fa.type = 'TOPIC_REPLY_LIKE' AND fa.source_id = x.id AND x.topic_reply_id = NEW.id;
    SELECT scope, nsfw INTO v_scope, v_nsfw FROM topic_feed_audience(NEW.topic_id);
    IF NEW.status <> 0 OR v_scope IS NULL THEN RETURN NEW; END IF;
    PERFORM feed_upsert('TOPIC_REPLY_LIKE', x.id, x.user_id, 0, '', '/topic/' || NEW.topic_id, v_nsfw, x.created, v_scope)
      FROM topic_reply_reaction x WHERE x.topic_reply_id = NEW.id AND x.reaction = 'like';
    RETURN NEW;
END $$;

CREATE OR REPLACE FUNCTION feed_sync_topic_comment_likes() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE v_scope text; v_nsfw boolean;
BEGIN
    DELETE FROM feed_activity fa USING topic_comment_like x
      WHERE fa.type = 'TOPIC_COMMENT_LIKE' AND fa.source_id = x.id AND x.topic_comment_id = NEW.id;
    SELECT scope, nsfw INTO v_scope, v_nsfw FROM topic_feed_audience(NEW.topic_id);
    IF NEW.status <> 0 OR v_scope IS NULL THEN RETURN NEW; END IF;
    PERFORM feed_upsert('TOPIC_COMMENT_LIKE', x.id, x.user_id, 0, '', '/topic/' || NEW.topic_id, v_nsfw, x.created, v_scope)
      FROM topic_comment_like x WHERE x.topic_comment_id = NEW.id;
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS trg_feed_topic_likes ON topic;
CREATE TRIGGER trg_feed_topic_likes AFTER UPDATE OF status, access_scope, is_nsfw ON topic
    FOR EACH ROW
    WHEN (OLD.status IS DISTINCT FROM NEW.status OR OLD.access_scope IS DISTINCT FROM NEW.access_scope
          OR OLD.is_nsfw IS DISTINCT FROM NEW.is_nsfw)
    EXECUTE FUNCTION feed_sync_topic_likes();
DROP TRIGGER IF EXISTS trg_feed_topic_reply_likes ON topic_reply;
CREATE TRIGGER trg_feed_topic_reply_likes AFTER UPDATE OF status ON topic_reply
    FOR EACH ROW WHEN (OLD.status IS DISTINCT FROM NEW.status)
    EXECUTE FUNCTION feed_sync_topic_reply_likes();
DROP TRIGGER IF EXISTS trg_feed_topic_comment_likes ON topic_comment;
CREATE TRIGGER trg_feed_topic_comment_likes AFTER UPDATE OF status ON topic_comment
    FOR EACH ROW WHEN (OLD.status IS DISTINCT FROM NEW.status)
    EXECUTE FUNCTION feed_sync_topic_comment_likes();

SELECT feed_rebuild_topic_likes(t.id) FROM topic t
  WHERE EXISTS (SELECT 1 FROM topic_reaction x WHERE x.topic_id = t.id AND x.reaction = 'like')
     OR EXISTS (SELECT 1 FROM topic_reply r JOIN topic_reply_reaction x ON x.topic_reply_id = r.id WHERE r.topic_id = t.id AND x.reaction = 'like')
     OR EXISTS (SELECT 1 FROM topic_comment c JOIN topic_comment_like x ON x.topic_comment_id = c.id WHERE c.topic_id = t.id);
SELECT feed_upsert('GALGAME_LIKE', x.id, x.user_id, x.work_id, '', '/galgame/' || x.work_id, false, x.created)
  FROM galgame_like x;
SELECT feed_upsert('GALGAME_RESOURCE_LIKE', x.id, x.user_id, r.work_id, '', '/galgame/' || r.work_id, false, x.created)
  FROM galgame_resource_like x JOIN galgame_resource r ON r.id = x.galgame_resource_id;
SELECT feed_upsert('GALGAME_RATING_LIKE', x.id, x.user_id, g.work_id, '', '/galgame/' || g.work_id, false, x.created)
  FROM galgame_rating_like x JOIN galgame_rating g ON g.id = x.galgame_rating_id;

COMMIT;
