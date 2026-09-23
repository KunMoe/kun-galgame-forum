DROP TRIGGER IF EXISTS trg_feed_galgame_toolset_resource ON galgame_toolset_resource;
CREATE TRIGGER trg_feed_galgame_toolset_resource
    AFTER INSERT OR DELETE OR UPDATE ON galgame_toolset_resource
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_toolset_resource();

DROP TRIGGER IF EXISTS trg_feed_galgame_toolset ON galgame_toolset;
CREATE TRIGGER trg_feed_galgame_toolset
    AFTER INSERT OR DELETE OR UPDATE ON galgame_toolset
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_toolset();

DROP INDEX IF EXISTS galgame_toolset_resource_toolset_id_content_key;
CREATE UNIQUE INDEX IF NOT EXISTS galgame_toolset_resource_toolset_id_content_key
    ON galgame_toolset_resource (toolset_id, content);

DROP INDEX IF EXISTS galgame_toolset_practicality_toolset_id_user_id_key;

DROP TABLE IF EXISTS toolset_upload;
