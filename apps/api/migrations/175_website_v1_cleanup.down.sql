-- 175 down: restore the unconditional feed trigger. The trimmed text and the
-- added https:// are not reverted; neither carried meaning.
DROP TRIGGER IF EXISTS trg_feed_galgame_website ON galgame_website;
CREATE TRIGGER trg_feed_galgame_website
    AFTER INSERT OR DELETE OR UPDATE ON galgame_website
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_website();
