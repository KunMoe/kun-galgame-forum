UPDATE galgame_resource
SET runtimes = '["onscripter", "tyranor"]'::jsonb,
    platforms = '["and"]'::jsonb
WHERE platform = 'emulator'
  AND runtimes = '["emulator"]'::jsonb
  AND platforms = '[]'::jsonb
  AND note LIKE '%本条为模拟器版（KRKR / Tyranor / ONS 等引擎资源）%';

UPDATE galgame_resource
SET runtimes = '["other"]'::jsonb,
    platforms = CASE WHEN platforms = '[]'::jsonb THEN '["oth"]'::jsonb ELSE platforms END
WHERE runtimes = '["emulator"]'::jsonb;

UPDATE galgame_resource
SET runtimes = runtimes - 'emulator'
WHERE runtimes ? 'emulator';
