-- The tables come back empty, as they were in production when 165 ran.
CREATE TABLE IF NOT EXISTS doc_tag (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(128) NOT NULL,
    title VARCHAR(128) NOT NULL,
    description VARCHAR(255) NOT NULL DEFAULT '',
    created TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated TIMESTAMP(3) WITH TIME ZONE NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS doc_tag_slug_key ON doc_tag (slug);

CREATE TABLE IF NOT EXISTS doc_article_tag_relation (
    doc_article_id INTEGER NOT NULL REFERENCES doc_article (id) ON UPDATE CASCADE ON DELETE CASCADE,
    doc_tag_id INTEGER NOT NULL REFERENCES doc_tag (id) ON UPDATE CASCADE ON DELETE CASCADE,
    created TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated TIMESTAMP(3) WITH TIME ZONE NOT NULL,
    PRIMARY KEY (doc_article_id, doc_tag_id)
);
