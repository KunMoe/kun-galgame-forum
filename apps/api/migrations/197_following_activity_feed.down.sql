ALTER TABLE kungal_user_state DROP COLUMN IF EXISTS following_activity_seen_at;
DROP INDEX IF EXISTS idx_feed_activity_user_keyset;
