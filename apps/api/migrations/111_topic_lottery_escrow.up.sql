-- 111: lottery point prizes are paid for by the lottery's author, and the
-- entrant list becomes a cursor collection.
--
-- point_escrow is how much moemoepoint this lottery is currently holding from
-- its author. Until now a point prize was minted at draw time: the winners were
-- credited and nobody was charged (350 points to 5 users on production by
-- 2026-09-23). The user decided on 2026-09-23 that the author funds the pool:
-- it is taken when the lottery is created, the difference is taken or returned
-- when the prizes are rewritten, and what was not paid out comes back on cancel,
-- delete or draw. Every existing row starts at 0, so no author is charged
-- retroactively and a draw of an old lottery refunds nothing.
--
-- The index serves the v1 entrant list, sorted created ASC, id ASC. Entries
-- share a second whenever a lottery gets popular, so the id column is the
-- tie-breaker, not decoration.

ALTER TABLE topic_lottery
  ADD COLUMN IF NOT EXISTS point_escrow integer NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_topic_lottery_entry_lottery_created_id
  ON topic_lottery_entry (lottery_id, created, id);
