-- 092: give galgame.release_date a keeper, and make "never mirrored" sayable.
--
-- 013 added release_date as a mirror of the wiki's field and fed it with
-- cmd/backfill-release-date. 157d7abb deleted that command with the rest of the
-- retired wiki lanes, on the reasoning that "the column it fed is backfilled and
-- effectively immutable". Measured on production 2026-09-07, a month later:
-- 2,413 of the 4,578 rows on /galgame (52.7%) had a NULL release_date, so the
-- date sort ran out of ordered rows at page 91 of 191 and the rest of the list
-- was arbitrary; the 发售年份 filter answered 22 games for 2026 and 0 for
-- 2026-08. Release dates are not immutable and, worse, a row minted after the
-- command was deleted never had one at all.
--
-- The mirror channel that already keeps content_limit in step now carries
-- release_date too, so the column has a keeper again. This adds the bookkeeping
-- the fill lane needs to converge: release_date IS NULL cannot mean "not
-- mirrored yet", because catalog legitimately has no date for a TBA work and
-- the lane would ask about those rows forever.
--
-- EXISTING ROWS START NULL ON PURPOSE. That is the repair: every local row is
-- unconfirmed, so the fill lane re-reads all of them (rate-capped, ~1,000 rows
-- a tick) and corrects the stale values along with the missing ones.

BEGIN;

ALTER TABLE galgame ADD COLUMN IF NOT EXISTS release_date_synced_at timestamptz;

COMMENT ON COLUMN galgame.release_date_synced_at IS
  'When the catalog mirror last wrote this row''s release_date (092). NULL = catalog has never confirmed it, which is what the fill lane looks for. Not user-visible.';

-- Drops to nothing once the sweep converges, which is exactly when a full-table
-- scan for the remaining stragglers would start costing the most.
CREATE INDEX IF NOT EXISTS idx_galgame_release_unsynced
  ON galgame (id) WHERE release_date_synced_at IS NULL;

COMMIT;
