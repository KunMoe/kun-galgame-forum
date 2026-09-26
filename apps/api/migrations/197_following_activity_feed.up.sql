-- 197: the home page's 关注 tab reads feed_activity filtered to the accounts
-- the viewer follows, and shows a red dot while it holds anything newer than
-- the viewer last saw.
--
-- The existing keyset indexes lead with created or type, so an actor filter
-- scanned the whole table; this index leads with user_id in the same keyset
-- order. The seen mark is new, nullable, and null for every existing row: a
-- null mark counts only the last seven days (see the FA wave contract).
-- Additive: migrate before deploy, which the deploy already does.
CREATE INDEX IF NOT EXISTS idx_feed_activity_user_keyset
  ON feed_activity (user_id, created DESC, type DESC, source_id DESC);

ALTER TABLE kungal_user_state
  ADD COLUMN IF NOT EXISTS following_activity_seen_at timestamptz;
