-- Rewrite the legacy /galgame-resource/:id links stored in notifications and in
-- the activity feed to the canonical /galgame/resource/:id.
--
-- /galgame-resource/:id has never been a page in the Nuxt app. It survived only
-- because of a compatibility route rule that redirects it to /galgame/resource/**.
-- Nitro expands the ** on a full page load, but Nuxt's client-side route-rules
-- middleware returns the redirect target verbatim, so tapping one of these links
-- inside the app navigated to the literal /galgame/resource/** and the API
-- answered "无效的资源 ID". Measured before this ran: 136 message rows and 127
-- feed_activity rows.
--
-- Idempotent: the predicate no longer matches once a row is rewritten.

UPDATE message
SET link = '/galgame/resource/' || substring(link from '^/galgame-resource/([0-9]+)$')
WHERE link ~ '^/galgame-resource/[0-9]+$';

UPDATE feed_activity
SET link = '/galgame/resource/' || substring(link from '^/galgame-resource/([0-9]+)$')
WHERE link ~ '^/galgame-resource/[0-9]+$';
