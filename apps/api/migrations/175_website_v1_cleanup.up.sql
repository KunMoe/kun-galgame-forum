-- 175: tidy galgame_website for /api/v1/websites and stop page views from
-- rewriting the home feed.
--
-- Production on 2026-09-23 (87 sites):
--   * 5 legacy icon URLs and 5 create_time values start with a space. v1 sends
--     the icon as external_icon_url with format uri, and founded as shown text.
--   * domain holds every address of a site; 170 of 171 elements carry a
--     scheme, one is the bare host "hacg.icu" (site 9). v1 sends them as urls
--     with format uri, so the bare one gets https://.
--   * trg_feed_galgame_website fired on every UPDATE, so each detail view (the
--     view counter) and each like or favorite also rewrote feed_activity. The
--     feed row reads only name, url, age_limit, user_id and created; the
--     trigger now fires only when one of those changes.
--
-- Every statement matches only rows still in the old shape, and the trigger is
-- dropped and recreated, so running it again changes nothing.

UPDATE galgame_website SET icon = btrim(icon) WHERE icon <> btrim(icon);
UPDATE galgame_website SET create_time = btrim(create_time) WHERE create_time <> btrim(create_time);

UPDATE galgame_website AS w
SET domain = (
    SELECT jsonb_agg(
        CASE WHEN e.value ~* '^https?://' THEN to_jsonb(e.value) ELSE to_jsonb('https://' || e.value) END
        ORDER BY e.ordinality
    )
    FROM jsonb_array_elements_text(w.domain) WITH ORDINALITY AS e(value, ordinality)
)
WHERE jsonb_typeof(w.domain) = 'array'
  AND EXISTS (SELECT 1 FROM jsonb_array_elements_text(w.domain) AS e(value) WHERE e.value !~* '^https?://');

DROP TRIGGER IF EXISTS trg_feed_galgame_website ON galgame_website;
CREATE TRIGGER trg_feed_galgame_website
    AFTER INSERT OR DELETE OR UPDATE OF name, url, age_limit, user_id, created ON galgame_website
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_website();
