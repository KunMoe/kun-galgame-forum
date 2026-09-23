-- 141: galgame.id is the catalog work id; the column that points at it is work_id.
--
-- cmd/align-galgame-ids renumbered every galgame.id onto catalog_work.id in the
-- G0 window (docs/proj/gid-is-work-id.md). From here on the forum holds no
-- mapping: the number in /galgame/:id, in every *_work_id column and in every
-- call to catalog is the same number. "gid" named the retired forum-only
-- numbering; galgame_id read as "the forum's own id" and was confused with the
-- catalog's often enough (148,548 folder items went to catalog under the wrong
-- work), so both names go.
--
-- DEPLOY-COUPLED: the renames break the binary that was built before them. This
-- runs only inside the G0 window, with kungal-api stopped, after the renumber
-- committed — the migrate container applies it before the G0b API starts.
--
-- Also here:
--   * galgame_quiz.galgame_id is dropped. 047 moved quizzes onto
--     galgame_quiz_galgame and promised to drop it; every row is NULL.
--   * galgame_redirect is dropped: merged-away ids are not redirected
--     (user ruling 2026-09-23). Folding a merged page's data stays.
--   * galgame.id loses its sequence: a local row is created only under a
--     catalog work id, so an insert that forgets the id must fail, not mint a
--     number no catalog work has.
--   * feed_upsert's p_gid becomes p_work_id. A parameter cannot be renamed by
--     CREATE OR REPLACE, so it is dropped and recreated; the trigger functions
--     resolve it by name at call time.

BEGIN;

ALTER TABLE galgame_resource              RENAME COLUMN galgame_id TO work_id;
ALTER TABLE galgame_like                  RENAME COLUMN galgame_id TO work_id;
ALTER TABLE galgame_favorite              RENAME COLUMN galgame_id TO work_id;
ALTER TABLE galgame_rating                RENAME COLUMN galgame_id TO work_id;
ALTER TABLE galgame_activity              RENAME COLUMN galgame_id TO work_id;
ALTER TABLE galgame_quiz_galgame          RENAME COLUMN galgame_id TO work_id;
ALTER TABLE galgame_contributor           RENAME COLUMN galgame_id TO work_id;
ALTER TABLE galgame_comment_community_map RENAME COLUMN galgame_id TO work_id;
ALTER TABLE galgame_collection_item       RENAME COLUMN galgame_id TO work_id;
ALTER TABLE feed_activity                 RENAME COLUMN galgame_id TO work_id;
ALTER TABLE galgame_merge_discarded       RENAME COLUMN old_gid TO old_work_id;
ALTER TABLE galgame_merge_discarded       RENAME COLUMN new_gid TO new_work_id;

ALTER TABLE galgame_quiz DROP COLUMN IF EXISTS galgame_id;

DROP TABLE IF EXISTS galgame_redirect;

ALTER TABLE galgame ALTER COLUMN id DROP DEFAULT;
DROP SEQUENCE IF EXISTS galgame_id_seq;

DROP FUNCTION IF EXISTS feed_upsert(text, integer, integer, integer, text, text, boolean, timestamp with time zone);
CREATE FUNCTION feed_upsert(p_type text, p_sid integer, p_uid integer, p_work_id integer, p_content text, p_link text, p_nsfw boolean, p_created timestamp with time zone)
 RETURNS void
 LANGUAGE sql
AS $function$
    INSERT INTO feed_activity (type, source_id, user_id, work_id, content, link, is_nsfw, created)
    VALUES (p_type, p_sid, p_uid, p_work_id, p_content, p_link, p_nsfw, p_created)
    ON CONFLICT (type, source_id) DO UPDATE SET
        user_id = EXCLUDED.user_id, work_id = EXCLUDED.work_id,
        content = EXCLUDED.content, link = EXCLUDED.link,
        is_nsfw = EXCLUDED.is_nsfw, created = EXCLUDED.created;
$function$;

CREATE OR REPLACE FUNCTION feed_sync_galgame_activity()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete(OLD.type, OLD.id); RETURN OLD; END IF;
    IF NEW.type IN ('GALGAME_EDIT', 'GALGAME_PR_CREATION') THEN
        PERFORM feed_upsert(NEW.type, NEW.id, NEW.user_id, NEW.work_id, '', '/galgame/' || NEW.work_id, false, NEW.created);
    END IF;
    RETURN NEW;
END $function$;

CREATE OR REPLACE FUNCTION feed_sync_galgame_rating()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('GALGAME_RATING_CREATION', OLD.id); RETURN OLD; END IF;
    PERFORM feed_upsert('GALGAME_RATING_CREATION', NEW.id, NEW.user_id, NEW.work_id,
        CASE WHEN NEW.spoiler_level <> 'none' THEN '⚠️ 该评分可能含有剧透内容，点进查看'
             ELSE SUBSTRING(COALESCE(NEW.short_summary, ''), 1, 100) END,
        '/galgame-rating/' || NEW.id, false, NEW.created);
    RETURN NEW;
END $function$;

CREATE OR REPLACE FUNCTION feed_sync_galgame_resource()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('GALGAME_RESOURCE_CREATION', OLD.id); RETURN OLD; END IF;
    PERFORM feed_upsert('GALGAME_RESOURCE_CREATION', NEW.id, NEW.user_id, NEW.work_id, '', '/galgame/' || NEW.work_id, false, NEW.created);
    RETURN NEW;
END $function$;

COMMIT;
