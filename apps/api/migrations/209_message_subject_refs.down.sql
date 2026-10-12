-- 209 down: drop the notification references. The links still carry what the
-- backfill read them from; anything written since is reachable only by link.
DROP FUNCTION IF EXISTS message_backfill_refs();
DROP INDEX IF EXISTS idx_message_root;
DROP INDEX IF EXISTS idx_message_subject;
ALTER TABLE message
    DROP COLUMN IF EXISTS is_suppressed,
    DROP COLUMN IF EXISTS root_id,
    DROP COLUMN IF EXISTS root_kind,
    DROP COLUMN IF EXISTS subject_id,
    DROP COLUMN IF EXISTS subject_kind;
