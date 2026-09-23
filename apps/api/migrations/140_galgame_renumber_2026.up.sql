-- 140: the ledger for the one-shot renumber of galgame.id onto catalog_work.id.
--
-- catalog minted catalog_work.id from a sequence in gid order on 2026-07-06 and
-- every missing wiki row shifted the rest: gid 1-922 are equal, from 923 the
-- catalog id drifts to -185, and only 2,493 of 15,990 forum pages had gid ==
-- catalog id on 2026-09-23. The forum kept a gid <-> catalog id bridge to route
-- correctly and several faces skipped it (the folder faces wrote 148,548 gids
-- into catalog as if they were work ids). The user ruled on 2026-09-23 that the
-- forum takes infra's id: galgame.id is renumbered to the catalog work id once,
-- and no mapping exists afterwards. See docs/proj/gid-is-work-id.md.
--
-- cmd/align-galgame-ids fills this table inside the renumber transaction, one
-- row per legacy number it knows about (every curated anchor, every kungal
-- claim, every local galgame row), not only the local rows: links, community
-- walls and trust reports name legacy numbers that never had a local row. The
-- infra side of the same window reads it to rewrite its own copies. Nothing in
-- the running forum reads it.
--
-- Inert on its own: an empty table.

BEGIN;

CREATE TABLE IF NOT EXISTS galgame_renumber_2026 (
  old_id  bigint      PRIMARY KEY,
  new_id  bigint      NOT NULL,
  how     text        NOT NULL CHECK (how IN ('curated', 'claim', 'unchanged')),
  created timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_galgame_renumber_2026_new ON galgame_renumber_2026 (new_id);

COMMIT;
