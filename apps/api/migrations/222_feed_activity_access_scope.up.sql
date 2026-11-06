-- 222: feed_activity carries the access scope of the topic an activity hangs
-- off; every other activity is 'public'. Readers filter on it with
-- topic/repository.SharedListPredicate, like every topic list.
--
-- Existing rows: all of them are public, because 090 deletes every activity of
-- a non-public topic, so the default is their true value. Nothing writes
-- 'login' until 223 changes the triggers; 223 is held back from the default
-- run and applied by hand once the code that filters on this column is live
-- (see 223's header). Additive: runs with the deploy.

ALTER TABLE feed_activity ADD COLUMN IF NOT EXISTS access_scope text NOT NULL DEFAULT 'public';
