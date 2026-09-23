-- The catalog half of the renumber map. Run read-only against kun_catalog and
-- save the output as the TSV that `align-galgame-ids -map` reads:
--
--   psql -X -q -At -F $'\t' -d kun_catalog -f catalog_map.sql > catalog-map.tsv
--
-- The rule is the one the site routed by until G0: a curated work anchor
-- names the work; a number without one resolves through a galgame-medium kungal
-- claim. Numbers neither half names are local rows whose id is already a
-- catalog id; the command adds them as 'unchanged'.
--
-- Medium 1 only: 9 product ids also carry an ASMR (medium 5) self-claim with
-- product_work_id = id, and every one of them has a curated anchor anyway.

WITH anchor AS (
  SELECT r.external_id::bigint AS old_id, r.entity_id AS new_id
  FROM catalog_external_ref r
  JOIN catalog_source s ON s.id = r.source_id
  WHERE s.key = 'curated' AND r.entity_type = 5 AND r.dead_at IS NULL
    AND r.external_id ~ '^[0-9]{1,18}$'
),
claim AS (
  SELECT w.product_work_id AS old_id, w.id AS new_id
  FROM catalog_work w
  WHERE w.site = 'kungal' AND w.medium_id = 1
    AND w.product_work_id IS NOT NULL AND w.deleted_at IS NULL
)
SELECT old_id, new_id, 'curated' FROM anchor
UNION ALL
SELECT c.old_id, c.new_id, 'claim' FROM claim c
WHERE NOT EXISTS (SELECT 1 FROM anchor a WHERE a.old_id = c.old_id)
ORDER BY 1;
