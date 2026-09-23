-- 142: the toolset tables for /api/v1/toolsets (G1, docs/proj/api-v1/waves/g1-toolsets.md).
--
-- Production on 2026-09-23 (119 toolsets, 110 resources, 61 practicality rows):
--   * Upload sessions lived only in artifact, so complete / resume / abort
--     could not tell whose session an artifact_uuid was. toolset_upload records
--     the owner and the toolset at init, and completed_at is set once, in the
--     same transaction as the daily quota, replacing a Redis SETNX that counted
--     the upload twice whenever Redis errored.
--   * galgame_toolset_practicality had no (toolset_id, user_id) key; the write
--     was find-then-insert, so two concurrent ratings could both land. 0
--     duplicate pairs exist, so the unique index builds.
--   * The (toolset_id, content) unique index refused a second file resource on
--     one toolset, because file resources keep content empty. It becomes
--     partial on content <> ''. No toolset has two empty-content rows and no
--     non-empty pair repeats, so the new index builds.
--   * Both feed triggers fired on every UPDATE, so each detail view and each
--     download also rewrote feed_activity. The trigger functions are unchanged;
--     the triggers now fire only when a column the function reads changes
--     (toolset: name, user_id, status, created; resource: note, content,
--     user_id, toolset_id, created).
--
-- Every statement is IF NOT EXISTS or drop-and-recreate, so running it again
-- changes nothing.

CREATE TABLE IF NOT EXISTS toolset_upload (
    artifact_uuid varchar(36)  PRIMARY KEY,
    toolset_id    integer      NOT NULL REFERENCES galgame_toolset (id) ON UPDATE CASCADE ON DELETE CASCADE,
    user_id       integer      NOT NULL,
    filename      varchar(1007) NOT NULL,
    file_size     bigint       NOT NULL,
    created       timestamptz  NOT NULL DEFAULT now(),
    completed_at  timestamptz
);
CREATE INDEX IF NOT EXISTS idx_toolset_upload_toolset_id ON toolset_upload (toolset_id);
CREATE INDEX IF NOT EXISTS idx_toolset_upload_user_id ON toolset_upload (user_id);

CREATE UNIQUE INDEX IF NOT EXISTS galgame_toolset_practicality_toolset_id_user_id_key
    ON galgame_toolset_practicality (toolset_id, user_id);

DROP INDEX IF EXISTS galgame_toolset_resource_toolset_id_content_key;
CREATE UNIQUE INDEX IF NOT EXISTS galgame_toolset_resource_toolset_id_content_key
    ON galgame_toolset_resource (toolset_id, content)
    WHERE content <> '';

DROP TRIGGER IF EXISTS trg_feed_galgame_toolset ON galgame_toolset;
CREATE TRIGGER trg_feed_galgame_toolset
    AFTER INSERT OR DELETE OR UPDATE OF name, user_id, status, created ON galgame_toolset
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_toolset();

DROP TRIGGER IF EXISTS trg_feed_galgame_toolset_resource ON galgame_toolset_resource;
CREATE TRIGGER trg_feed_galgame_toolset_resource
    AFTER INSERT OR DELETE OR UPDATE OF note, content, user_id, toolset_id, created ON galgame_toolset_resource
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_toolset_resource();
