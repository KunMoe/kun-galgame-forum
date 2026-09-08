-- The catalog is the canonical favorites store from this wave on: a collection
-- is a catalog folder (nextmoe-infra /v2/me/folders, deviations 106/115) and
-- galgame_collection stops being a store at all.
--
-- The table stays, for two reasons. Its id is the id in every
-- /galgame/collection/<id> link ever shared, so the row survives as the ALIAS
-- that maps that id to the catalog folder; and its content columns (name,
-- description, visibility, is_default, item_count) are frozen at cutover as
-- the rollback material for the guarded retirement that comes later. Nothing
-- reads them after this migration, and rows created after it leave them at
-- their defaults — a rollback restores the world as of the cutover, which is
-- what a rollback is.
--
-- Existing rows are filled by nextmoe-infra's `import-favorites -write-back`:
-- that process created the folders and holds the mapping in
-- catalog_user_folder_import. Run it AFTER this migration and BEFORE deploying
-- the cutover, or every existing collection resolves to no folder and answers
-- 404.
ALTER TABLE galgame_collection ADD COLUMN IF NOT EXISTS catalog_folder_id BIGINT;

-- Partial, because the column is NULL on every row until the write-back runs
-- and on any row whose folder was deleted upstream.
CREATE UNIQUE INDEX IF NOT EXISTS idx_galgame_collection_catalog_folder
  ON galgame_collection (catalog_folder_id)
  WHERE catalog_folder_id IS NOT NULL;
