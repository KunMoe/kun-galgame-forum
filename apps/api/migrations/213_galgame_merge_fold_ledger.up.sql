-- 213: every merge fold writes down what it did, so an undone merge can be
-- folded back.
--
-- Since 2026-10-21 catalog can undo a work merge within 30 days: the old id
-- comes back and /v2/catalog/redirects reports the redirect as revoked. The
-- fold (GalgameMergeRepository.FoldTx) had moved the old work's resources,
-- activities, likes, favorites, contributors, quiz links and view buckets
-- onto the survivor with bulk UPDATEs and deleted the old galgame row, keeping
-- only the rows it dropped (galgame_merge_discarded, 082). Nothing said which
-- survivor rows had come from the old work, so an undo could not be followed.
--
-- galgame_merge_fold is one row per fold. old_row is the old work's galgame
-- row before the fold; survivor_before / survivor_after are the survivor's
-- row around it (survivor_before is NULL when the fold created the survivor
-- row). undone_at / undo_problems record the reverse and every row it could
-- not put back, as [{table, key, reason, work_id}].
--
-- galgame_merge_fold_row is one row per touched row: action 'moved' (work_id
-- rewritten; row_key names the row), 'dropped' (deleted because the survivor
-- already had its peer; before is the row), 'summed' (added into a survivor
-- row; before holds both rows) and 'repointed' (feed_activity rewritten by
-- hand; before is its old work_id and link).
--
-- Folds that ran before this migration have no ledger and stay as they are.
-- Additive: migrate before or with the deploy.

CREATE TABLE IF NOT EXISTS galgame_merge_fold (
  id              bigserial   PRIMARY KEY,
  old_work_id     integer     NOT NULL,
  new_work_id     integer     NOT NULL,
  folded_at       timestamptz NOT NULL DEFAULT now(),
  old_row         jsonb,
  survivor_before jsonb,
  survivor_after  jsonb,
  undone_at       timestamptz,
  undo_problems   jsonb
);

CREATE INDEX IF NOT EXISTS idx_galgame_merge_fold_active_old
  ON galgame_merge_fold (old_work_id) WHERE undone_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_galgame_merge_fold_active_new
  ON galgame_merge_fold (new_work_id) WHERE undone_at IS NULL;

CREATE TABLE IF NOT EXISTS galgame_merge_fold_row (
  id         bigserial PRIMARY KEY,
  fold_id    bigint    NOT NULL REFERENCES galgame_merge_fold (id) ON DELETE CASCADE,
  table_name text      NOT NULL,
  action     text      NOT NULL,
  row_key    jsonb     NOT NULL,
  before     jsonb
);

CREATE INDEX IF NOT EXISTS idx_galgame_merge_fold_row_fold ON galgame_merge_fold_row (fold_id);

SELECT user_purge_archive_attach();
