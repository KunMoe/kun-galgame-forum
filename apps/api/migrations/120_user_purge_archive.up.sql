-- 120: an archive of every row a user-content purge changes, so undoing a
-- purge is one query instead of forensics.
--
-- On 2026-09-23 production purged two users by mistake (1484 and 104136) and
-- nothing recorded what went: 1484 came back from a dump, 104136 from dead
-- tuples, page free space and the trust scan's copies of resource text.
--
-- The purge's own statements are not the whole of it. DELETE FROM topic
-- cascades into other users' replies, comments, reactions, polls and
-- lotteries; three ON DELETE SET NULL keys rewrite other users' rows
-- (topic.best_answer_id, topic.pinned_reply_id, topic_comment.parent_comment_id,
-- 104 / 25 / 1,292 non-null rows on production); the feed_sync_* triggers
-- delete, insert and rewrite feed_activity rows; recounts, the website
-- hand-over and the todo release are UPDATEs. A list of tables would repeat
-- the purge's own history of covering half of what it touches, so capture is
-- a row trigger on every table:
--
-- * user_purge_capture() records the old row of a DELETE, the old row plus the
--   changed columns of an UPDATE, and the new row of an INSERT, but only in a
--   transaction that has set kungal.purge_id / kungal.purge_target_user_id /
--   kungal.purge_operator_id with set_config(..., true). The trigger's WHEN
--   clause tests the setting, so outside a purge no event is queued and
--   plpgsql is never entered. After a transaction that set it ends, the
--   setting reads back as '' rather than NULL, hence the coalesce.
-- * user_purge_archive_attach() puts the trigger on every table in public
--   except user_purge_archive (it would archive its own inserts) and
--   _migrations (the runner's bookkeeping). A migration that creates a table
--   calls it at the end; TestPurgeArchiveCoverage fails while a table lacks
--   the trigger and holds the same two exclusions.
-- * user_purge_restore(purge_id) undoes one purge; the recipe and its limits
--   are in docs/proj/api-v1/waves/u3c-purge.md §6.
--
-- Rows of one purge share created_at (now() is the transaction's start), so
-- the 30-day expiry cron never splits a purge.
--
-- Locks: CREATE TRIGGER takes SHARE ROW EXCLUSIVE on each table, and the
-- runner sends this file as one simple query, which Postgres runs as one
-- implicit transaction, so every table locked so far stays write-blocked
-- while the next one waits. lock_timeout makes a table that is busy fail the
-- migration within 10 s instead of stalling writes site-wide; a failed
-- migration keeps the new API from starting (depends_on
-- service_completed_successfully) and the old one keeps serving, so the
-- deploy is simply retried.
--
-- Existing rows: none are read or written. Running the file again replaces
-- the functions and attaches nothing new.

SET LOCAL lock_timeout = '10s';

CREATE TABLE IF NOT EXISTS user_purge_archive (
    id             bigserial   PRIMARY KEY,
    purge_id       uuid        NOT NULL,
    target_user_id integer     NOT NULL,
    operator_id    integer     NOT NULL,
    table_name     text        NOT NULL,
    operation      text        NOT NULL CHECK (operation IN ('delete', 'update', 'insert')),
    row_pk         jsonb       NOT NULL,
    row_data       jsonb       NOT NULL,
    new_values     jsonb,
    created_at     timestamptz NOT NULL DEFAULT now(),
    restored_at    timestamptz
);

CREATE INDEX IF NOT EXISTS idx_user_purge_archive_purge ON user_purge_archive (purge_id);
CREATE INDEX IF NOT EXISTS idx_user_purge_archive_target ON user_purge_archive (target_user_id);
CREATE INDEX IF NOT EXISTS idx_user_purge_archive_created ON user_purge_archive (created_at);

CREATE OR REPLACE FUNCTION user_purge_capture() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    v_row     jsonb;
    v_changed jsonb;
    v_pk      jsonb;
BEGIN
    IF TG_OP = 'INSERT' THEN
        v_row := to_jsonb(NEW);
    ELSE
        v_row := to_jsonb(OLD);
    END IF;
    IF TG_OP = 'UPDATE' THEN
        SELECT jsonb_object_agg(n.key, n.value) INTO v_changed
          FROM jsonb_each(to_jsonb(NEW)) n
         WHERE n.value IS DISTINCT FROM v_row -> n.key;
        IF v_changed IS NULL THEN
            RETURN NULL;
        END IF;
    END IF;
    SELECT jsonb_object_agg(a.attname, v_row -> a.attname) INTO v_pk
      FROM pg_index i
      JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY (i.indkey)
     WHERE i.indrelid = TG_RELID AND i.indisprimary;
    INSERT INTO public.user_purge_archive
        (purge_id, target_user_id, operator_id, table_name, operation, row_pk, row_data, new_values)
    VALUES (current_setting('kungal.purge_id')::uuid,
            current_setting('kungal.purge_target_user_id')::integer,
            current_setting('kungal.purge_operator_id')::integer,
            TG_TABLE_NAME, lower(TG_OP), v_pk, v_row, v_changed);
    RETURN NULL;
END $$;

CREATE OR REPLACE FUNCTION user_purge_archive_attach() RETURNS integer
LANGUAGE plpgsql AS $$
DECLARE
    t record;
    n integer := 0;
BEGIN
    FOR t IN
        SELECT c.relname
          FROM pg_class c
         WHERE c.relnamespace = 'public'::regnamespace
           AND c.relkind = 'r'
           AND NOT c.relispartition
           AND c.relname NOT IN ('user_purge_archive', '_migrations')
           AND NOT EXISTS (SELECT 1 FROM pg_trigger g
                            WHERE g.tgrelid = c.oid AND g.tgname = 'trg_user_purge_archive')
         ORDER BY c.relname
    LOOP
        EXECUTE format(
            'CREATE OR REPLACE TRIGGER trg_user_purge_archive
                 AFTER INSERT OR UPDATE OR DELETE ON public.%I
                 FOR EACH ROW
                 WHEN (coalesce(current_setting(''kungal.purge_id'', true), '''') <> '''')
                 EXECUTE FUNCTION user_purge_capture()', t.relname);
        n := n + 1;
    END LOOP;
    RETURN n;
END $$;

CREATE OR REPLACE FUNCTION user_purge_restore(p_purge_id uuid)
RETURNS TABLE (table_name text, operation text, archived bigint, restored bigint)
LANGUAGE plpgsql AS $$
#variable_conflict use_column
DECLARE
    r        record;
    k        text;
    fk       record;
    v_pk     text;
    v_pks    jsonb := '{}';
    v_n      bigint;
    v_done   boolean;
    v_report jsonb := '{}';
    v_key    text;
    v_role   text := current_setting('session_replication_role');
BEGIN
    PERFORM 1 FROM public.user_purge_archive a
      WHERE a.purge_id = p_purge_id AND a.restored_at IS NULL
      FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'user_purge_restore: purge % has no unrestored archive rows', p_purge_id;
    END IF;

    -- Triggers stay off: re-inserting a topic would make feed_sync_topic mint a
    -- second feed row next to the archived one. Foreign keys are off too, and
    -- checked explicitly below.
    PERFORM set_config('session_replication_role', 'replica', true);

    FOR r IN
        SELECT a.id, a.table_name AS tbl, a.operation AS op, a.row_pk, a.row_data, a.new_values
          FROM public.user_purge_archive a
         WHERE a.purge_id = p_purge_id AND a.restored_at IS NULL
         ORDER BY a.id DESC
    LOOP
        IF NOT v_pks ? r.tbl THEN
            SELECT string_agg(format('c.%1$I = (jsonb_populate_record(NULL::public.%2$I, $3)).%1$I', a.attname, r.tbl), ' AND ')
              INTO v_pk
              FROM pg_index i
              JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY (i.indkey)
             WHERE i.indrelid = format('public.%I', r.tbl)::regclass AND i.indisprimary;
            v_pks := v_pks || jsonb_build_object(r.tbl, v_pk);
        END IF;
        v_pk := v_pks ->> r.tbl;

        IF r.op = 'delete' THEN
            EXECUTE format(
                'INSERT INTO public.%1$I SELECT p.* FROM jsonb_populate_record(NULL::public.%1$I, $1) p
                 ON CONFLICT DO NOTHING', r.tbl)
              USING r.row_data;
            GET DIAGNOSTICS v_n = ROW_COUNT;
            v_done := v_n = 1;
        ELSIF r.op = 'insert' THEN
            EXECUTE format('DELETE FROM public.%1$I c WHERE %2$s', r.tbl, v_pk)
              USING NULL::jsonb, NULL::jsonb, r.row_pk;
            GET DIAGNOSTICS v_n = ROW_COUNT;
            v_done := v_n = 1;
        ELSE
            v_done := true;
            FOR k IN SELECT jsonb_object_keys(r.new_values) LOOP
                -- A counter gets back what the purge took off it, so likes
                -- added since the purge survive the restore.
                IF k ~ '_(count|sum)$'
                   AND jsonb_typeof(r.row_data -> k) = 'number'
                   AND jsonb_typeof(r.new_values -> k) = 'number' THEN
                    EXECUTE format(
                        'UPDATE public.%1$I c SET %2$I = c.%2$I
                             + (jsonb_populate_record(NULL::public.%1$I, $1)).%2$I
                             - (jsonb_populate_record(NULL::public.%1$I, $2)).%2$I
                          WHERE %3$s', r.tbl, k, v_pk)
                      USING r.row_data, r.new_values, r.row_pk;
                ELSE
                    EXECUTE format(
                        'UPDATE public.%1$I c SET %2$I = (jsonb_populate_record(NULL::public.%1$I, $1)).%2$I
                          WHERE %3$s
                            AND c.%2$I IS NOT DISTINCT FROM (jsonb_populate_record(NULL::public.%1$I, $2)).%2$I',
                        r.tbl, k, v_pk)
                      USING r.row_data, r.new_values, r.row_pk;
                END IF;
                GET DIAGNOSTICS v_n = ROW_COUNT;
                IF v_n = 0 THEN
                    v_done := false;
                END IF;
            END LOOP;
        END IF;

        v_key := r.tbl || '|' || r.op;
        v_report := jsonb_set(v_report, ARRAY[v_key], jsonb_build_array(
            coalesce((v_report -> v_key ->> 0)::bigint, 0) + 1,
            coalesce((v_report -> v_key ->> 1)::bigint, 0) + CASE WHEN v_done THEN 1 ELSE 0 END));
    END LOOP;

    PERFORM set_config('session_replication_role', v_role, true);

    FOR fk IN
        SELECT DISTINCT a.table_name AS child, c.conname,
               c.confrelid::regclass::text AS parent,
               (SELECT string_agg(format('x.%I IS NOT NULL', ca.attname), ' AND ' ORDER BY u.i)
                  FROM unnest(c.conkey) WITH ORDINALITY u(att, i)
                  JOIN pg_attribute ca ON ca.attrelid = c.conrelid AND ca.attnum = u.att) AS present,
               (SELECT string_agg(format('p.%I = x.%I', pa.attname, ca.attname), ' AND ' ORDER BY u.i)
                  FROM unnest(c.conkey, c.confkey) WITH ORDINALITY u(att, patt, i)
                  JOIN pg_attribute ca ON ca.attrelid = c.conrelid AND ca.attnum = u.att
                  JOIN pg_attribute pa ON pa.attrelid = c.confrelid AND pa.attnum = u.patt) AS matches
          FROM public.user_purge_archive a
          JOIN pg_constraint c ON c.conrelid = format('public.%I', a.table_name)::regclass AND c.contype = 'f'
         WHERE a.purge_id = p_purge_id AND a.restored_at IS NULL AND a.operation IN ('delete', 'update')
    LOOP
        SELECT string_agg(format('x.%1$I = (jsonb_populate_record(NULL::public.%2$I, a.row_pk)).%1$I', ka.attname, fk.child), ' AND ')
          INTO v_pk
          FROM pg_index i
          JOIN pg_attribute ka ON ka.attrelid = i.indrelid AND ka.attnum = ANY (i.indkey)
         WHERE i.indrelid = format('public.%I', fk.child)::regclass AND i.indisprimary;
        EXECUTE format(
            'SELECT count(*) FROM public.user_purge_archive a
               JOIN public.%1$I x ON %2$s
              WHERE a.purge_id = $1 AND a.restored_at IS NULL AND a.table_name = %1$L
                AND a.operation IN (''delete'', ''update'')
                AND %3$s
                AND NOT EXISTS (SELECT 1 FROM %4$s p WHERE %5$s)',
            fk.child, v_pk, fk.present, fk.parent, fk.matches)
          INTO v_n USING p_purge_id;
        IF v_n > 0 THEN
            RAISE EXCEPTION 'user_purge_restore: % restored % rows point at a missing % row (constraint %); nothing was restored',
                v_n, fk.child, fk.parent, fk.conname;
        END IF;
    END LOOP;

    UPDATE public.user_purge_archive a SET restored_at = now()
     WHERE a.purge_id = p_purge_id AND a.restored_at IS NULL;

    RETURN QUERY
        SELECT split_part(e.key, '|', 1), split_part(e.key, '|', 2),
               (e.value ->> 0)::bigint, (e.value ->> 1)::bigint
          FROM jsonb_each(v_report) e
         ORDER BY 1, 2;
END $$;

SELECT user_purge_archive_attach();
