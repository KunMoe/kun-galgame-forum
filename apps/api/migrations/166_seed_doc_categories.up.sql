-- 166: make the four doc categories exist on every database.
--
-- /api/v1 serves a doc's category as the closed enum doc_category
-- (galgame | notice | kun | other; K25 in docs/proj/api-v1/waves/d-doc.md),
-- stored as doc_article.category_id → doc_category.slug. Production has had
-- exactly these four rows since the 2025-12 import and nothing ever added,
-- renamed or removed one; a fresh database had none, so no doc could be
-- created there. Existing rows are left as they are.

INSERT INTO doc_category (slug, title, sort_order, updated) VALUES
    ('galgame', 'Galgame', 0, now()),
    ('notice', '网站公告', 1, now()),
    ('kun', '关于鲲', 2, now()),
    ('other', '其它', 3, now())
ON CONFLICT (slug) DO NOTHING;
