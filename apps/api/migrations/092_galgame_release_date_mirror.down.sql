-- Drops the bookkeeping column only. release_date itself stays, and the mirror
-- goes back to being unable to tell "catalog has no date" from "never asked".

BEGIN;

DROP INDEX IF EXISTS idx_galgame_release_unsynced;
ALTER TABLE galgame DROP COLUMN IF EXISTS release_date_synced_at;

COMMIT;
