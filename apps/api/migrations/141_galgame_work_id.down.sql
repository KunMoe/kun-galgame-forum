-- Restores the names and objects 141 removed. It does not restore data: the
-- renumber that preceded 141 is not reversed here (galgame_renumber_2026 holds
-- old_id -> new_id if that is ever needed), galgame_redirect comes back empty,
-- and galgame_quiz.galgame_id comes back NULL, as it was.

BEGIN;

ALTER TABLE galgame_resource              RENAME COLUMN work_id TO galgame_id;
ALTER TABLE galgame_like                  RENAME COLUMN work_id TO galgame_id;
ALTER TABLE galgame_favorite              RENAME COLUMN work_id TO galgame_id;
ALTER TABLE galgame_rating                RENAME COLUMN work_id TO galgame_id;
ALTER TABLE galgame_activity              RENAME COLUMN work_id TO galgame_id;
ALTER TABLE galgame_quiz_galgame          RENAME COLUMN work_id TO galgame_id;
ALTER TABLE galgame_contributor           RENAME COLUMN work_id TO galgame_id;
ALTER TABLE galgame_comment_community_map RENAME COLUMN work_id TO galgame_id;
ALTER TABLE galgame_collection_item       RENAME COLUMN work_id TO galgame_id;
ALTER TABLE feed_activity                 RENAME COLUMN work_id TO galgame_id;
ALTER TABLE galgame_merge_discarded       RENAME COLUMN old_work_id TO old_gid;
ALTER TABLE galgame_merge_discarded       RENAME COLUMN new_work_id TO new_gid;

ALTER TABLE galgame_quiz ADD COLUMN IF NOT EXISTS galgame_id integer;

CREATE TABLE IF NOT EXISTS galgame_redirect (
  old_gid integer     PRIMARY KEY,
  new_gid integer     NOT NULL,
  created timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_galgame_redirect_new ON galgame_redirect (new_gid);

CREATE SEQUENCE IF NOT EXISTS galgame_id_seq OWNED BY galgame.id;
SELECT setval('galgame_id_seq', 2000000000, false);
ALTER TABLE galgame ALTER COLUMN id SET DEFAULT nextval('galgame_id_seq');

DROP FUNCTION IF EXISTS feed_upsert(text, integer, integer, integer, text, text, boolean, timestamp with time zone);
CREATE FUNCTION feed_upsert(p_type text, p_sid integer, p_uid integer, p_gid integer, p_content text, p_link text, p_nsfw boolean, p_created timestamp with time zone)
 RETURNS void
 LANGUAGE sql
AS $function$
    INSERT INTO feed_activity (type, source_id, user_id, galgame_id, content, link, is_nsfw, created)
    VALUES (p_type, p_sid, p_uid, p_gid, p_content, p_link, p_nsfw, p_created)
    ON CONFLICT (type, source_id) DO UPDATE SET
        user_id = EXCLUDED.user_id, galgame_id = EXCLUDED.galgame_id,
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
        PERFORM feed_upsert(NEW.type, NEW.id, NEW.user_id, NEW.galgame_id, '', '/galgame/' || NEW.galgame_id, false, NEW.created);
    END IF;
    RETURN NEW;
END $function$;

CREATE OR REPLACE FUNCTION feed_sync_galgame_rating()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('GALGAME_RATING_CREATION', OLD.id); RETURN OLD; END IF;
    PERFORM feed_upsert('GALGAME_RATING_CREATION', NEW.id, NEW.user_id, NEW.galgame_id,
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
    PERFORM feed_upsert('GALGAME_RESOURCE_CREATION', NEW.id, NEW.user_id, NEW.galgame_id, '', '/galgame/' || NEW.galgame_id, false, NEW.created);
    RETURN NEW;
END $function$;

COMMIT;
