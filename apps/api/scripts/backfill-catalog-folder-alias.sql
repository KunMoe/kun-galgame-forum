-- Fills galgame_collection.catalog_folder_id from the mapping the catalog
-- importer recorded when it created the folders. Run it AFTER migration 091
-- and AFTER nextmoe-infra's `import-favorites -apply`, and BEFORE deploying
-- the collection cutover: between the migration and this script every existing
-- collection resolves to no folder and answers 404.
--
-- Cross-database on purpose. catalog_user_folder_import lives in kun_catalog
-- and galgame_collection in kungalgame, so the mapping travels through a file
-- rather than a join. Run as a superuser with \i, in one psql session:
--
--   sudo docker exec -i <postgres> psql -U postgres -v ON_ERROR_STOP=1 \
--     -f /path/to/backfill-catalog-folder-alias.sql
--
-- Idempotent: re-running it re-writes the same values. Run it again after the
-- deploy to sweep collections created in the window between the import and the
-- cutover.

\set ON_ERROR_STOP on
\pset pager off

\c kun_catalog
\echo '=== forum-origin folder mappings in the catalog'
SELECT count(*) AS mappings FROM catalog_user_folder_import WHERE site = 'forum';
\copy (SELECT source_id, folder_id FROM catalog_user_folder_import WHERE site = 'forum') TO '/tmp/forum_folder_alias.csv' CSV

\c kungalgame
CREATE TEMP TABLE folder_alias (source_id BIGINT PRIMARY KEY, folder_id BIGINT NOT NULL);
\copy folder_alias FROM '/tmp/forum_folder_alias.csv' CSV

\echo '=== before'
SELECT count(*) FILTER (WHERE catalog_folder_id IS NOT NULL) AS mapped,
       count(*) FILTER (WHERE catalog_folder_id IS NULL) AS unmapped,
       count(*) AS total
  FROM galgame_collection;

BEGIN;
UPDATE galgame_collection c
   SET catalog_folder_id = a.folder_id
  FROM folder_alias a
 WHERE c.id = a.source_id
   AND c.catalog_folder_id IS DISTINCT FROM a.folder_id;
COMMIT;

\echo '=== after'
SELECT count(*) FILTER (WHERE catalog_folder_id IS NOT NULL) AS mapped,
       count(*) FILTER (WHERE catalog_folder_id IS NULL) AS unmapped,
       count(*) AS total
  FROM galgame_collection;

\echo '=== collections the catalog has never heard of (these answer 404 after the cutover)'
SELECT id, user_id, created FROM galgame_collection
 WHERE catalog_folder_id IS NULL ORDER BY created DESC LIMIT 20;
