-- 176: point feed cards and notifications at the current host of a renamed
-- website.
--
-- A link to a website page is stored as /website/<host> when the card or the
-- notification is written. Three sites were later renamed, and nothing
-- rewrote those links; production on 2026-09-23 had 8 dead feed cards
-- (GALGAME_WEBSITE_COMMENT_CREATION) and 5 dead notifications ('commented').
-- Each old host was traced to its site through the comment behind the row,
-- which lives on that site's wall in kun_community:
--   hacg.onl         -> website 9  (hacg.icu; its comments discuss the swap
--                       from the pirate hacg.onl, and the admin's reply says
--                       it was changed)
--   res.nyne.dev     -> website 37 (nysoure.com)
--   www.nysoure.com  -> website 37 (nysoure.com)
--   www.touchgal.top -> website 43 (www.touchgal.ink)
-- Links are rewritten to the site's current host, so a rename between now and
-- the deploy is still followed. Renames made through /api/v1 rewrite these
-- links themselves from now on.
--
-- Only rows still pointing at an old host match, so running it again changes
-- nothing.

WITH renamed(old_host, website_id) AS (
    VALUES ('hacg.onl', 9), ('res.nyne.dev', 37), ('www.nysoure.com', 37), ('www.touchgal.top', 43)
)
UPDATE feed_activity AS f
SET link = '/website/' || w.url
FROM renamed AS r JOIN galgame_website AS w ON w.id = r.website_id
WHERE f.link = '/website/' || r.old_host;

WITH renamed(old_host, website_id) AS (
    VALUES ('hacg.onl', 9), ('res.nyne.dev', 37), ('www.nysoure.com', 37), ('www.touchgal.top', 43)
)
UPDATE message AS m
SET link = '/website/' || w.url
FROM renamed AS r JOIN galgame_website AS w ON w.id = r.website_id
WHERE m.link = '/website/' || r.old_host;
