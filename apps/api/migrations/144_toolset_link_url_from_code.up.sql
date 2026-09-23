-- 144: lift the one toolset download link that was stored inside an
-- extraction code (G1.1, docs/proj/api-v1/waves/g1-toolsets.md §3.12).
--
-- Production on 2026-09-23: 2 of the 26 link resources (type 'user') have an
-- empty content, which v1 sent as download_url "" against its own pattern.
--   * id 179 (toolset 130): the poster pasted the whole cloud-drive share text
--     into code, and it holds exactly one https URL. That URL is copied into
--     content. code keeps the full share text; no other column changes.
--   * id 122 (toolset 100) has no URL anywhere and is left alone; v1 now
--     answers its download with download_url null.
-- File resources 44 and 121 have neither an artifact nor an object key: the
-- legacy PATCH overwrote content on file resources. Nothing here can restore
-- them; v1 answers null for them too.
--
-- Only a link row whose content is still empty and whose code holds exactly
-- one URL is touched, so running it again changes nothing.

UPDATE galgame_toolset_resource
SET content = substring(code FROM 'https?://[^[:space:]]+')
WHERE type = 'user'
  AND content = ''
  AND (SELECT count(*) FROM regexp_matches(code, 'https?://', 'g')) = 1;
