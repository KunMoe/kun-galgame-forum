-- 206 down: put the feed trigger back as 141 left it and move update_time back
-- eight hours. The fixed rows end at 2026-06-04 19:47 UTC and the first row the
-- column DEFAULT wrote is 2026-06-05 10:36 UTC, so the cutoff below separates
-- them. The comment feed rows that were removed are not restored.

BEGIN;

UPDATE galgame_resource
SET update_time = update_time - interval '8 hours'
WHERE update_time < timestamptz '2026-06-04 20:00:00+00';

CREATE OR REPLACE FUNCTION feed_sync_galgame_resource()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('GALGAME_RESOURCE_CREATION', OLD.id); RETURN OLD; END IF;
    PERFORM feed_upsert('GALGAME_RESOURCE_CREATION', NEW.id, NEW.user_id, NEW.work_id, '', '/galgame/' || NEW.work_id, false, NEW.created);
    RETURN NEW;
END $function$;

COMMIT;
