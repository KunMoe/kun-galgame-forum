DROP TRIGGER IF EXISTS trg_feed_galgame_quiz ON galgame_quiz;
CREATE TRIGGER trg_feed_galgame_quiz
    AFTER INSERT OR DELETE OR UPDATE ON galgame_quiz
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_quiz();

ALTER TABLE galgame_quiz DROP CONSTRAINT IF EXISTS galgame_quiz_type_check;
ALTER TABLE galgame_quiz ADD CONSTRAINT galgame_quiz_type_check
    CHECK (type IN ('single', 'multiple', 'judge', 'fill', 'essay'));

DROP INDEX IF EXISTS idx_galgame_quiz_answer_quiz_created_id;

DROP INDEX IF EXISTS galgame_quiz_favorite_id_key;
ALTER TABLE galgame_quiz_favorite DROP COLUMN IF EXISTS id;
