-- Sections are listed as one flat list, and each can carry a header banner and
-- a set of mascot images.
--
--   * topic_section.position becomes one order across all sections. Existing
--     rows are renumbered in the order they were shown (group, then position
--     inside the group). Groups stay only to give a section the legacy
--     category the App still reads; nothing shows or orders by them.
--   * banner_image_hash / mascot_image_hashes: image-service hashes, named so
--     the reference ping keeps them alive. Existing sections get none.

BEGIN;

UPDATE topic_section s SET position = o.rn
FROM (
  SELECT ts.id, row_number() OVER (ORDER BY g.position, g.id, ts.position, ts.id) - 1 AS rn
  FROM topic_section ts JOIN topic_section_group g ON g.id = ts.group_id
) o
WHERE o.id = s.id AND s.position IS DISTINCT FROM o.rn;

ALTER TABLE topic_section
  ADD COLUMN IF NOT EXISTS banner_image_hash   text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS mascot_image_hashes jsonb NOT NULL DEFAULT '[]'::jsonb;

COMMIT;
