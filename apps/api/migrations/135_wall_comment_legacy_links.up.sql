-- 135: rewrite the last notification links that point at a pre-community
-- galgame comment id.
--
-- A link of the form /galgame/<gid>?comment=<old id>&thread=<old id> told the
-- galgame page to translate the old comment id through
-- GET /api/galgame/:gid/comments/locate. The RC track retires that route; the
-- v1 page only understands ?comment=<community post id>.
--
-- Production on 2026-09-22: 64 such links, all type 'mentioned', all in exactly
-- this shape. 36 old ids are in galgame_comment_community_map and become the
-- post id. The other 28 were deleted before the community import and have no
-- post; their link drops the query and opens the galgame page itself.
--
-- Only rows still carrying thread= are touched, so running it again changes
-- nothing.

UPDATE message AS m
SET link = substring(m.link from '^(/galgame/[0-9]+)') || '?comment=' || map.post_id
FROM galgame_comment_community_map AS map
WHERE m.link ~ '^/galgame/[0-9]+\?comment=[0-9]+&thread=[0-9]+$'
  AND map.old_comment_id = substring(m.link from 'comment=([0-9]+)')::int;

UPDATE message
SET link = substring(link from '^(/galgame/[0-9]+)')
WHERE link ~ '^/galgame/[0-9]+\?comment=[0-9]+&thread=[0-9]+$';
