-- 211 down: play_status comes back empty and nullable. The rating_cutover
-- snapshot is gone for good; infra's import ledger is the record of it.
ALTER TABLE galgame_rating ADD COLUMN IF NOT EXISTS play_status text;
