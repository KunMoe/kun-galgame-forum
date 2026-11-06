-- Back to 090: the feed holds public topics only. The login rows are deleted
-- (the push enqueue trigger sees each delete; none of them was ever sent).

BEGIN;

DELETE FROM feed_activity WHERE access_scope <> 'public';

CREATE OR REPLACE FUNCTION topic_feed_visible(p_tid integer) RETURNS boolean
LANGUAGE sql STABLE AS $$
    SELECT COALESCE(
        (SELECT status = 0 AND access_scope = 'public' FROM topic WHERE id = p_tid),
        false);
$$;

DROP FUNCTION IF EXISTS feed_upsert(text, integer, integer, integer, text, text, boolean, timestamp with time zone, text);
CREATE FUNCTION feed_upsert(p_type text, p_sid integer, p_uid integer, p_work_id integer, p_content text, p_link text, p_nsfw boolean, p_created timestamp with time zone)
RETURNS void LANGUAGE sql AS $$
    INSERT INTO feed_activity (type, source_id, user_id, work_id, content, link, is_nsfw, created)
    VALUES (p_type, p_sid, p_uid, p_work_id, p_content, p_link, p_nsfw, p_created)
    ON CONFLICT (type, source_id) DO UPDATE SET
        user_id = EXCLUDED.user_id, work_id = EXCLUDED.work_id,
        content = EXCLUDED.content, link = EXCLUDED.link,
        is_nsfw = EXCLUDED.is_nsfw, created = EXCLUDED.created;
$$;

CREATE OR REPLACE FUNCTION public.feed_sync_topic() RETURNS trigger
LANGUAGE plpgsql AS $function$
DECLARE v_visible boolean; v_was boolean;
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('TOPIC_CREATION', OLD.id); RETURN OLD; END IF;
    v_visible := NEW.status = 0 AND NEW.access_scope = 'public';
    IF v_visible THEN
        PERFORM feed_upsert('TOPIC_CREATION', NEW.id, NEW.user_id, 0, NEW.title, '/topic/' || NEW.id, NEW.is_nsfw, NEW.created);
    ELSE
        PERFORM feed_delete('TOPIC_CREATION', NEW.id);
    END IF;
    IF TG_OP = 'UPDATE' THEN
        v_was := OLD.status = 0 AND OLD.access_scope = 'public';
        IF v_was AND NOT v_visible THEN
            DELETE FROM feed_activity fa USING topic_reply r
              WHERE fa.type = 'TOPIC_REPLY_CREATION' AND fa.source_id = r.id AND r.topic_id = NEW.id;
            DELETE FROM feed_activity fa USING topic_comment c
              WHERE fa.type = 'TOPIC_COMMENT_CREATION' AND fa.source_id = c.id AND c.topic_id = NEW.id;
            DELETE FROM feed_activity fa USING topic_upvote u
              WHERE fa.type = 'TOPIC_UPVOTE' AND fa.source_id = u.id AND u.topic_id = NEW.id;
            DELETE FROM feed_activity
              WHERE type IN ('MESSAGE_UPVOTE', 'MESSAGE_SOLUTION')
                AND (link = '/topic/' || NEW.id OR link LIKE '/topic/' || NEW.id || '?%');
        ELSIF NOT v_was AND v_visible THEN
            PERFORM feed_upsert('TOPIC_REPLY_CREATION', r.id, r.user_id, 0, COALESCE(r.content, ''), '/topic/' || NEW.id, NEW.is_nsfw, r.created)
              FROM topic_reply r WHERE r.topic_id = NEW.id AND r.status = 0;
            PERFORM feed_upsert('TOPIC_COMMENT_CREATION', c.id, c.user_id, 0, SUBSTRING(c.content, 1, 100), '/topic/' || NEW.id, NEW.is_nsfw, c.created)
              FROM topic_comment c WHERE c.topic_id = NEW.id AND c.status = 0;
            PERFORM feed_upsert('TOPIC_UPVOTE', u.id, u.user_id, 0, COALESCE(u.description, ''), '/topic/' || NEW.id, NEW.is_nsfw, u.created)
              FROM topic_upvote u WHERE u.topic_id = NEW.id;
            PERFORM feed_upsert(CASE m.type WHEN 'upvoted' THEN 'MESSAGE_UPVOTE' ELSE 'MESSAGE_SOLUTION' END,
                                m.id, m.sender_id, 0, m.content, m.link, NEW.is_nsfw, m.created)
              FROM message m
              WHERE m.type IN ('upvoted', 'solution')
                AND (m.link = '/topic/' || NEW.id OR m.link LIKE '/topic/' || NEW.id || '?%');
        END IF;
        IF NEW.is_nsfw IS DISTINCT FROM OLD.is_nsfw THEN
            UPDATE feed_activity fa SET is_nsfw = NEW.is_nsfw
              FROM topic_reply r   WHERE fa.type = 'TOPIC_REPLY_CREATION'   AND fa.source_id = r.id AND r.topic_id = NEW.id;
            UPDATE feed_activity fa SET is_nsfw = NEW.is_nsfw
              FROM topic_comment c WHERE fa.type = 'TOPIC_COMMENT_CREATION' AND fa.source_id = c.id AND c.topic_id = NEW.id;
            UPDATE feed_activity fa SET is_nsfw = NEW.is_nsfw
              FROM topic_upvote u  WHERE fa.type = 'TOPIC_UPVOTE'           AND fa.source_id = u.id AND u.topic_id = NEW.id;
            UPDATE feed_activity fa SET is_nsfw = NEW.is_nsfw
              WHERE fa.type IN ('MESSAGE_UPVOTE', 'MESSAGE_SOLUTION') AND fa.link = '/topic/' || NEW.id;
        END IF;
    END IF;
    RETURN NEW;
END $function$;

CREATE OR REPLACE FUNCTION public.feed_sync_topic_reply() RETURNS trigger
LANGUAGE plpgsql AS $function$
DECLARE v_nsfw boolean;
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('TOPIC_REPLY_CREATION', OLD.id); RETURN OLD; END IF;
    IF NEW.status <> 0 OR NOT topic_feed_visible(NEW.topic_id) THEN
        PERFORM feed_delete('TOPIC_REPLY_CREATION', NEW.id); RETURN NEW;
    END IF;
    SELECT is_nsfw INTO v_nsfw FROM topic WHERE id = NEW.topic_id;
    PERFORM feed_upsert('TOPIC_REPLY_CREATION', NEW.id, NEW.user_id, 0, COALESCE(NEW.content, ''), '/topic/' || NEW.topic_id, COALESCE(v_nsfw, false), NEW.created);
    RETURN NEW;
END $function$;

CREATE OR REPLACE FUNCTION public.feed_sync_topic_comment() RETURNS trigger
LANGUAGE plpgsql AS $function$
DECLARE v_nsfw boolean;
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('TOPIC_COMMENT_CREATION', OLD.id); RETURN OLD; END IF;
    IF NEW.status <> 0 OR NOT topic_feed_visible(NEW.topic_id) THEN
        PERFORM feed_delete('TOPIC_COMMENT_CREATION', NEW.id); RETURN NEW;
    END IF;
    SELECT is_nsfw INTO v_nsfw FROM topic WHERE id = NEW.topic_id;
    PERFORM feed_upsert('TOPIC_COMMENT_CREATION', NEW.id, NEW.user_id, 0, SUBSTRING(NEW.content, 1, 100), '/topic/' || NEW.topic_id, COALESCE(v_nsfw, false), NEW.created);
    RETURN NEW;
END $function$;

CREATE OR REPLACE FUNCTION public.feed_sync_topic_upvote() RETURNS trigger
LANGUAGE plpgsql AS $function$
DECLARE v_nsfw boolean;
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('TOPIC_UPVOTE', OLD.id); RETURN OLD; END IF;
    IF NOT topic_feed_visible(NEW.topic_id) THEN
        PERFORM feed_delete('TOPIC_UPVOTE', NEW.id); RETURN NEW;
    END IF;
    SELECT is_nsfw INTO v_nsfw FROM topic WHERE id = NEW.topic_id;
    PERFORM feed_upsert('TOPIC_UPVOTE', NEW.id, NEW.user_id, 0, COALESCE(NEW.description, ''), '/topic/' || NEW.topic_id, COALESCE(v_nsfw, false), NEW.created);
    RETURN NEW;
END $function$;

CREATE OR REPLACE FUNCTION public.feed_sync_message() RETURNS trigger
LANGUAGE plpgsql AS $function$
DECLARE v_type text; v_tid int; v_nsfw boolean;
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM feed_delete('MESSAGE_UPVOTE', OLD.id);
        PERFORM feed_delete('MESSAGE_SOLUTION', OLD.id);
        RETURN OLD;
    END IF;
    v_type := CASE NEW.type WHEN 'upvoted' THEN 'MESSAGE_UPVOTE' WHEN 'solution' THEN 'MESSAGE_SOLUTION' ELSE NULL END;
    IF v_type IS NOT NULL THEN
        v_tid := (regexp_match(NEW.link, '^/topic/([0-9]+)'))[1]::int;
        IF v_tid IS NOT NULL AND NOT topic_feed_visible(v_tid) THEN
            PERFORM feed_delete(v_type, NEW.id); RETURN NEW;
        END IF;
        IF v_tid IS NOT NULL THEN SELECT is_nsfw INTO v_nsfw FROM topic WHERE id = v_tid; END IF;
        PERFORM feed_upsert(v_type, NEW.id, NEW.sender_id, 0, NEW.content, NEW.link, COALESCE(v_nsfw, false), NEW.created);
    END IF;
    RETURN NEW;
END $function$;

DROP FUNCTION IF EXISTS topic_feed_audience(integer);
DROP FUNCTION IF EXISTS topic_feed_scope(integer, text);

COMMIT;
