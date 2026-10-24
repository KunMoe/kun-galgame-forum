-- 215: a topic draft keeps the polls and lotteries attached to it.
--
-- Since 2026-10-24 the topic editor lets an author attach polls and lotteries
-- before the topic exists; the web creates them on the new topic right after
-- createTopic. A draft saved from that editor has to carry them, or loading it
-- back silently drops a whole lottery.
--
-- polls and lotteries hold the draft shapes of TopicDraftPoll and
-- TopicDraftLottery as JSON arrays. A draft never holds redemption codes: the
-- type has no field for them, and the author enters them again before
-- publishing.
--
-- prize_image_hashes is every prize image hash of the draft's lotteries, flat.
-- It exists for cron/reference_ping.go, which pings jsonb arrays named
-- *image_hashes and would not look inside lotteries, so drafted prize images
-- would otherwise age towards the image service's soft delete.
--
-- Existing drafts get empty arrays. Additive: migrate before or with the deploy.

ALTER TABLE topic_draft
  ADD COLUMN IF NOT EXISTS polls              jsonb NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS lotteries          jsonb NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS prize_image_hashes jsonb NOT NULL DEFAULT '[]'::jsonb;
