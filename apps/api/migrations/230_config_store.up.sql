-- 230: the config store — environment variables and secrets read once at
-- startup, so they can leave the Dokploy panel.
--
--   * config_secret: one sealed value per variable (AES-256-GCM under
--     KUN_CONFIG_KEK, which itself never leaves the panel). The environment
--     still wins over a row, and a missing row falls back to the code default.
--   * config_secret_audit: every write and delete. A secret is recorded only by
--     its fingerprint and last characters, a plain value as itself.
--
-- New tables only, nothing seeded: without a KUN_CONFIG_KEK in the panel the
-- API never reads them. Additive; the deploy's migrate step runs it first.

BEGIN;

CREATE TABLE IF NOT EXISTS config_secret (
  key        text PRIMARY KEY CHECK (key ~ '^[A-Z][A-Z0-9_]*$'),
  cipher     bytea NOT NULL,
  updated_by integer,
  updated    timestamptz(3) NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS config_secret_audit (
  id        bigserial PRIMARY KEY,
  key       text NOT NULL,
  action    text NOT NULL CHECK (action IN ('set', 'delete', 'import')),
  old_shown text NOT NULL DEFAULT '',
  new_shown text NOT NULL DEFAULT '',
  note      text NOT NULL DEFAULT '',
  user_id   integer,
  created   timestamptz(3) NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS config_secret_audit_created_idx ON config_secret_audit (created DESC, id DESC);

COMMIT;
