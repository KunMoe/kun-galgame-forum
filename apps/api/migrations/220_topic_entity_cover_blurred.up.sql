-- The author can ask that every entity card cover on the topic page, and every
-- cover in the preview panel those cards open, be shown blurred, whatever the
-- reader's own NSFW stance (a reader who hides NSFW still hides it). It exists
-- because some SFW works carry nude covers that no sexual grade marks.
-- Existing topics and drafts get false: nothing changes for them.

BEGIN;

ALTER TABLE topic
  ADD COLUMN IF NOT EXISTS is_entity_cover_blurred boolean NOT NULL DEFAULT false;

ALTER TABLE topic_draft
  ADD COLUMN IF NOT EXISTS is_entity_cover_blurred boolean NOT NULL DEFAULT false;

COMMIT;
