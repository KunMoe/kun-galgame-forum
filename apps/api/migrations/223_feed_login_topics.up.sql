-- 223: login-scoped topics enter the activity feed for signed-in readers.
--
-- Until now every activity of a topic whose access_scope was not 'public' was
-- deleted from feed_activity (migration 090), so a topic switched to 'login'
-- vanished from the home feed for everyone, its author included. The triggers
-- now decide only whether a row exists (published topic, scope public or
-- login) and stamp the topic's scope on feed_activity.access_scope (222); who
-- may read a row is decided at read time by
-- topic/repository.SharedListPredicate, the predicate every topic list already
-- uses. 'role' and 'users' topics stay out of the feed.
--
-- feed_activity rows are also pushed to NextMoe community, whose following
-- feed and kind-10 notifications reach every site. The pusher reads only
-- access_scope = 'public' rows, so a login row is never sent, and a row that
-- turns login is tombstoned there by the drainer: the scope UPDATE fires
-- trg_activity_push_enqueue like any other change.
--
-- DEPLOY ORDER: held back by cmd/migrate's default -exclude and run by hand
-- with --only=223 AFTER the API that filters on access_scope is live. The
-- binary before it reads feed_activity unfiltered: it would show the rows
-- rebuilt below to anonymous readers and push them to community, notifying
-- followers, during the seconds between this commit and its own shutdown —
-- and its pusher drains every 5 s.
--
-- Existing rows stay as 222 left them (public). The activities of published
-- login topics, deleted by 090 and by every scope switch since, are rebuilt
-- from the live tables at the bottom.

BEGIN;

-- Callers that pass eight arguments (every non-topic trigger and the wall
-- store in Go) keep writing 'public'.
DROP FUNCTION IF EXISTS feed_upsert(text, integer, integer, integer, text, text, boolean, timestamp with time zone);
DROP FUNCTION IF EXISTS feed_upsert(text, integer, integer, integer, text, text, boolean, timestamp with time zone, text);
CREATE FUNCTION feed_upsert(p_type text, p_sid integer, p_uid integer, p_work_id integer, p_content text, p_link text,
                            p_nsfw boolean, p_created timestamp with time zone, p_scope text DEFAULT 'public')
RETURNS void LANGUAGE sql AS $$
    INSERT INTO feed_activity (type, source_id, user_id, work_id, content, link, is_nsfw, created, access_scope)
    VALUES (p_type, p_sid, p_uid, p_work_id, p_content, p_link, p_nsfw, p_created, p_scope)
    ON CONFLICT (type, source_id) DO UPDATE SET
        user_id = EXCLUDED.user_id, work_id = EXCLUDED.work_id,
        content = EXCLUDED.content, link = EXCLUDED.link,
        is_nsfw = EXCLUDED.is_nsfw, created = EXCLUDED.created,
        access_scope = EXCLUDED.access_scope;
$$;

-- The scope a topic's activities carry in the feed; NULL means the feed holds
-- none of them. Readers decide who sees which scope.
CREATE OR REPLACE FUNCTION topic_feed_scope(p_status integer, p_scope text) RETURNS text
LANGUAGE sql IMMUTABLE AS $$
    SELECT CASE WHEN p_status = 0 AND p_scope IN ('public', 'login') THEN p_scope END;
$$;

CREATE OR REPLACE FUNCTION topic_feed_audience(p_tid integer, OUT scope text, OUT nsfw boolean)
LANGUAGE sql STABLE AS $$
    SELECT topic_feed_scope(status, access_scope), is_nsfw FROM topic WHERE id = p_tid;
$$;

CREATE OR REPLACE FUNCTION public.feed_sync_topic() RETURNS trigger
LANGUAGE plpgsql AS $function$
DECLARE v_scope text; v_was text;
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('TOPIC_CREATION', OLD.id); RETURN OLD; END IF;
    v_scope := topic_feed_scope(NEW.status, NEW.access_scope);
    IF v_scope IS NOT NULL THEN
        PERFORM feed_upsert('TOPIC_CREATION', NEW.id, NEW.user_id, 0, NEW.title, '/topic/' || NEW.id, NEW.is_nsfw, NEW.created, v_scope);
    ELSE
        PERFORM feed_delete('TOPIC_CREATION', NEW.id);
    END IF;
    IF TG_OP = 'UPDATE' THEN
        v_was := topic_feed_scope(OLD.status, OLD.access_scope);
        IF v_was IS NOT NULL AND v_scope IS NULL THEN
            -- Children leave the feed with the topic. Message rows are keyed
            -- by link: exact or '?...' only — LIKE '/topic/<id>%' would also
            -- match topic 41209 when topic 4120 goes dark.
            DELETE FROM feed_activity fa USING topic_reply r
              WHERE fa.type = 'TOPIC_REPLY_CREATION' AND fa.source_id = r.id AND r.topic_id = NEW.id;
            DELETE FROM feed_activity fa USING topic_comment c
              WHERE fa.type = 'TOPIC_COMMENT_CREATION' AND fa.source_id = c.id AND c.topic_id = NEW.id;
            DELETE FROM feed_activity fa USING topic_upvote u
              WHERE fa.type = 'TOPIC_UPVOTE' AND fa.source_id = u.id AND u.topic_id = NEW.id;
            DELETE FROM feed_activity
              WHERE type IN ('MESSAGE_UPVOTE', 'MESSAGE_SOLUTION')
                AND (link = '/topic/' || NEW.id OR link LIKE '/topic/' || NEW.id || '?%');
        ELSIF v_was IS NULL AND v_scope IS NOT NULL THEN
            -- Children rejoin from the live tables.
            PERFORM feed_upsert('TOPIC_REPLY_CREATION', r.id, r.user_id, 0, COALESCE(r.content, ''), '/topic/' || NEW.id, NEW.is_nsfw, r.created, v_scope)
              FROM topic_reply r WHERE r.topic_id = NEW.id AND r.status = 0;
            PERFORM feed_upsert('TOPIC_COMMENT_CREATION', c.id, c.user_id, 0, SUBSTRING(c.content, 1, 100), '/topic/' || NEW.id, NEW.is_nsfw, c.created, v_scope)
              FROM topic_comment c WHERE c.topic_id = NEW.id AND c.status = 0;
            PERFORM feed_upsert('TOPIC_UPVOTE', u.id, u.user_id, 0, COALESCE(u.description, ''), '/topic/' || NEW.id, NEW.is_nsfw, u.created, v_scope)
              FROM topic_upvote u WHERE u.topic_id = NEW.id;
            PERFORM feed_upsert(CASE m.type WHEN 'upvoted' THEN 'MESSAGE_UPVOTE' ELSE 'MESSAGE_SOLUTION' END,
                                m.id, m.sender_id, 0, m.content, m.link, NEW.is_nsfw, m.created, v_scope)
              FROM message m
              WHERE m.type IN ('upvoted', 'solution')
                AND (m.link = '/topic/' || NEW.id OR m.link LIKE '/topic/' || NEW.id || '?%');
        ELSIF v_was IS DISTINCT FROM v_scope THEN
            UPDATE feed_activity fa SET access_scope = v_scope
              FROM topic_reply r   WHERE fa.type = 'TOPIC_REPLY_CREATION'   AND fa.source_id = r.id AND r.topic_id = NEW.id;
            UPDATE feed_activity fa SET access_scope = v_scope
              FROM topic_comment c WHERE fa.type = 'TOPIC_COMMENT_CREATION' AND fa.source_id = c.id AND c.topic_id = NEW.id;
            UPDATE feed_activity fa SET access_scope = v_scope
              FROM topic_upvote u  WHERE fa.type = 'TOPIC_UPVOTE'           AND fa.source_id = u.id AND u.topic_id = NEW.id;
            UPDATE feed_activity SET access_scope = v_scope
              WHERE type IN ('MESSAGE_UPVOTE', 'MESSAGE_SOLUTION')
                AND (link = '/topic/' || NEW.id OR link LIKE '/topic/' || NEW.id || '?%');
        END IF;
        -- A change to the topic's NSFW flag re-flags everything that hangs off it.
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
DECLARE v_scope text; v_nsfw boolean;
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('TOPIC_REPLY_CREATION', OLD.id); RETURN OLD; END IF;
    SELECT scope, nsfw INTO v_scope, v_nsfw FROM topic_feed_audience(NEW.topic_id);
    IF NEW.status <> 0 OR v_scope IS NULL THEN
        PERFORM feed_delete('TOPIC_REPLY_CREATION', NEW.id); RETURN NEW;
    END IF;
    PERFORM feed_upsert('TOPIC_REPLY_CREATION', NEW.id, NEW.user_id, 0, COALESCE(NEW.content, ''), '/topic/' || NEW.topic_id, v_nsfw, NEW.created, v_scope);
    RETURN NEW;
END $function$;

CREATE OR REPLACE FUNCTION public.feed_sync_topic_comment() RETURNS trigger
LANGUAGE plpgsql AS $function$
DECLARE v_scope text; v_nsfw boolean;
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('TOPIC_COMMENT_CREATION', OLD.id); RETURN OLD; END IF;
    SELECT scope, nsfw INTO v_scope, v_nsfw FROM topic_feed_audience(NEW.topic_id);
    IF NEW.status <> 0 OR v_scope IS NULL THEN
        PERFORM feed_delete('TOPIC_COMMENT_CREATION', NEW.id); RETURN NEW;
    END IF;
    PERFORM feed_upsert('TOPIC_COMMENT_CREATION', NEW.id, NEW.user_id, 0, SUBSTRING(NEW.content, 1, 100), '/topic/' || NEW.topic_id, v_nsfw, NEW.created, v_scope);
    RETURN NEW;
END $function$;

CREATE OR REPLACE FUNCTION public.feed_sync_topic_upvote() RETURNS trigger
LANGUAGE plpgsql AS $function$
DECLARE v_scope text; v_nsfw boolean;
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('TOPIC_UPVOTE', OLD.id); RETURN OLD; END IF;
    SELECT scope, nsfw INTO v_scope, v_nsfw FROM topic_feed_audience(NEW.topic_id);
    IF v_scope IS NULL THEN
        PERFORM feed_delete('TOPIC_UPVOTE', NEW.id); RETURN NEW;
    END IF;
    PERFORM feed_upsert('TOPIC_UPVOTE', NEW.id, NEW.user_id, 0, COALESCE(NEW.description, ''), '/topic/' || NEW.topic_id, v_nsfw, NEW.created, v_scope);
    RETURN NEW;
END $function$;

CREATE OR REPLACE FUNCTION public.feed_sync_message() RETURNS trigger
LANGUAGE plpgsql AS $function$
DECLARE v_type text; v_tid int; v_scope text := 'public'; v_nsfw boolean;
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM feed_delete('MESSAGE_UPVOTE', OLD.id);
        PERFORM feed_delete('MESSAGE_SOLUTION', OLD.id);
        RETURN OLD;
    END IF;
    v_type := CASE NEW.type WHEN 'upvoted' THEN 'MESSAGE_UPVOTE' WHEN 'solution' THEN 'MESSAGE_SOLUTION' ELSE NULL END;
    IF v_type IS NOT NULL THEN
        v_tid := (regexp_match(NEW.link, '^/topic/([0-9]+)'))[1]::int;
        -- Non-topic links (v_tid NULL) keep the old unconditional path.
        IF v_tid IS NOT NULL THEN
            SELECT scope, nsfw INTO v_scope, v_nsfw FROM topic_feed_audience(v_tid);
            IF v_scope IS NULL THEN
                PERFORM feed_delete(v_type, NEW.id); RETURN NEW;
            END IF;
        END IF;
        PERFORM feed_upsert(v_type, NEW.id, NEW.sender_id, 0, NEW.content, NEW.link, COALESCE(v_nsfw, false), NEW.created, v_scope);
    END IF;
    RETURN NEW;
END $function$;

DROP FUNCTION IF EXISTS topic_feed_visible(integer);

-- Rebuild the activities of published login topics, which 090 and every
-- scope switch since deleted.
SELECT feed_upsert('TOPIC_CREATION', t.id, t.user_id, 0, t.title, '/topic/' || t.id, t.is_nsfw, t.created, 'login')
  FROM topic t WHERE t.status = 0 AND t.access_scope = 'login';
SELECT feed_upsert('TOPIC_REPLY_CREATION', r.id, r.user_id, 0, COALESCE(r.content, ''), '/topic/' || t.id, t.is_nsfw, r.created, 'login')
  FROM topic_reply r JOIN topic t ON t.id = r.topic_id
  WHERE t.status = 0 AND t.access_scope = 'login' AND r.status = 0;
SELECT feed_upsert('TOPIC_COMMENT_CREATION', c.id, c.user_id, 0, SUBSTRING(c.content, 1, 100), '/topic/' || t.id, t.is_nsfw, c.created, 'login')
  FROM topic_comment c JOIN topic t ON t.id = c.topic_id
  WHERE t.status = 0 AND t.access_scope = 'login' AND c.status = 0;
SELECT feed_upsert('TOPIC_UPVOTE', u.id, u.user_id, 0, COALESCE(u.description, ''), '/topic/' || t.id, t.is_nsfw, u.created, 'login')
  FROM topic_upvote u JOIN topic t ON t.id = u.topic_id
  WHERE t.status = 0 AND t.access_scope = 'login';
SELECT feed_upsert(CASE m.type WHEN 'upvoted' THEN 'MESSAGE_UPVOTE' ELSE 'MESSAGE_SOLUTION' END,
                   m.id, m.sender_id, 0, m.content, m.link, t.is_nsfw, m.created, 'login')
  FROM message m JOIN topic t ON m.link = '/topic/' || t.id OR m.link LIKE '/topic/' || t.id || '?%'
  WHERE m.type IN ('upvoted', 'solution') AND t.status = 0 AND t.access_scope = 'login';

COMMIT;
