-- 208 down: drop toolset favorites and their cached count.
DROP TABLE IF EXISTS galgame_toolset_favorite;
ALTER TABLE galgame_toolset DROP COLUMN IF EXISTS favorite_count;
