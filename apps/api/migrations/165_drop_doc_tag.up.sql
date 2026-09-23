-- 165: drop the doc tag tables (deploy-then-drop).
--
-- Doc tags never held a row in production: doc_tag and
-- doc_article_tag_relation were both empty on 2026-09-23, and no page ever
-- showed a tag. The D track removed the feature from the API and the editor
-- instead of carrying it into /api/v1 (docs/proj/api-v1/waves/d-doc.md §7).
--
-- DEPLOY ORDER MATTERS. The binary before that deploy reads
-- doc_article_tag_relation on every doc page (GET /api/doc/article/:slug), so
-- dropping it while that binary is live turns doc pages into 500s. 165 is in
-- cmd/migrate's default exclude list; run it with --only=165 once the deploy
-- that removed the tag code is live. Only tables go, no column of a table the
-- live binary reads, so the API needs no restart afterwards.

DROP TABLE IF EXISTS doc_article_tag_relation;
DROP TABLE IF EXISTS doc_tag;
