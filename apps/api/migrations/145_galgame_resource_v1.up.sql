-- 145: the galgame resource tables for /api/v1/galgame-resources (G3,
-- docs/proj/api-v1/waves/g3-resources.md).
--
-- Production on 2026-09-23 (50,377 resources, 51,765 links):
--   * Six links lost the first letters of their scheme when they were
--     pasted, so the v1 downloadlink pattern would reject them on read:
--     'ttps://' on resources 390, 13947, 18001, 35234 and 35100, 'tps://' on
--     36049. Each is rewritten to 'https://'. None of the six resources
--     already holds the corrected URL, so the (galgame_resource_id, url) key
--     stays unique. No other link changes.
--   * The browse order is created DESC, id DESC; idx_galgame_resource_created
--     covers created alone, so ties had no index order.
--   * The feed trigger fired on every UPDATE, so each view, download, like and
--     expiry report also rewrote feed_activity. The function is unchanged; the
--     trigger fires only when a column it reads changes (user_id, work_id,
--     created).
--
-- The rewrites match only the truncated schemes and every other statement is
-- IF NOT EXISTS or drop-and-recreate, so running it again changes nothing.

UPDATE galgame_resource_link SET url = 'h' || url WHERE url LIKE 'ttps://%';
UPDATE galgame_resource_link SET url = 'ht' || url WHERE url LIKE 'tps://%';

CREATE INDEX IF NOT EXISTS idx_galgame_resource_created_id
    ON galgame_resource (created DESC, id DESC);

DROP TRIGGER IF EXISTS trg_feed_galgame_resource ON galgame_resource;
CREATE TRIGGER trg_feed_galgame_resource
    AFTER INSERT OR DELETE OR UPDATE OF user_id, work_id, created
    ON galgame_resource
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_resource();
