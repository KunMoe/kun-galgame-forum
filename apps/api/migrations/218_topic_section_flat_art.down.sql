-- The flat order is kept: positions still sort correctly inside each group.

BEGIN;

ALTER TABLE topic_section
  DROP COLUMN IF EXISTS banner_image_hash,
  DROP COLUMN IF EXISTS mascot_image_hashes;

COMMIT;
