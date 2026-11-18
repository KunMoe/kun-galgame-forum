-- 231: config_secret_audit rows written before the fix carry 0001-01-01.
--
-- envconfig inserted the audit struct with a zero `created`, and GORM sends a
-- zero time instead of leaving the column to its DEFAULT, so every row of the
-- 2026-11-18 production rollout (12 imports at 08:42Z and two admin edits at
-- 13:56Z) reads year 1 in /admin/setting. The writes now stamp the time.
--
-- Existing rows get the closest time the table can prove: a set row takes its
-- variable's config_secret.updated, which is exact while each variable has
-- been set once since the import; every other zeroed row (the imports, and a
-- delete whose row is gone) takes the earliest config_secret.updated, which
-- is the import run. Rows already carrying a real time are not touched.

BEGIN;

UPDATE config_secret_audit a
   SET created = s.updated
  FROM config_secret s
 WHERE a.created < '1970-01-01'
   AND a.action = 'set'
   AND s.key = a.key;

UPDATE config_secret_audit
   SET created = (SELECT min(updated) FROM config_secret)
 WHERE created < '1970-01-01'
   AND EXISTS (SELECT 1 FROM config_secret);

COMMIT;
