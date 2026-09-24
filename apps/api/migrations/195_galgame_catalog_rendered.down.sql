-- 195 down: drop both columns. The mirror goes back to not knowing which rows catalog stopped rendering.
ALTER TABLE galgame DROP COLUMN IF EXISTS catalog_rendered;
ALTER TABLE galgame DROP COLUMN IF EXISTS catalog_checked_at;
