-- Run 223's down first: its feed_upsert writes this column.

ALTER TABLE feed_activity DROP COLUMN IF EXISTS access_scope;
