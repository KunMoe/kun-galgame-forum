-- 228: topic favorites enter feed_activity as the push-only type
-- TOPIC_FAVORITE, so the pusher sends them to community as verb favorite
-- (object_ref topic:<id>) and a user's timeline shows them. Community accepts
-- the verb since its spec 1.3.0 (2026-11-15).
--
-- A favorite follows its topic exactly like a topic like does (226): a row
-- exists only while the topic is published with scope public or login and
-- carries that scope. feed_rebuild_topic_likes, which 226's topic trigger runs
-- whenever a topic's status, access_scope or is_nsfw changes, is redefined to
-- rebuild favorite rows too; everything else in it is unchanged.
--
-- Unfavoriting deletes the topic_favorite row, which deletes the feed row and
-- enqueues the tombstone. The rows of existing favorites are built at the
-- bottom with each favorite's own time; their queue entries are not
-- notifications (the pusher notifies only publish verbs).
--
-- Additive; the deploy's migrate step runs it before the new API starts. The
-- old API is still draining during that window and acks TOPIC_FAVORITE queue
-- rows it does not push; the 04:30 reconcile sends those.
BEGIN;

CREATE OR REPLACE FUNCTION feed_sync_topic_favorite() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE v_scope text; v_nsfw boolean;
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('TOPIC_FAVORITE', OLD.id); RETURN OLD; END IF;
    SELECT scope, nsfw INTO v_scope, v_nsfw FROM topic_feed_audience(NEW.topic_id);
    IF v_scope IS NULL THEN
        PERFORM feed_delete('TOPIC_FAVORITE', NEW.id); RETURN NEW;
    END IF;
    PERFORM feed_upsert('TOPIC_FAVORITE', NEW.id, NEW.user_id, 0, '', '/topic/' || NEW.topic_id, v_nsfw, NEW.created, v_scope);
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS trg_feed_topic_favorite ON topic_favorite;
CREATE TRIGGER trg_feed_topic_favorite AFTER INSERT OR UPDATE OF topic_id, user_id, created OR DELETE ON topic_favorite
    FOR EACH ROW EXECUTE FUNCTION feed_sync_topic_favorite();

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
    DELETE FROM feed_activity fa USING topic_favorite x
      WHERE fa.type = 'TOPIC_FAVORITE' AND fa.source_id = x.id AND x.topic_id = p_tid;
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
    PERFORM feed_upsert('TOPIC_FAVORITE', x.id, x.user_id, 0, '', '/topic/' || p_tid, v_nsfw, x.created, v_scope)
      FROM topic_favorite x WHERE x.topic_id = p_tid;
END $$;

SELECT feed_upsert('TOPIC_FAVORITE', x.id, x.user_id, 0, '', '/topic/' || x.topic_id, a.nsfw, x.created, a.scope)
  FROM topic_favorite x
  CROSS JOIN LATERAL topic_feed_audience(x.topic_id) a
  WHERE a.scope IS NOT NULL;

COMMIT;
