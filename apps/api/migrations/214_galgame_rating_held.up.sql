-- 214: ratings NextMoe takes off a work for a while are held aside, not
-- dropped.
--
-- /v2/ratings/changes reports a rating as gone both when its author deleted
-- it and when catalog deleted its work (infra #454) or merged the work and
-- folded the older of one user's two ratings (#455). The last two come back
-- under the same id when the deletion or merge is undone. The rating mirror
-- deleted every gone rating, and with it the forum-only columns (galgame_type,
-- view, like_count, comment_count) and its likes (galgame_rating_like
-- cascades), which the rating sent back cannot return.
--
-- A held rating is its galgame_rating row and its likes as JSON, out of every
-- read. When NextMoe sends it again the mirror writes it back with them.
-- reason is deleted (the work was deleted; restorable at any time, so kept) or
-- merged (folded by a merge; undoable for 30 days, so purged after 31).
-- A purged author's ratings are never held.
--
-- galgame_rating_gone_retry holds the gone ratings catalog could not say
-- anything about yet, so the mirror's cursor can move past them. It is a table
-- and not a Redis key because the cursor has already passed them: a lost key
-- would leave each of them readable for good.
--
-- Additive: migrate before or with the deploy.

CREATE TABLE IF NOT EXISTS galgame_rating_held (
  id      integer     PRIMARY KEY,
  user_id integer     NOT NULL,
  work_id integer     NOT NULL,
  reason  text        NOT NULL,
  row     jsonb       NOT NULL,
  likes   jsonb       NOT NULL DEFAULT '[]'::jsonb,
  held_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_galgame_rating_held_merged ON galgame_rating_held (held_at) WHERE reason = 'merged';

CREATE TABLE IF NOT EXISTS galgame_rating_gone_retry (
  rating_id integer     PRIMARY KEY,
  work_id   integer     NOT NULL,
  parked_at timestamptz NOT NULL DEFAULT now()
);

SELECT user_purge_archive_attach();
