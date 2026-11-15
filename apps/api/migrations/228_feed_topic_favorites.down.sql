-- Removes the favorite trigger and the favorite rows 228 put in feed_activity,
-- and restores 226's feed_rebuild_topic_likes. Each deleted row enqueues
-- itself, so the pusher withdraws what it sent.
BEGIN;

DROP TRIGGER IF EXISTS trg_feed_topic_favorite ON topic_favorite;
DROP FUNCTION IF EXISTS feed_sync_topic_favorite();

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

DELETE FROM feed_activity WHERE type = 'TOPIC_FAVORITE';

COMMIT;
