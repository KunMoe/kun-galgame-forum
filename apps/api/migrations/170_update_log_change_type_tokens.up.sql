-- 170: normalize the update_log change-type tokens and one version string.
--
-- /api/v1/update-logs exposes update_log.type as the closed vocabulary
-- change_type. Two stored spellings were not what they meant: 'pref' is a typo
-- of 'perf' (性能优化) and 'styles' was the only plural token. One version was
-- stored with a leading space; the v1 write path trims, so the stored value is
-- trimmed to match.
--
-- Production on 2026-09-23: 28 'pref' rows, 32 'styles' rows, 1 version with
-- surrounding whitespace, out of 1234. No other table stores these tokens; the
-- home feed (feed_activity) copies only the content, and the update_log
-- trigger refreshes it on UPDATE without changing anything visible.
--
-- Every statement matches only rows still in the old shape, so running it
-- again changes nothing.

UPDATE update_log SET type = 'perf' WHERE type = 'pref';
UPDATE update_log SET type = 'style' WHERE type = 'styles';
UPDATE update_log SET version = btrim(version) WHERE version <> btrim(version);
