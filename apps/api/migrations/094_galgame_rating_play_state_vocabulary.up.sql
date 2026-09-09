-- 094: unify the play-state vocabulary with catalog's work-state resource.
--
-- galgame_rating.play_status and catalog's catalog_user_work_state held the same
-- concept in two different vocabularies, so the galgame detail page carried two
-- "游玩状态" selectors that could disagree with each other. The forum now stores
-- one flat value that maps onto catalog's two axes (state + completion):
--
--   not_started   -> wish
--   in_progress   -> doing
--   finished_one  -> done_one_route   (state=done, completion=one_route)
--   finished_main -> done_main        (state=done, completion=main)
--   finished_all  -> done_all         (state=done, completion=all)
--   dropped       -> dropped          (unchanged)
--
-- on_hold is new to the rating side and has no legacy rows. Counted on prod
-- 2026-09-09: finished_all 2561, finished_main 405, finished_one 345,
-- not_started 176, in_progress 123, dropped 66 — 3676 rows, every one of them a
-- value listed above, so nothing is left behind. Re-running matches nothing.
--
-- Ship this WITH the code, not before it: the running frontend maps only the old
-- values, so an early migration makes every rating badge render the raw enum
-- string and empties the play-status filter on /galgame-rating.

UPDATE galgame_rating SET play_status = 'wish'           WHERE play_status = 'not_started';
UPDATE galgame_rating SET play_status = 'doing'          WHERE play_status = 'in_progress';
UPDATE galgame_rating SET play_status = 'done_one_route' WHERE play_status = 'finished_one';
UPDATE galgame_rating SET play_status = 'done_main'      WHERE play_status = 'finished_main';
UPDATE galgame_rating SET play_status = 'done_all'       WHERE play_status = 'finished_all';
