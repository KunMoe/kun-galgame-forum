-- 224: a view counter for catalog folders (收藏夹), kept on this site.
--
-- Catalog owns the folders; how often this forum's readers open one is the
-- forum's own number, so it lives here, keyed by the catalog folder id. A row
-- is created on the first read of a folder's page; there are no existing rows
-- to backfill. Every read adds one: no dedupe by reader.
--
-- Additive: runs with the deploy.

CREATE TABLE IF NOT EXISTS galgame_collection_view (
    catalog_folder_id bigint PRIMARY KEY,
    view_count bigint NOT NULL DEFAULT 0,
    updated timestamptz NOT NULL DEFAULT now()
);
