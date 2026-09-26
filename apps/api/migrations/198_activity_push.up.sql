-- 198: transactional outbox so the forum can push public feed_activity rows to
-- NextMoe community POST /activities, and remember the last accepted write so a
-- later delete can still name the actor on a tombstone.
--
-- activity_push_queue is the pending work. A trigger on feed_activity upserts
-- the touched (type, source_id) and refreshes enqueued only — a pending
-- backfill row stays a backfill row. Existing feed_activity rows are enqueued
-- once with backfill = true so the first drain does not notify followers of
-- publications that already fanned out locally. activity_push_sent is empty
-- until community accepts a write. Additive: migrate before deploy.

CREATE TABLE IF NOT EXISTS activity_push_queue (
    type       varchar(64) NOT NULL,
    source_id  int         NOT NULL,
    backfill   boolean     NOT NULL DEFAULT false,
    enqueued   timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (type, source_id)
);

CREATE INDEX IF NOT EXISTS idx_activity_push_queue_enqueued ON activity_push_queue (enqueued);

CREATE TABLE IF NOT EXISTS activity_push_sent (
    type       varchar(64) NOT NULL,
    source_id  int         NOT NULL,
    actor_id   int         NOT NULL,
    revision   bigint      NOT NULL,
    removed    boolean     NOT NULL,
    sent_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (type, source_id)
);

CREATE OR REPLACE FUNCTION activity_push_enqueue() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO activity_push_queue (type, source_id)
        VALUES (NEW.type, NEW.source_id)
        ON CONFLICT (type, source_id) DO UPDATE SET enqueued = clock_timestamp();
        RETURN NEW;
    END IF;
    IF TG_OP = 'UPDATE' THEN
        INSERT INTO activity_push_queue (type, source_id)
        VALUES (NEW.type, NEW.source_id)
        ON CONFLICT (type, source_id) DO UPDATE SET enqueued = clock_timestamp();
        IF NEW.type IS DISTINCT FROM OLD.type OR NEW.source_id IS DISTINCT FROM OLD.source_id THEN
            INSERT INTO activity_push_queue (type, source_id)
            VALUES (OLD.type, OLD.source_id)
            ON CONFLICT (type, source_id) DO UPDATE SET enqueued = clock_timestamp();
        END IF;
        RETURN NEW;
    END IF;
    INSERT INTO activity_push_queue (type, source_id)
    VALUES (OLD.type, OLD.source_id)
    ON CONFLICT (type, source_id) DO UPDATE SET enqueued = clock_timestamp();
    RETURN OLD;
END;
$$;

DROP TRIGGER IF EXISTS trg_activity_push_enqueue ON feed_activity;
CREATE TRIGGER trg_activity_push_enqueue
    AFTER INSERT OR UPDATE OR DELETE ON feed_activity
    FOR EACH ROW
    EXECUTE FUNCTION activity_push_enqueue();

INSERT INTO activity_push_queue (type, source_id, backfill)
SELECT type, source_id, true FROM feed_activity
ON CONFLICT DO NOTHING;

SELECT user_purge_archive_attach();
