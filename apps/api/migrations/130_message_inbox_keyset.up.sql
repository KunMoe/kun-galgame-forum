-- 130: index the inbox tables for the v1 notification and conversation faces.
--
-- message had no index that starts with receiver_id (production: 356k rows),
-- so every inbox list, count and mark-read scanned the table; the v1 list
-- walks (created DESC, id DESC) within one receiver. chat_message had no
-- index on chat_room_id at all, and chat_room_participant could only be
-- probed by (chat_room_id, user_id), not by user.
--
-- All three are additive, so deploy order does not matter, and no row is
-- touched.

CREATE INDEX IF NOT EXISTS idx_message_receiver_created_id
  ON message (receiver_id, created DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_chat_message_room_id
  ON chat_message (chat_room_id, id DESC);

CREATE INDEX IF NOT EXISTS idx_chat_room_participant_user
  ON chat_room_participant (user_id, chat_room_id);
