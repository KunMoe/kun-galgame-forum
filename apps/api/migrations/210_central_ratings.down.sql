-- 210 down: the trigger forgets written_here and the mirror columns go. The
-- foreign key, play_status's NOT NULL and the id default are not restored: the
-- mirror may hold ratings on works without a local row, and rows the mirror
-- wrote carry no play status. rating_cutover stays; it is infra's backfill
-- source and is dropped only by the migration that drops play_status.

CREATE OR REPLACE FUNCTION feed_sync_galgame_rating()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('GALGAME_RATING_CREATION', OLD.id); RETURN OLD; END IF;
    PERFORM feed_upsert('GALGAME_RATING_CREATION', NEW.id, NEW.user_id, NEW.work_id,
        CASE WHEN NEW.spoiler_level <> 'none' THEN '⚠️ 该评分可能含有剧透内容，点进查看'
             ELSE SUBSTRING(COALESCE(NEW.short_summary, ''), 1, 100) END,
        '/galgame-rating/' || NEW.id, false, NEW.created);
    RETURN NEW;
END $function$;

DROP TRIGGER IF EXISTS trg_feed_galgame_rating ON galgame_rating;
CREATE TRIGGER trg_feed_galgame_rating
    AFTER INSERT OR DELETE OR UPDATE OF user_id, work_id, short_summary, spoiler_level, created
    ON galgame_rating
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_rating();

ALTER TABLE galgame_rating
    DROP COLUMN IF EXISTS author_state,
    DROP COLUMN IF EXISTS author_completion,
    DROP COLUMN IF EXISTS client_id,
    DROP COLUMN IF EXISTS site,
    DROP COLUMN IF EXISTS written_here;
