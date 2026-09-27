DROP TRIGGER IF EXISTS trg_anchor_presentation_galgame ON galgame;
DROP TRIGGER IF EXISTS trg_anchor_presentation_galgame_resource ON galgame_resource;
DROP TRIGGER IF EXISTS trg_anchor_presentation_galgame_rating ON galgame_rating;
DROP TRIGGER IF EXISTS trg_anchor_presentation_galgame_quiz ON galgame_quiz;
DROP TRIGGER IF EXISTS trg_anchor_presentation_galgame_quiz_galgame ON galgame_quiz_galgame;
DROP TRIGGER IF EXISTS trg_anchor_presentation_galgame_toolset ON galgame_toolset;
DROP TRIGGER IF EXISTS trg_anchor_presentation_galgame_website ON galgame_website;
DROP TRIGGER IF EXISTS trg_anchor_presentation_feed_activity ON feed_activity;

DROP FUNCTION IF EXISTS anchor_presentation_on_galgame();
DROP FUNCTION IF EXISTS anchor_presentation_on_galgame_resource();
DROP FUNCTION IF EXISTS anchor_presentation_on_galgame_rating();
DROP FUNCTION IF EXISTS anchor_presentation_on_galgame_quiz();
DROP FUNCTION IF EXISTS anchor_presentation_on_galgame_quiz_galgame();
DROP FUNCTION IF EXISTS anchor_presentation_on_galgame_toolset();
DROP FUNCTION IF EXISTS anchor_presentation_on_galgame_website();
DROP FUNCTION IF EXISTS anchor_presentation_on_feed_activity();
DROP FUNCTION IF EXISTS anchor_presentation_enqueue_work_children(integer);
DROP FUNCTION IF EXISTS anchor_presentation_enqueue(smallint, text);

DROP TABLE IF EXISTS anchor_presentation_queue;
DROP TABLE IF EXISTS anchor_presentation_sent;
