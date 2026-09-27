-- 200: transactional outbox so the forum can push comment-wall page presentations
-- to NextMoe community PUT /anchor-presentations (title, url, work, cover,
-- content_limit per wall anchor), and remember the last accepted write so a
-- later hide/delete can tombstone only an anchor community has accepted.
--
-- anchor_presentation_queue is the pending work. Triggers on the six wall page
-- tables (and on GALGAME_COMMENT_CREATION feed rows, which is how a work with
-- no local galgame row enters) upsert the touched (anchor_kind, anchor_id) and
-- refresh enqueued. Existing anchors (~74k rows in production) are enqueued
-- once so the first drain fills community before comments on those walls can
-- produce feed items. anchor_presentation_sent is empty until community accepts
-- a write. Additive: migrate before deploy.

CREATE TABLE IF NOT EXISTS anchor_presentation_queue (
    anchor_kind smallint     NOT NULL,
    anchor_id   varchar(128) NOT NULL,
    enqueued    timestamptz  NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (anchor_kind, anchor_id)
);

CREATE INDEX IF NOT EXISTS idx_anchor_presentation_queue_enqueued
    ON anchor_presentation_queue (enqueued);

CREATE TABLE IF NOT EXISTS anchor_presentation_sent (
    anchor_kind smallint     NOT NULL,
    anchor_id   varchar(128) NOT NULL,
    revision    bigint       NOT NULL,
    removed     boolean      NOT NULL,
    sent_at     timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (anchor_kind, anchor_id)
);

CREATE OR REPLACE FUNCTION anchor_presentation_enqueue(p_kind smallint, p_id text) RETURNS void
LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO anchor_presentation_queue (anchor_kind, anchor_id)
    VALUES (p_kind, p_id)
    ON CONFLICT (anchor_kind, anchor_id) DO UPDATE SET enqueued = clock_timestamp();
END;
$$;

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

-- galgame is updated on every page view (view, like_count) and by every catalog
-- mirror poll, which rewrites catalog_rendered with the value it already holds.
-- The trigger names its columns so a view never calls this function, and the
-- function compares values so a poll enqueues nothing.
CREATE OR REPLACE FUNCTION anchor_presentation_on_galgame() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        PERFORM anchor_presentation_enqueue(1::smallint, NEW.id::text);
        RETURN NEW;
    END IF;
    IF TG_OP = 'DELETE' THEN
        PERFORM anchor_presentation_enqueue(1::smallint, OLD.id::text);
        PERFORM anchor_presentation_enqueue_work_children(OLD.id);
        RETURN OLD;
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.published IS DISTINCT FROM OLD.published
       OR NEW.content_limit IS DISTINCT FROM OLD.content_limit
       OR NEW.catalog_rendered IS DISTINCT FROM OLD.catalog_rendered THEN
        PERFORM anchor_presentation_enqueue(1::smallint, NEW.id::text);
        PERFORM anchor_presentation_enqueue_work_children(NEW.id);
        IF NEW.id IS DISTINCT FROM OLD.id THEN
            PERFORM anchor_presentation_enqueue(1::smallint, OLD.id::text);
            PERFORM anchor_presentation_enqueue_work_children(OLD.id);
        END IF;
    END IF;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION anchor_presentation_on_galgame_resource() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        PERFORM anchor_presentation_enqueue(2::smallint, 'resource:' || NEW.id);
        RETURN NEW;
    END IF;
    IF TG_OP = 'DELETE' THEN
        PERFORM anchor_presentation_enqueue(2::smallint, 'resource:' || OLD.id);
        RETURN OLD;
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.work_id IS DISTINCT FROM OLD.work_id
       OR NEW.user_id IS DISTINCT FROM OLD.user_id THEN
        PERFORM anchor_presentation_enqueue(2::smallint, 'resource:' || NEW.id);
        IF NEW.id IS DISTINCT FROM OLD.id THEN
            PERFORM anchor_presentation_enqueue(2::smallint, 'resource:' || OLD.id);
        END IF;
    END IF;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION anchor_presentation_on_galgame_rating() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        PERFORM anchor_presentation_enqueue(2::smallint, 'rating:' || NEW.id);
        RETURN NEW;
    END IF;
    IF TG_OP = 'DELETE' THEN
        PERFORM anchor_presentation_enqueue(2::smallint, 'rating:' || OLD.id);
        RETURN OLD;
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.work_id IS DISTINCT FROM OLD.work_id
       OR NEW.user_id IS DISTINCT FROM OLD.user_id THEN
        PERFORM anchor_presentation_enqueue(2::smallint, 'rating:' || NEW.id);
        IF NEW.id IS DISTINCT FROM OLD.id THEN
            PERFORM anchor_presentation_enqueue(2::smallint, 'rating:' || OLD.id);
        END IF;
    END IF;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION anchor_presentation_on_galgame_quiz() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        PERFORM anchor_presentation_enqueue(2::smallint, 'quiz:' || NEW.id);
        RETURN NEW;
    END IF;
    IF TG_OP = 'DELETE' THEN
        PERFORM anchor_presentation_enqueue(2::smallint, 'quiz:' || OLD.id);
        RETURN OLD;
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.question IS DISTINCT FROM OLD.question
       OR NEW.hide_galgame IS DISTINCT FROM OLD.hide_galgame
       OR NEW.user_id IS DISTINCT FROM OLD.user_id THEN
        PERFORM anchor_presentation_enqueue(2::smallint, 'quiz:' || NEW.id);
        IF NEW.id IS DISTINCT FROM OLD.id THEN
            PERFORM anchor_presentation_enqueue(2::smallint, 'quiz:' || OLD.id);
        END IF;
    END IF;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION anchor_presentation_on_galgame_quiz_galgame() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        PERFORM anchor_presentation_enqueue(2::smallint, 'quiz:' || NEW.quiz_id);
        RETURN NEW;
    END IF;
    IF TG_OP = 'UPDATE' THEN
        PERFORM anchor_presentation_enqueue(2::smallint, 'quiz:' || NEW.quiz_id);
        PERFORM anchor_presentation_enqueue(2::smallint, 'quiz:' || OLD.quiz_id);
        RETURN NEW;
    END IF;
    PERFORM anchor_presentation_enqueue(2::smallint, 'quiz:' || OLD.quiz_id);
    RETURN OLD;
END;
$$;

CREATE OR REPLACE FUNCTION anchor_presentation_on_galgame_toolset() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        PERFORM anchor_presentation_enqueue(2::smallint, 'toolset:' || NEW.id);
        RETURN NEW;
    END IF;
    IF TG_OP = 'DELETE' THEN
        PERFORM anchor_presentation_enqueue(2::smallint, 'toolset:' || OLD.id);
        RETURN OLD;
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.name IS DISTINCT FROM OLD.name
       OR NEW.status IS DISTINCT FROM OLD.status
       OR NEW.user_id IS DISTINCT FROM OLD.user_id THEN
        PERFORM anchor_presentation_enqueue(2::smallint, 'toolset:' || NEW.id);
        IF NEW.id IS DISTINCT FROM OLD.id THEN
            PERFORM anchor_presentation_enqueue(2::smallint, 'toolset:' || OLD.id);
        END IF;
    END IF;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION anchor_presentation_on_galgame_website() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        PERFORM anchor_presentation_enqueue(2::smallint, 'website:' || NEW.id);
        RETURN NEW;
    END IF;
    IF TG_OP = 'DELETE' THEN
        PERFORM anchor_presentation_enqueue(2::smallint, 'website:' || OLD.id);
        RETURN OLD;
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.name IS DISTINCT FROM OLD.name
       OR NEW.url IS DISTINCT FROM OLD.url
       OR NEW.age_limit IS DISTINCT FROM OLD.age_limit THEN
        PERFORM anchor_presentation_enqueue(2::smallint, 'website:' || NEW.id);
        IF NEW.id IS DISTINCT FROM OLD.id THEN
            PERFORM anchor_presentation_enqueue(2::smallint, 'website:' || OLD.id);
        END IF;
    END IF;
    RETURN NEW;
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

DROP TRIGGER IF EXISTS trg_anchor_presentation_galgame ON galgame;
CREATE TRIGGER trg_anchor_presentation_galgame
    AFTER INSERT OR DELETE OR UPDATE OF id, published, content_limit, catalog_rendered ON galgame
    FOR EACH ROW
    EXECUTE FUNCTION anchor_presentation_on_galgame();

DROP TRIGGER IF EXISTS trg_anchor_presentation_galgame_resource ON galgame_resource;
CREATE TRIGGER trg_anchor_presentation_galgame_resource
    AFTER INSERT OR DELETE OR UPDATE OF id, work_id, user_id ON galgame_resource
    FOR EACH ROW
    EXECUTE FUNCTION anchor_presentation_on_galgame_resource();

DROP TRIGGER IF EXISTS trg_anchor_presentation_galgame_rating ON galgame_rating;
CREATE TRIGGER trg_anchor_presentation_galgame_rating
    AFTER INSERT OR DELETE OR UPDATE OF id, work_id, user_id ON galgame_rating
    FOR EACH ROW
    EXECUTE FUNCTION anchor_presentation_on_galgame_rating();

DROP TRIGGER IF EXISTS trg_anchor_presentation_galgame_quiz ON galgame_quiz;
CREATE TRIGGER trg_anchor_presentation_galgame_quiz
    AFTER INSERT OR DELETE OR UPDATE OF id, question, hide_galgame, user_id ON galgame_quiz
    FOR EACH ROW
    EXECUTE FUNCTION anchor_presentation_on_galgame_quiz();

DROP TRIGGER IF EXISTS trg_anchor_presentation_galgame_quiz_galgame ON galgame_quiz_galgame;
CREATE TRIGGER trg_anchor_presentation_galgame_quiz_galgame
    AFTER INSERT OR UPDATE OR DELETE ON galgame_quiz_galgame
    FOR EACH ROW
    EXECUTE FUNCTION anchor_presentation_on_galgame_quiz_galgame();

DROP TRIGGER IF EXISTS trg_anchor_presentation_galgame_toolset ON galgame_toolset;
CREATE TRIGGER trg_anchor_presentation_galgame_toolset
    AFTER INSERT OR DELETE OR UPDATE OF id, name, status, user_id ON galgame_toolset
    FOR EACH ROW
    EXECUTE FUNCTION anchor_presentation_on_galgame_toolset();

DROP TRIGGER IF EXISTS trg_anchor_presentation_galgame_website ON galgame_website;
CREATE TRIGGER trg_anchor_presentation_galgame_website
    AFTER INSERT OR DELETE OR UPDATE OF id, name, url, age_limit ON galgame_website
    FOR EACH ROW
    EXECUTE FUNCTION anchor_presentation_on_galgame_website();

DROP TRIGGER IF EXISTS trg_anchor_presentation_feed_activity ON feed_activity;
CREATE TRIGGER trg_anchor_presentation_feed_activity
    AFTER INSERT ON feed_activity
    FOR EACH ROW
    EXECUTE FUNCTION anchor_presentation_on_feed_activity();

INSERT INTO anchor_presentation_queue (anchor_kind, anchor_id)
SELECT 1, id::text FROM galgame
UNION
SELECT 1, work_id::text FROM feed_activity
    WHERE type = 'GALGAME_COMMENT_CREATION' AND work_id > 0
UNION
SELECT 2, 'resource:' || id FROM galgame_resource
UNION
SELECT 2, 'rating:' || id FROM galgame_rating
UNION
SELECT 2, 'quiz:' || id FROM galgame_quiz
UNION
SELECT 2, 'toolset:' || id FROM galgame_toolset
UNION
SELECT 2, 'website:' || id FROM galgame_website
ON CONFLICT DO NOTHING;

SELECT user_purge_archive_attach();
