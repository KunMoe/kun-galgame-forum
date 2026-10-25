-- Restores the 200 bodies: walkthrough walls stop entering the outbox.

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
END;
$$;

CREATE OR REPLACE FUNCTION anchor_presentation_on_feed_activity() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.type = 'GALGAME_COMMENT_CREATION' AND NEW.work_id > 0 THEN
        PERFORM anchor_presentation_enqueue(1::smallint, NEW.work_id::text);
    END IF;
    RETURN NEW;
END;
$$;

COMMIT;
