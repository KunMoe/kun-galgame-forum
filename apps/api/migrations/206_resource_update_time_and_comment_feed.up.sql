-- 206: two leftovers on galgame_resource that the 2026-10-01 relocation to
-- LetMoe turned up.
--
-- 1. update_time is eight hours early on every row written before the Go
--    cutover. 022 converted the column AT TIME ZONE 'Asia/Shanghai', on the
--    premise that only the column DEFAULT ever wrote it. That holds since the
--    cutover (first row 2026-06-05 10:36 UTC). Before it the old stack wrote the
--    column itself, in UTC, at the same instant as created, so those values
--    landed eight hours before the truth. Production on 2026-10-01: 22,419 rows
--    below the cutoff, 15,782 of them exactly created - 8h, 6,637 carrying the
--    2025-07-20 site-migration stamp (13:58 UTC as stored; the likes written in
--    the same statement say 21:58 UTC), and none in the eight hours before the
--    cutoff. No forum code reads the column; the export to LetMoe did, and sent
--    88 resources "updated" before they were created.
--
--    The EXISTS makes a second run a no-op: once the rows are fixed, none is
--    exactly eight hours behind its created.
--
-- 2. Deleting a resource left the feed rows of its comments behind, linking to
--    a page that is gone: 21 on production, 4 of them from the relocation. The
--    delete branch of the feed trigger now takes them along, and the rows
--    already orphaned are removed.

BEGIN;

UPDATE galgame_resource
SET update_time = update_time + interval '8 hours'
WHERE update_time < timestamptz '2026-06-04 12:00:00+00'
  AND EXISTS (
    SELECT 1 FROM galgame_resource
    WHERE update_time < timestamptz '2026-06-04 12:00:00+00'
      AND created - update_time = interval '8 hours'
  );

CREATE OR REPLACE FUNCTION feed_sync_galgame_resource()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM feed_delete('GALGAME_RESOURCE_CREATION', OLD.id);
        DELETE FROM feed_activity
        WHERE type = 'GALGAME_RESOURCE_COMMENT_CREATION' AND link = '/galgame/resource/' || OLD.id;
        RETURN OLD;
    END IF;
    PERFORM feed_upsert('GALGAME_RESOURCE_CREATION', NEW.id, NEW.user_id, NEW.work_id, '', '/galgame/' || NEW.work_id, false, NEW.created);
    RETURN NEW;
END $function$;

DELETE FROM feed_activity f
WHERE f.type = 'GALGAME_RESOURCE_COMMENT_CREATION'
  AND NOT EXISTS (SELECT 1 FROM galgame_resource r WHERE '/galgame/resource/' || r.id = f.link);

COMMIT;
