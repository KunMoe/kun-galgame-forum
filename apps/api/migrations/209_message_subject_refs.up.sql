-- 209: a notification names what it is about (notifications v2, phase N1;
-- design refs/docs/proj/notification-v2.md).
--
-- Until now the only record of a notification's subject was the link string,
-- and several writers shared one link for different subjects: a like on a
-- work, on one of its resources and on one of its ratings all wrote
-- (liked, /galgame/W), so the second and third were deduplicated away. These
-- columns hold the reference instead. subject_kind / subject_id is the thing
-- the notification is about; root_kind / root_id is the container whose
-- removal removes it (the topic of a reply, the work of a resource, the wall
-- of a community post), equal to the subject for top-level subjects and NULL
-- where there is none (a followee activity, a system notice). is_suppressed is
-- for N2 (hidden / unpublished subjects); nothing sets it yet.
--
-- The writers fill the columns from this release on, so this migrates before
-- the deploy, which the deploy already does. content and link are still
-- written; the read path does not look at the new columns yet.
--
-- Existing rows: message_backfill_refs() places every row whose subject_kind
-- is NULL from its type and link and is called once below. Rules run most
-- specific first and each only touches rows still unplaced:
--   lottery-* / poll-closed on /topic/T  the topic's only lottery (poll), else
--                                        the only one the receiver entered
--                                        (voted in), else the topic
--   subscribed-topic                     the sender's newest reply of the fold,
--                                        else the reply at the link's floor
--   /topic/T?reply=F                     that reply, when it still exists
--   /topic/T?comment=C                   topic_comment C
--   /topic/T                             the topic
--   /galgame/W?comment=P                 community post P on work W
--   liked /galgame/W                     the receiver's work, resource or
--                                        rating of W the sender still likes,
--                                        when exactly one fits, else the work
--   expired /galgame/W                   the receiver's only resource on W (or
--                                        only expired one), else the work
--   /galgame/W                           the work (edit proposals have no id
--                                        in the link, nor have legacy comments)
--   /galgame/resource/R, -rating/N       on comment-family rows the wall
--                                        itself (subject = root = R); on others
--                                        the resource / rating under its work
--   quiz-answered /galgame-quiz/Q        the sender's answer, else the quiz
--   /galgame-quiz/Q, /toolset/N          the quiz, the toolset
--   /website/<host>                      the website with that url
--   /user/U                              the user
-- Rows nothing parses (/shop, /message/notice) stay NULL. The UPDATEs set
-- only the new columns, so trg_feed_message (UPDATE OF type, link, sender_id,
-- content, created) does not fire.
--
-- On the 2026-09-25 dev copy (368,494 rows) the whole file took 9 s, every row
-- placed, and message stays locked for that long: the old API waits on it.
-- Rows the old API writes after this runs and before it is replaced have no
-- references; SELECT message_backfill_refs(); places them and is safe to rerun.

ALTER TABLE message
    ADD COLUMN IF NOT EXISTS subject_kind  text,
    ADD COLUMN IF NOT EXISTS subject_id    bigint,
    ADD COLUMN IF NOT EXISTS root_kind     text,
    ADD COLUMN IF NOT EXISTS root_id       bigint,
    ADD COLUMN IF NOT EXISTS is_suppressed boolean NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_message_subject ON message (subject_kind, subject_id);
CREATE INDEX IF NOT EXISTS idx_message_root ON message (root_kind, root_id);

CREATE OR REPLACE FUNCTION message_backfill_refs() RETURNS bigint
LANGUAGE plpgsql AS $$
DECLARE
    placed bigint := 0;
    n      bigint;
    comment_family constant text[] := ARRAY['commented', 'replied', 'mentioned', 'followed', 'liked'];
BEGIN
    UPDATE message m SET subject_kind = 'lottery', subject_id = c.pick, root_kind = 'topic', root_id = c.topic_id
    FROM (
        SELECT x.id, x.topic_id, COALESCE(
            (SELECT min(l.id) FROM topic_lottery l WHERE l.topic_id = x.topic_id HAVING count(*) = 1),
            (SELECT min(l.id) FROM topic_lottery l
                JOIN topic_lottery_entry e ON e.lottery_id = l.id AND e.user_id = x.receiver_id
                WHERE l.topic_id = x.topic_id HAVING count(*) = 1)) AS pick
        FROM (SELECT id, receiver_id, (regexp_match(link, '^/topic/([0-9]{1,18})(?:[?&#/]|$)'))[1]::bigint AS topic_id
              FROM message
              WHERE subject_kind IS NULL AND type IN ('lottery-won', 'lottery-closed', 'lottery-expired')) x
        WHERE x.topic_id IS NOT NULL
    ) c
    WHERE m.id = c.id AND c.pick IS NOT NULL;
    GET DIAGNOSTICS n = ROW_COUNT;
    placed := placed + n;

    UPDATE message m SET subject_kind = 'poll', subject_id = c.pick, root_kind = 'topic', root_id = c.topic_id
    FROM (
        SELECT x.id, x.topic_id, COALESCE(
            (SELECT min(p.id) FROM topic_poll p WHERE p.topic_id = x.topic_id HAVING count(*) = 1),
            (SELECT min(p.id) FROM topic_poll p
                WHERE p.topic_id = x.topic_id
                  AND EXISTS (SELECT 1 FROM topic_poll_vote v WHERE v.poll_id = p.id AND v.user_id = x.receiver_id)
                HAVING count(*) = 1)) AS pick
        FROM (SELECT id, receiver_id, (regexp_match(link, '^/topic/([0-9]{1,18})(?:[?&#/]|$)'))[1]::bigint AS topic_id
              FROM message
              WHERE subject_kind IS NULL AND type = 'poll-closed') x
        WHERE x.topic_id IS NOT NULL
    ) c
    WHERE m.id = c.id AND c.pick IS NOT NULL;
    GET DIAGNOSTICS n = ROW_COUNT;
    placed := placed + n;

    -- A fold moves created to its newest reply and sender_id to that reply's
    -- author, but never rewrites the link (the fold's first floor).
    UPDATE message m SET subject_kind = 'reply', subject_id = c.pick, root_kind = 'topic', root_id = c.topic_id
    FROM (
        SELECT x.id, x.topic_id,
            (SELECT r.id FROM topic_reply r
             WHERE r.topic_id = x.topic_id AND r.user_id = x.sender_id AND r.floor >= x.floor
               AND r.created <= x.created + interval '1 minute'
             ORDER BY r.floor DESC LIMIT 1) AS pick
        FROM (SELECT id, sender_id, created, t[1]::bigint AS topic_id, t[2]::int AS floor
              FROM (SELECT id, sender_id, created, regexp_match(link, '^/topic/([0-9]{1,18})\?reply=([0-9]{1,9})(?:[&#]|$)') AS t
                    FROM message WHERE subject_kind IS NULL AND type = 'subscribed-topic') s
              WHERE t IS NOT NULL) x
    ) c
    WHERE m.id = c.id AND c.pick IS NOT NULL;
    GET DIAGNOSTICS n = ROW_COUNT;
    placed := placed + n;

    UPDATE message m SET subject_kind = 'reply', subject_id = r.id, root_kind = 'topic', root_id = x.topic_id
    FROM (SELECT id, t[1]::bigint AS topic_id, t[2]::int AS floor
          FROM (SELECT id, regexp_match(link, '^/topic/([0-9]{1,18})\?reply=([0-9]{1,9})(?:[&#]|$)') AS t
                FROM message WHERE subject_kind IS NULL) s
          WHERE t IS NOT NULL) x
    JOIN topic_reply r ON r.topic_id = x.topic_id AND r.floor = x.floor
    WHERE m.id = x.id;
    GET DIAGNOSTICS n = ROW_COUNT;
    placed := placed + n;

    UPDATE message m SET subject_kind = 'topic_comment', subject_id = x.t[2]::bigint, root_kind = 'topic', root_id = x.t[1]::bigint
    FROM (SELECT id, regexp_match(link, '^/topic/([0-9]{1,18})\?comment=([0-9]{1,18})(?:[&#]|$)') AS t
          FROM message WHERE subject_kind IS NULL) x
    WHERE m.id = x.id AND x.t IS NOT NULL;
    GET DIAGNOSTICS n = ROW_COUNT;
    placed := placed + n;

    UPDATE message m SET subject_kind = 'topic', subject_id = x.topic_id, root_kind = 'topic', root_id = x.topic_id
    FROM (SELECT id, (regexp_match(link, '^/topic/([0-9]{1,18})(?:[?&#/]|$)'))[1]::bigint AS topic_id
          FROM message WHERE subject_kind IS NULL) x
    WHERE m.id = x.id AND x.topic_id IS NOT NULL;
    GET DIAGNOSTICS n = ROW_COUNT;
    placed := placed + n;

    UPDATE message m SET subject_kind = 'community_post', subject_id = x.t[2]::bigint, root_kind = 'work', root_id = x.t[1]::bigint
    FROM (SELECT id, regexp_match(link, '^/galgame/([0-9]{1,18})\?comment=([0-9]{1,18})(?:[&#]|$)') AS t
          FROM message WHERE subject_kind IS NULL) x
    WHERE m.id = x.id AND x.t IS NOT NULL;
    GET DIAGNOSTICS n = ROW_COUNT;
    placed := placed + n;

    UPDATE message m SET subject_kind = c.kind, subject_id = c.pick, root_kind = 'work', root_id = c.work_id
    FROM (
        SELECT x.id, x.work_id, min(k.kind) AS kind, min(k.pick) AS pick
        FROM (SELECT id, sender_id, receiver_id, (regexp_match(link, '^/galgame/([0-9]{1,18})(?:[?&#/]|$)'))[1]::bigint AS work_id
              FROM message WHERE subject_kind IS NULL AND type = 'liked') x
        JOIN LATERAL (
            SELECT 'work' AS kind, g.id::bigint AS pick FROM galgame g
            WHERE g.id = x.work_id AND g.creator_user_id = x.receiver_id
              AND EXISTS (SELECT 1 FROM galgame_like l WHERE l.work_id = g.id AND l.user_id = x.sender_id)
            UNION ALL
            SELECT 'resource', r.id FROM galgame_resource r
            WHERE r.work_id = x.work_id AND r.user_id = x.receiver_id
              AND EXISTS (SELECT 1 FROM galgame_resource_like l WHERE l.galgame_resource_id = r.id AND l.user_id = x.sender_id)
            UNION ALL
            SELECT 'rating', g.id FROM galgame_rating g
            WHERE g.work_id = x.work_id AND g.user_id = x.receiver_id
              AND EXISTS (SELECT 1 FROM galgame_rating_like l WHERE l.galgame_rating_id = g.id AND l.user_id = x.sender_id)
        ) k ON true
        WHERE x.work_id IS NOT NULL
        GROUP BY x.id, x.work_id
        HAVING count(*) = 1
    ) c
    WHERE m.id = c.id;
    GET DIAGNOSTICS n = ROW_COUNT;
    placed := placed + n;

    UPDATE message m SET subject_kind = 'resource', subject_id = c.pick, root_kind = 'work', root_id = c.work_id
    FROM (
        SELECT x.id, x.work_id, COALESCE(
            (SELECT min(r.id) FROM galgame_resource r
             WHERE r.work_id = x.work_id AND r.user_id = x.receiver_id HAVING count(*) = 1),
            (SELECT min(r.id) FROM galgame_resource r
             WHERE r.work_id = x.work_id AND r.user_id = x.receiver_id AND r.status = 1 HAVING count(*) = 1)) AS pick
        FROM (SELECT id, receiver_id, (regexp_match(link, '^/galgame/([0-9]{1,18})(?:[?&#/]|$)'))[1]::bigint AS work_id
              FROM message WHERE subject_kind IS NULL AND type = 'expired') x
        WHERE x.work_id IS NOT NULL
    ) c
    WHERE m.id = c.id AND c.pick IS NOT NULL;
    GET DIAGNOSTICS n = ROW_COUNT;
    placed := placed + n;

    UPDATE message m SET subject_kind = 'work', subject_id = x.work_id, root_kind = 'work', root_id = x.work_id
    FROM (SELECT id, (regexp_match(link, '^/galgame/([0-9]{1,18})(?:[?&#/]|$)'))[1]::bigint AS work_id
          FROM message WHERE subject_kind IS NULL) x
    WHERE m.id = x.id AND x.work_id IS NOT NULL;
    GET DIAGNOSTICS n = ROW_COUNT;
    placed := placed + n;

    UPDATE message m SET subject_kind = 'resource', subject_id = x.resource_id,
        root_kind = CASE WHEN x.wall THEN 'resource' WHEN w.work_id IS NOT NULL THEN 'work' END,
        root_id = CASE WHEN x.wall THEN x.resource_id ELSE w.work_id END
    FROM (SELECT id, type = ANY (comment_family) AS wall,
                 (regexp_match(link, '^/galgame/resource/([0-9]{1,18})(?:[?&#/]|$)'))[1]::bigint AS resource_id
          FROM message WHERE subject_kind IS NULL) x
    LEFT JOIN LATERAL (
        SELECT COALESCE(
            (SELECT r.work_id FROM galgame_resource r WHERE r.id = x.resource_id),
            (SELECT g.work_id FROM galgame_resource_relocation g WHERE g.resource_id = x.resource_id))::bigint AS work_id
    ) w ON true
    WHERE m.id = x.id AND x.resource_id IS NOT NULL;
    GET DIAGNOSTICS n = ROW_COUNT;
    placed := placed + n;

    UPDATE message m SET subject_kind = 'rating', subject_id = x.rating_id,
        root_kind = CASE WHEN x.wall THEN 'rating' WHEN g.work_id IS NOT NULL THEN 'work' END,
        root_id = CASE WHEN x.wall THEN x.rating_id ELSE g.work_id END
    FROM (SELECT id, type = ANY (comment_family) AS wall,
                 (regexp_match(link, '^/galgame-rating/([0-9]{1,18})(?:[?&#/]|$)'))[1]::bigint AS rating_id
          FROM message WHERE subject_kind IS NULL) x
    LEFT JOIN galgame_rating g ON g.id = x.rating_id
    WHERE m.id = x.id AND x.rating_id IS NOT NULL;
    GET DIAGNOSTICS n = ROW_COUNT;
    placed := placed + n;

    UPDATE message m SET subject_kind = 'quiz_answer', subject_id = a.id, root_kind = 'quiz', root_id = x.quiz_id
    FROM (SELECT id, sender_id, (regexp_match(link, '^/galgame-quiz/([0-9]{1,18})(?:[?&#/]|$)'))[1]::bigint AS quiz_id
          FROM message WHERE subject_kind IS NULL AND type = 'quiz-answered') x
    JOIN galgame_quiz_answer a ON a.quiz_id = x.quiz_id AND a.user_id = x.sender_id
    WHERE m.id = x.id;
    GET DIAGNOSTICS n = ROW_COUNT;
    placed := placed + n;

    UPDATE message m SET subject_kind = 'quiz', subject_id = x.quiz_id, root_kind = 'quiz', root_id = x.quiz_id
    FROM (SELECT id, (regexp_match(link, '^/galgame-quiz/([0-9]{1,18})(?:[?&#/]|$)'))[1]::bigint AS quiz_id
          FROM message WHERE subject_kind IS NULL) x
    WHERE m.id = x.id AND x.quiz_id IS NOT NULL;
    GET DIAGNOSTICS n = ROW_COUNT;
    placed := placed + n;

    UPDATE message m SET subject_kind = 'toolset', subject_id = x.toolset_id, root_kind = 'toolset', root_id = x.toolset_id
    FROM (SELECT id, (regexp_match(link, '^/toolset/([0-9]{1,18})(?:[?&#/]|$)'))[1]::bigint AS toolset_id
          FROM message WHERE subject_kind IS NULL) x
    WHERE m.id = x.id AND x.toolset_id IS NOT NULL;
    GET DIAGNOSTICS n = ROW_COUNT;
    placed := placed + n;

    UPDATE message m SET subject_kind = 'website', subject_id = w.id, root_kind = 'website', root_id = w.id
    FROM (SELECT id, (regexp_match(link, '^/website/([^?#/]+)'))[1] AS host
          FROM message WHERE subject_kind IS NULL) x
    JOIN galgame_website w ON w.url = x.host
    WHERE m.id = x.id;
    GET DIAGNOSTICS n = ROW_COUNT;
    placed := placed + n;

    UPDATE message m SET subject_kind = 'user', subject_id = x.user_id, root_kind = 'user', root_id = x.user_id
    FROM (SELECT id, (regexp_match(link, '^/user/([0-9]{1,18})(?:[?&#/]|$)'))[1]::bigint AS user_id
          FROM message WHERE subject_kind IS NULL) x
    WHERE m.id = x.id AND x.user_id IS NOT NULL;
    GET DIAGNOSTICS n = ROW_COUNT;
    placed := placed + n;

    RETURN placed;
END $$;

SELECT message_backfill_refs();
