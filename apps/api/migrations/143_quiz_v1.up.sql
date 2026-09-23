-- 143: the quiz tables for /api/v1/quizzes (G2, docs/proj/api-v1/waves/g2-quizzes.md).
--
-- Production on 2026-09-23 (267 quizzes, 11,116 answerer rows, 4 favourites):
--   * galgame_quiz_favorite was keyed only by (quiz_id, user_id), so the
--     favourite moemoepoint key had nothing stable to name and fell back to a
--     nonce. Each row gets its own id; the 4 existing rows are numbered by the
--     sequence as the column is added.
--   * The answer cursor walks one quiz's answerers newest first; the quiz_id
--     index alone sorted up to 251 rows per page.
--   * fill and essay were never offered by the web and have 0 rows; v1 accepts
--     single, multiple and judge only, and the CHECK now says the same.
--   * The feed trigger fired on every UPDATE, so each detail view, answer,
--     rating and favourite also rewrote feed_activity. The function is
--     unchanged; the trigger fires only when a column it reads changes
--     (question, user_id, created).
--
-- Every statement is IF NOT EXISTS or drop-and-recreate, so running it again
-- changes nothing.

ALTER TABLE galgame_quiz_favorite
    ADD COLUMN IF NOT EXISTS id bigserial;
CREATE UNIQUE INDEX IF NOT EXISTS galgame_quiz_favorite_id_key
    ON galgame_quiz_favorite (id);

CREATE INDEX IF NOT EXISTS idx_galgame_quiz_answer_quiz_created_id
    ON galgame_quiz_answer (quiz_id, created DESC, id DESC)
    WHERE role = 'answerer';

ALTER TABLE galgame_quiz DROP CONSTRAINT IF EXISTS galgame_quiz_type_check;
ALTER TABLE galgame_quiz ADD CONSTRAINT galgame_quiz_type_check
    CHECK (type IN ('single', 'multiple', 'judge'));

DROP TRIGGER IF EXISTS trg_feed_galgame_quiz ON galgame_quiz;
CREATE TRIGGER trg_feed_galgame_quiz
    AFTER INSERT OR DELETE OR UPDATE OF question, user_id, created
    ON galgame_quiz
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_quiz();
