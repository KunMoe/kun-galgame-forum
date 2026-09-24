-- 196: an emulator pack whose uploader never said which emulator is stored as
-- runtime emulator, not other, so the 模拟器 filter finds it again.
--
-- The old resource form offered 模拟器 as a platform next to 安卓直装. 098 and
-- the backfill-resource-axes run moved that choice onto the runtime axis, but
-- only where the note named an emulator; every other legacy emulator row got
-- runtimes ["other"] and platforms ["oth"]. The platform filter had no
-- 模拟器 left, and those rows sat under 其它平台 instead. On 2026-09-24 that
-- was 1,888 rows (1,882 with platforms ["oth"], 6 with ["and"]; 140 of them
-- hidden or deleted).
--
-- Existing rows, all still carrying the legacy scalar platform = 'emulator':
-- 1. runtimes ["other"] → ["emulator"]. A new-form write never leaves that
--    pair, since other is not an emulator runtime and the scalar would have
--    been recomputed. platforms ["oth"] → []: the files' own platform is
--    unknown, and ["oth"] was only the placeholder.
-- 2. 587 rows of one uploader whose note reads "本条为模拟器版（KRKR / Tyranor
--    / ONS 等引擎资源）". The backfill took ONS and Tyranor out of that list and
--    derived platform and from them, but the note names no emulator for the
--    row, so they get runtimes ["emulator"] and platforms [] like step 1.
--
-- Not touched: 398 collections of another uploader carry the dvd platform
-- with nothing in the note about DVD. They were set by the uploader through
-- the new form between 2026-09-21 and 2026-09-24, when the legacy update path
-- did not stamp edited, so they are that user's own choice.
UPDATE galgame_resource
SET runtimes = '["emulator"]'::jsonb,
    platforms = CASE WHEN platforms = '["oth"]'::jsonb THEN '[]'::jsonb ELSE platforms END
WHERE platform = 'emulator'
  AND runtimes = '["other"]'::jsonb;

UPDATE galgame_resource
SET runtimes = '["emulator"]'::jsonb,
    platforms = '[]'::jsonb
WHERE platform = 'emulator'
  AND runtimes = '["onscripter", "tyranor"]'::jsonb
  AND platforms = '["and"]'::jsonb
  AND note LIKE '%本条为模拟器版（KRKR / Tyranor / ONS 等引擎资源）%';
