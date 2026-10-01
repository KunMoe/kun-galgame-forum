-- 205 down: drop the relocation ledger. Once resources have been retired this
-- also drops their only snapshot and the redirects to LetMoe.
DROP TABLE IF EXISTS galgame_resource_relocation;
