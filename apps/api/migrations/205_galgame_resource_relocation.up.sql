-- 205: where a resource went when it left the forum.
--
-- The forum hosts download resources only for works whose original language is
-- Japanese or Chinese. The resources of every other work move to LetMoe (same
-- work id, LetMoe's own resource id). cmd/relocate-resources writes one row
-- here per resource before anything is sent, and the row outlives the resource:
--
--   payload            the resource exactly as it was exported, with its links
--                      and likes. It is the only rollback material once the
--                      galgame_resource row is deleted, so nothing rewrites it
--                      after pushed_at is set.
--   destination_id     LetMoe's resource id, from the import receipt.
--   destination_public false when LetMoe holds the resource back from public
--                      view (an expired link lands as needs_fix); the redirect
--                      then targets the game page instead of the resource page.
--   retired_at         when the forum's own row was deleted.
--
-- uploader_id is kept after an account purge: the row is the redirect for links
-- that already point at the old resource page.
--
-- Existing rows: none. The table starts empty and nothing reads it until the
-- relocation runs.
CREATE TABLE IF NOT EXISTS galgame_resource_relocation (
    resource_id        integer PRIMARY KEY,
    work_id            integer NOT NULL,
    uploader_id        integer NOT NULL,
    payload            jsonb NOT NULL,
    exported_at        timestamptz NOT NULL DEFAULT now(),
    destination_id     bigint,
    destination_public boolean,
    pushed_at          timestamptz,
    retired_at         timestamptz
);

CREATE INDEX IF NOT EXISTS idx_galgame_resource_relocation_work
    ON galgame_resource_relocation (work_id);

SELECT user_purge_archive_attach();
