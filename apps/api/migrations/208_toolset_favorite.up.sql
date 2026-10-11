-- 208: toolsets can be favorited (todo #126), the way quizzes are (048, 143).
--
-- galgame_toolset_favorite is its own table, not a shared polymorphic one, so
-- the foreign key holds and deleting a toolset takes its favorites with it.
-- id names the row in the moemoepoint idempotency key the author's ±1 is
-- pushed under; (toolset_id, user_id) stays the primary key the favorite is
-- toggled on. The user index serves the user's favorite list, newest first.
--
-- galgame_toolset.favorite_count is the cached count, moved in the same
-- transaction as the row. Existing rows: none; every toolset starts at 0.
--
-- Additive: migrate before deploy, which the deploy already does.

ALTER TABLE galgame_toolset
    ADD COLUMN IF NOT EXISTS favorite_count integer NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS galgame_toolset_favorite (
    id         bigserial   NOT NULL UNIQUE,
    toolset_id integer     NOT NULL REFERENCES galgame_toolset (id) ON DELETE CASCADE,
    user_id    integer     NOT NULL,
    created    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (toolset_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_galgame_toolset_favorite_user_created
    ON galgame_toolset_favorite (user_id, created DESC, id DESC);

SELECT user_purge_archive_attach();
