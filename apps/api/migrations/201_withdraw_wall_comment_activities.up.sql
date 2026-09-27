-- 201: wave 13 D5 cutover. Community projects wall comments into the feed
-- itself (community:post:<id>) once kungal's community_site_config row exists,
-- so the forum stops pushing its six wall-comment activity types and withdraws
-- what it pushed of them. Every accepted, not yet removed key of those types
-- is queued; the drainer tombstones a queued key whose type it no longer
-- pushes. The 04:30 reconcile would tombstone the same keys as orphans, but
-- until then each comment would show twice in the feed.
--
-- Production on 2026-09-27: 10,917 live keys (galgame 10,224, rating 230,
-- resource 217, website 117, toolset 74, quiz 55). The feed_activity rows stay:
-- the forum's own feeds still read them. Running it again re-queues only keys
-- community still holds live.

INSERT INTO activity_push_queue (type, source_id)
SELECT type, source_id FROM activity_push_sent
WHERE NOT removed
  AND type IN (
    'GALGAME_COMMENT_CREATION',
    'GALGAME_RESOURCE_COMMENT_CREATION',
    'GALGAME_RATING_COMMENT_CREATION',
    'GALGAME_QUIZ_COMMENT_CREATION',
    'GALGAME_WEBSITE_COMMENT_CREATION',
    'TOOLSET_COMMENT_CREATION'
  )
ON CONFLICT (type, source_id) DO UPDATE SET enqueued = clock_timestamp();
