-- Drops the author's is_entity_cover_blurred choice from topics and drafts.

BEGIN;

ALTER TABLE topic DROP COLUMN IF EXISTS is_entity_cover_blurred;
ALTER TABLE topic_draft DROP COLUMN IF EXISTS is_entity_cover_blurred;

COMMIT;
