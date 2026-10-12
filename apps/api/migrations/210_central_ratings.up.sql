-- 210: galgame_rating becomes a mirror of NextMoe's ratings (infra spec 2.44.0,
-- owner rulings 2026-10-11). A person has one rating per work across every
-- site; the forum writes it through /v2/me/work-records and /v2/me/ratings and
-- reads it back off /v2/ratings/changes. The row id is NextMoe's rating id: the
-- import of 2026-10-11 kept every forum id, so existing rows keep theirs.
--
-- The snapshot comes first, before any ALTER. infra's backfill import after the
-- deploy reads rating_cutover.galgame_rating to pick up the ratings the old
-- code wrote between its incremental import and this migration. It is dropped
-- by the later migration that drops play_status, once infra has confirmed.
--
-- No row is deleted or truncated here: galgame_rating_like cascades off the
-- rating, so emptying the table would take every like with it. The mirror
-- upserts by id over the existing rows.
--
-- The foreign key to galgame goes: the mirror holds every rating NextMoe has,
-- whether or not the forum has a local row for the work yet. LetMoe kept its
-- equivalent check and missed 88 ratings on 27 works that reached its local
-- table only after their ratings had passed in the feed.
--
-- written_here marks a rating first written through this forum or the forum's
-- apps; only those reach the home feed and, through it, the following
-- activity feed. The mirror sets it on every row from the rating's client_id.
-- Every existing row was written here, so the column is added with true and
-- its default then turns false: a later insert that forgets it must not push
-- another site's rating out.
--
-- The feed function returns early on an UPDATE that leaves its columns as
-- they were. A trigger's UPDATE OF list fires whenever the column is in the
-- SET list, changed or not, so without it every mirror upsert, and every play
-- state change NextMoe sends, would re-push the rating's activity.
--
-- play_status stops being written (the author's current state comes with the
-- rating as author_state/author_completion), so it loses its NOT NULL and its
-- stale 'not_started' default; the column itself is dropped deploy-then-drop.
-- The id loses its sequence default so nothing can mint a local id again.

CREATE SCHEMA IF NOT EXISTS rating_cutover;
CREATE TABLE IF NOT EXISTS rating_cutover.galgame_rating AS TABLE public.galgame_rating;

ALTER TABLE galgame_rating DROP CONSTRAINT IF EXISTS galgame_rating_galgame_id_fkey;

-- The feed trigger names short_summary in its UPDATE OF list, and Postgres
-- refuses to change the type of a column a trigger names, so it is dropped
-- here and recreated below.
DROP TRIGGER IF EXISTS trg_feed_galgame_rating ON galgame_rating;

ALTER TABLE galgame_rating
    ADD COLUMN IF NOT EXISTS author_state      text,
    ADD COLUMN IF NOT EXISTS author_completion text,
    ADD COLUMN IF NOT EXISTS client_id         text    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS site              text    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS written_here      boolean NOT NULL DEFAULT true;
ALTER TABLE galgame_rating ALTER COLUMN written_here SET DEFAULT false;

ALTER TABLE galgame_rating ALTER COLUMN short_summary TYPE varchar(2000);
ALTER TABLE galgame_rating ALTER COLUMN play_status DROP NOT NULL;
ALTER TABLE galgame_rating ALTER COLUMN play_status DROP DEFAULT;
ALTER TABLE galgame_rating ALTER COLUMN id DROP DEFAULT;

CREATE OR REPLACE FUNCTION feed_sync_galgame_rating()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('GALGAME_RATING_CREATION', OLD.id); RETURN OLD; END IF;
    IF TG_OP = 'UPDATE'
       AND (OLD.user_id, OLD.work_id, OLD.short_summary, OLD.spoiler_level, OLD.created, OLD.written_here)
           IS NOT DISTINCT FROM
           (NEW.user_id, NEW.work_id, NEW.short_summary, NEW.spoiler_level, NEW.created, NEW.written_here) THEN
        RETURN NEW;
    END IF;
    IF NOT NEW.written_here THEN
        PERFORM feed_delete('GALGAME_RATING_CREATION', NEW.id);
        RETURN NEW;
    END IF;
    PERFORM feed_upsert('GALGAME_RATING_CREATION', NEW.id, NEW.user_id, NEW.work_id,
        CASE WHEN NEW.spoiler_level <> 'none' THEN '⚠️ 该评分可能含有剧透内容，点进查看'
             ELSE SUBSTRING(COALESCE(NEW.short_summary, ''), 1, 100) END,
        '/galgame-rating/' || NEW.id, false, NEW.created);
    RETURN NEW;
END $function$;

CREATE TRIGGER trg_feed_galgame_rating
    AFTER INSERT OR DELETE OR UPDATE OF user_id, work_id, short_summary, spoiler_level, created, written_here
    ON galgame_rating
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_rating();
