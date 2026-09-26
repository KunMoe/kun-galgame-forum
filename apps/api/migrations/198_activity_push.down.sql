DROP TRIGGER IF EXISTS trg_activity_push_enqueue ON feed_activity;
DROP FUNCTION IF EXISTS activity_push_enqueue();
DROP TABLE IF EXISTS activity_push_queue;
DROP TABLE IF EXISTS activity_push_sent;
