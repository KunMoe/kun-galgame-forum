-- 212: an edit activity carries the field keys its catalog revision touched.
--
-- The activity card used to look them up itself: on mount it called
-- GET /api/v1/works/{id}/edit-revisions/diff, one or two catalog reads per card.
-- Those edit endpoints are retired (editing lives on editor.nextmoe.com), and
-- the revision feed the edit-revision sync already reads carries
-- changed_fields, so the sync stores them here.
--
-- Existing rows stay NULL and their cards show no field list. Do not replay the
-- sync to fill them: a replay also re-inserts edit activities that a user purge
-- deleted.
--
-- Additive and nullable: migrate before or with the deploy.

ALTER TABLE galgame_activity ADD COLUMN IF NOT EXISTS edit_changed_fields jsonb;
