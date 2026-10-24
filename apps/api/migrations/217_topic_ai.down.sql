BEGIN;

ALTER TABLE topic
  DROP COLUMN IF EXISTS ai_summary,
  DROP COLUMN IF EXISTS ai_summary_model,
  DROP COLUMN IF EXISTS ai_summary_at,
  DROP COLUMN IF EXISTS ai_summary_drift;

DROP TABLE IF EXISTS topic_section_request;
DROP TABLE IF EXISTS topic_classification;
DROP TABLE IF EXISTS site_setting;

COMMIT;
