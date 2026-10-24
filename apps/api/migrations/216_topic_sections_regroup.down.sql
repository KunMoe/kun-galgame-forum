-- Restores the 27 prefix-coded sections and each topic's old filing from
-- topic_section_legacy_relation. Topics created after 216 have no legacy row:
-- they get the old section their new one came from first, so every topic
-- keeps a section. topic.category is rewritten from the restored prefix.
-- Drafts are not rolled back: a draft saved under a new key loads without a
-- section and the author picks one again.

BEGIN;

DROP INDEX IF EXISTS topic_section_relation_one_per_topic;

INSERT INTO topic_section (name, updated)
SELECT v.name, CURRENT_TIMESTAMP
FROM (VALUES
  ('g-walkthrough'), ('g-chatting'), ('g-article'), ('g-seeking'), ('g-news'), ('g-releases'), ('g-other'),
  ('t-crack'), ('t-web'), ('t-languages'), ('t-help'), ('t-linux'), ('t-practical'), ('t-ai'),
  ('t-android'), ('t-adobe'), ('t-algorithm'), ('t-other'),
  ('o-anime'), ('o-comics'), ('o-music'), ('o-novel'), ('o-daily'), ('o-essay'), ('o-forum'), ('o-patch'), ('o-other')
) AS v(name)
ON CONFLICT (name) DO NOTHING;

CREATE TEMP TABLE section_fallback (new_name text PRIMARY KEY, old_name text NOT NULL) ON COMMIT DROP;
INSERT INTO section_fallback VALUES
  ('galgame-discussion', 'g-chatting'), ('galgame-walkthrough', 'g-walkthrough'), ('review', 'g-article'),
  ('news', 'g-news'), ('resource-seeking', 'g-seeking'), ('resource-release', 'g-other'),
  ('tech-help', 't-help'), ('game-running', 't-practical'), ('development', 't-other'),
  ('acgn', 'o-anime'), ('daily', 'o-daily'), ('site', 'o-forum');

CREATE TEMP TABLE topic_restore ON COMMIT DROP AS
SELECT l.topic_id, l.name, l.position FROM topic_section_legacy_relation l
UNION ALL
SELECT r.topic_id, f.old_name, 0
FROM topic_section_relation r
JOIN topic_section s ON s.id = r.topic_section_id
JOIN section_fallback f ON f.new_name = s.name
WHERE NOT EXISTS (SELECT 1 FROM topic_section_legacy_relation l WHERE l.topic_id = r.topic_id);

UPDATE user_purge_archive a
SET row_data = jsonb_set(a.row_data, '{topic_section_id}', to_jsonb(os.id)),
    row_pk = CASE WHEN a.row_pk ? 'topic_section_id' THEN jsonb_set(a.row_pk, '{topic_section_id}', to_jsonb(os.id)) ELSE a.row_pk END
FROM topic_section ns, section_fallback f, topic_section os
WHERE a.table_name = 'topic_section_relation'
  AND ns.id = (a.row_data ->> 'topic_section_id')::int
  AND f.new_name = ns.name AND os.name = f.old_name;

DELETE FROM topic_section_relation r
USING topic_section s, section_fallback f
WHERE s.id = r.topic_section_id AND f.new_name = s.name;

INSERT INTO topic_section_relation (topic_id, topic_section_id, position, updated)
SELECT x.topic_id, s.id, x.position, CURRENT_TIMESTAMP
FROM topic_restore x JOIN topic_section s ON s.name = x.name
ON CONFLICT DO NOTHING;

UPDATE topic t SET category = CASE left(s.name, 2) WHEN 'g-' THEN 'galgame' WHEN 't-' THEN 'technique' ELSE 'others' END
FROM topic_section_relation r JOIN topic_section s ON s.id = r.topic_section_id
WHERE r.topic_id = t.id AND r.position = 0;

DELETE FROM topic_section s USING section_fallback f WHERE f.new_name = s.name;

DROP TABLE IF EXISTS topic_section_legacy_relation;

ALTER TABLE topic_section
  DROP COLUMN IF EXISTS group_id,
  DROP COLUMN IF EXISTS label,
  DROP COLUMN IF EXISTS description,
  DROP COLUMN IF EXISTS icon,
  DROP COLUMN IF EXISTS color,
  DROP COLUMN IF EXISTS position,
  DROP COLUMN IF EXISTS is_archived,
  DROP COLUMN IF EXISTS costs_moemoepoint,
  DROP COLUMN IF EXISTS hidden_from_feeds;

DROP TABLE IF EXISTS topic_section_group;

DROP INDEX IF EXISTS topic_section_name_key;

COMMIT;
