-- 120 down: detach the capture trigger from every table, then drop the
-- functions and the archive. Every archived purge becomes unrecoverable.
DO $$
DECLARE
    t record;
BEGIN
    FOR t IN
        SELECT g.tgrelid::regclass AS rel FROM pg_trigger g WHERE g.tgname = 'trg_user_purge_archive'
    LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS trg_user_purge_archive ON %s', t.rel);
    END LOOP;
END $$;

DROP FUNCTION IF EXISTS user_purge_restore(uuid);
DROP FUNCTION IF EXISTS user_purge_archive_attach();
DROP FUNCTION IF EXISTS user_purge_capture();
DROP TABLE IF EXISTS user_purge_archive;
