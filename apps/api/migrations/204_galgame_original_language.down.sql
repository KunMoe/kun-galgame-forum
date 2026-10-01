-- 204 down: drop the mirrored original language. Catalog still holds every value.
ALTER TABLE galgame DROP COLUMN IF EXISTS original_language;
