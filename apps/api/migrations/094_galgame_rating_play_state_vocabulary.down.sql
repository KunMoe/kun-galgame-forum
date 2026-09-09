-- Reverse of 094. on_hold has no pre-094 equivalent on the rating side, so rows
-- written as on_hold after 094 land on in_progress rather than disappearing.

UPDATE galgame_rating SET play_status = 'not_started'   WHERE play_status = 'wish';
UPDATE galgame_rating SET play_status = 'in_progress'   WHERE play_status = 'doing';
UPDATE galgame_rating SET play_status = 'finished_one'  WHERE play_status = 'done_one_route';
UPDATE galgame_rating SET play_status = 'finished_main' WHERE play_status = 'done_main';
UPDATE galgame_rating SET play_status = 'finished_all'  WHERE play_status = 'done_all';
UPDATE galgame_rating SET play_status = 'in_progress'   WHERE play_status = 'on_hold';
