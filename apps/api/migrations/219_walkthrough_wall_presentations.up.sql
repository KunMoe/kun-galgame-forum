-- 219: comment walls on catalog walkthroughs (anchor 2, 'walkthrough:<id>').
--
-- A walkthrough has no local row, so its wall enters the presentation outbox
-- the way a work with no galgame row does: through its comment feed rows,
-- GALGAME_WALKTHROUGH_COMMENT_CREATION, whose work_id is the walkthrough's work
-- and whose link is /galgame-walkthrough/<id>. The feed trigger enqueues the
-- anchor on insert, and a change to the work (catalog_rendered, published,
-- content_limit) enqueues its walkthrough walls like its other children.
--
-- No backfill: no walkthrough wall has a comment before this ships.
-- Additive: migrate before deploy.

BEGIN;

CREATE OR REPLACE FUNCTION anchor_presentation_enqueue_work_children(p_work_id integer) RETURNS void
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM anchor_presentation_enqueue(2::smallint, 'resource:' || r.id)
    FROM galgame_resource r WHERE r.work_id = p_work_id;
    PERFORM anchor_presentation_enqueue(2::smallint, 'rating:' || r.id)
    FROM galgame_rating r WHERE r.work_id = p_work_id;
    PERFORM anchor_presentation_enqueue(2::smallint, 'quiz:' || q.quiz_id)
    FROM galgame_quiz_galgame q WHERE q.work_id = p_work_id;
    PERFORM anchor_presentation_enqueue(2::smallint, 'walkthrough:' || w.id)
    FROM (
        SELECT DISTINCT substring(fa.link FROM '^/galgame-walkthrough/([0-9]{1,18})$') AS id
        FROM feed_activity fa
        WHERE fa.type = 'GALGAME_WALKTHROUGH_COMMENT_CREATION' AND fa.work_id = p_work_id
    ) w WHERE w.id IS NOT NULL;
END;
$$;

CREATE OR REPLACE FUNCTION anchor_presentation_on_feed_activity() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    walkthrough_id text;
BEGIN
    IF NEW.type = 'GALGAME_COMMENT_CREATION' AND NEW.work_id > 0 THEN
        PERFORM anchor_presentation_enqueue(1::smallint, NEW.work_id::text);
    ELSIF NEW.type = 'GALGAME_WALKTHROUGH_COMMENT_CREATION' THEN
        walkthrough_id := substring(NEW.link FROM '^/galgame-walkthrough/([0-9]{1,18})$');
        IF walkthrough_id IS NOT NULL THEN
            PERFORM anchor_presentation_enqueue(2::smallint, 'walkthrough:' || walkthrough_id);
        END IF;
    END IF;
    RETURN NEW;
END;
$$;

COMMIT;
