-- Drops the mirrored cooked forms of a rating's short review.

BEGIN;

ALTER TABLE galgame_rating
  DROP COLUMN IF EXISTS short_summary_html,
  DROP COLUMN IF EXISTS short_summary_text,
  DROP COLUMN IF EXISTS short_summary_has_spoiler;

COMMIT;
