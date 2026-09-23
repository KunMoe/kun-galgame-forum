-- 170 down: restore the old tokens. The trimmed version is not restored; the
-- leading space carried no meaning.
UPDATE update_log SET type = 'pref' WHERE type = 'perf';
UPDATE update_log SET type = 'styles' WHERE type = 'style';
