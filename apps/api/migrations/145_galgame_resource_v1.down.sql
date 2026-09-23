DROP TRIGGER IF EXISTS trg_feed_galgame_resource ON galgame_resource;
CREATE TRIGGER trg_feed_galgame_resource
    AFTER INSERT OR DELETE OR UPDATE ON galgame_resource
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_resource();

DROP INDEX IF EXISTS idx_galgame_resource_created_id;
