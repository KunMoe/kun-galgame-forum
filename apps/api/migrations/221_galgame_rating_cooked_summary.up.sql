-- NextMoe now treats a rating's short_summary as Markdown and cooks it when the
-- rating is written: short_summary_html is the sanitized HTML every site shows,
-- short_summary_text the plain text with spoilers masked, for excerpts, and
-- short_summary_has_spoiler whether the review masks anything itself. The mirror
-- keeps them next to the source.
--
-- Existing rows start empty. After deploying, delete the Redis key
-- catalog:ratings:changes:cursor: the mirror then replays every rating NextMoe
-- holds and fills them with whatever NextMoe has cooked by then. NextMoe's later
-- changed_at bump reaches the mirror as usual. Without the reset, a bump that ran
-- before this deploy has already passed the cursor and those rows stay empty.

BEGIN;

ALTER TABLE galgame_rating
  ADD COLUMN IF NOT EXISTS short_summary_html text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS short_summary_text text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS short_summary_has_spoiler boolean NOT NULL DEFAULT false;

COMMIT;
