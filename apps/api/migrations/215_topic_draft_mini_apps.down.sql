ALTER TABLE topic_draft
  DROP COLUMN IF EXISTS prize_image_hashes,
  DROP COLUMN IF EXISTS lotteries,
  DROP COLUMN IF EXISTS polls;
