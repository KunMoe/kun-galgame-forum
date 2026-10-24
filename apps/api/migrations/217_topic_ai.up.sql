-- Model-decided topic sections, author requests to move a topic, AI
-- summaries, and the settings center that switches them on.
--
--   * site_setting: admin-edited values; a missing row means the code default.
--     Nothing is seeded, so the model stays off until an admin turns it on.
--   * topic_classification: the model's verdict per topic. source fallback
--     marks a topic published while the model was unreachable; a cron retries
--     it and files a staff request when the verdict disagrees.
--   * topic_section_request: an author's (or that cron's) request to move a
--     topic; at most one pending per topic.
--   * topic.ai_summary*: the summary and how much the body changed since it
--     was written (runes inserted plus deleted). Existing topics get NULL and
--     0; the backfill tool writes their summaries.
--
-- Additive: migrate before or with the deploy.

BEGIN;

CREATE TABLE IF NOT EXISTS site_setting (
  key        text PRIMARY KEY,
  value      jsonb NOT NULL,
  updated_by integer,
  updated    timestamptz(3) NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS topic_classification (
  topic_id integer PRIMARY KEY REFERENCES topic(id) ON DELETE CASCADE,
  section  text NOT NULL,
  source   text NOT NULL CHECK (source IN ('model', 'fallback', 'backfill')),
  model    text NOT NULL DEFAULT '',
  reason   text NOT NULL DEFAULT '',
  created  timestamptz(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated  timestamptz(3) NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS topic_classification_fallback_idx
  ON topic_classification (created) WHERE source = 'fallback';

CREATE TABLE IF NOT EXISTS topic_section_request (
  id           serial PRIMARY KEY,
  topic_id     integer NOT NULL REFERENCES topic(id) ON DELETE CASCADE,
  requested_by integer,
  from_section text NOT NULL,
  to_section   text NOT NULL,
  reason       text NOT NULL DEFAULT '',
  state        text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'approved', 'rejected')),
  decided_by   integer,
  decided_at   timestamptz(3),
  created      timestamptz(3) NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS topic_section_request_one_pending
  ON topic_section_request (topic_id) WHERE state = 'pending';
CREATE INDEX IF NOT EXISTS topic_section_request_state_idx
  ON topic_section_request (state, created DESC);

ALTER TABLE topic
  ADD COLUMN IF NOT EXISTS ai_summary       text,
  ADD COLUMN IF NOT EXISTS ai_summary_model text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS ai_summary_at    timestamptz(3),
  ADD COLUMN IF NOT EXISTS ai_summary_drift integer NOT NULL DEFAULT 0;

SELECT user_purge_archive_attach();

COMMIT;
