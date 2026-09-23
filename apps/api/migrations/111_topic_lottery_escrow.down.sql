DROP INDEX IF EXISTS idx_topic_lottery_entry_lottery_created_id;

ALTER TABLE topic_lottery DROP COLUMN IF EXISTS point_escrow;
