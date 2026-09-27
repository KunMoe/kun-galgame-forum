-- 202: topic subscriptions (SB wave). A reader picks watching, normal or
-- muted per topic; watching gets a folded notification for every new reply,
-- muted silences replied, mentioned and commented for that topic. normal is
-- the absence of a row, so only watching and muted are stored.
--
-- Authors watch their own topics. Every existing topic gets its author's
-- watching row read up to the topic's last floor, with activity_at at its
-- last reply, so the backfill makes nothing unread and sends nothing; from
-- here on the author's per-reply "replied" message is sent only while that
-- row says watching.
--
-- notice_message_id points at the recipient's open folded notification and
-- has no foreign key: a deleted notification only means the next reply opens
-- a new fold. Additive: migrate before deploy, which the deploy already does.

CREATE TABLE IF NOT EXISTS topic_subscription (
    user_id            int         NOT NULL,
    topic_id           int         NOT NULL REFERENCES topic (id) ON DELETE CASCADE,
    notification_level text        NOT NULL CHECK (notification_level IN ('watching', 'muted')),
    last_read_floor    int         NOT NULL DEFAULT 0,
    activity_at        timestamptz NOT NULL DEFAULT now(),
    notice_message_id  int,
    notice_from_floor  int,
    created            timestamptz NOT NULL DEFAULT now(),
    updated            timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, topic_id)
);

CREATE INDEX IF NOT EXISTS idx_topic_subscription_watchers
    ON topic_subscription (topic_id) WHERE notification_level = 'watching';

CREATE INDEX IF NOT EXISTS idx_topic_subscription_activity
    ON topic_subscription (user_id, activity_at DESC, topic_id DESC) WHERE notification_level = 'watching';

INSERT INTO topic_subscription (user_id, topic_id, notification_level, last_read_floor, activity_at)
SELECT t.user_id, t.id, 'watching', t.last_reply_floor,
       COALESCE((SELECT max(r.created) FROM topic_reply r WHERE r.topic_id = t.id), t.created)
FROM topic t
WHERE t.user_id > 0
ON CONFLICT (user_id, topic_id) DO NOTHING;

SELECT user_purge_archive_attach();
