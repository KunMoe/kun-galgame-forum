-- 211: the rating cutover's leftovers go (210 moved ratings to NextMoe).
-- play_status was the rating's own play status; since 210 the mirror carries
-- the author's current state in author_state/author_completion and nothing
-- reads play_status. rating_cutover held 210's snapshot for infra's backfill
-- import, which infra confirmed done before this ran.
--
-- Deploy-then-drop: the code that stopped reading play_status shipped with
-- 210, so this rides an ordinary deploy.

ALTER TABLE galgame_rating DROP COLUMN IF EXISTS play_status;
DROP SCHEMA IF EXISTS rating_cutover CASCADE;
