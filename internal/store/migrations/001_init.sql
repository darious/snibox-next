-- Snibox Next initial schema.
-- See SPEC.md §2.2. Search is a plain LIKE over a denormalised
-- `search_blob` column kept in sync by triggers — no FTS5.

CREATE TABLE items (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    body        TEXT NOT NULL DEFAULT '',
    type        TEXT NOT NULL CHECK (type IN ('snippet','note','link')),
    language    TEXT,
    url         TEXT,
    pinned      INTEGER NOT NULL DEFAULT 0,
    archived    INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    search_blob TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_items_type     ON items(type)     WHERE archived = 0;
CREATE INDEX idx_items_pinned   ON items(pinned)   WHERE archived = 0;
CREATE INDEX idx_items_archived ON items(archived);
CREATE INDEX idx_items_updated  ON items(updated_at DESC);

CREATE TABLE tags (
    item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    tag     TEXT NOT NULL,
    PRIMARY KEY (item_id, tag)
);
CREATE INDEX idx_tags_tag ON tags(tag);

-- Rebuild search_blob from current row + joined tags.
-- Lowercased so LIKE can be case-insensitive without per-row collation.
CREATE TRIGGER trg_items_blob_ai
AFTER INSERT ON items
BEGIN
    UPDATE items
       SET search_blob = lower(
              coalesce(NEW.title,'')    || char(10) ||
              coalesce(NEW.body,'')     || char(10) ||
              coalesce(NEW.language,'') || char(10) ||
              coalesce(NEW.url,'')      || char(10) ||
              coalesce((SELECT group_concat(tag, ' ') FROM tags WHERE item_id = NEW.id), '')
           )
     WHERE id = NEW.id;
END;

CREATE TRIGGER trg_items_blob_au
AFTER UPDATE OF title, body, language, url ON items
BEGIN
    UPDATE items
       SET search_blob = lower(
              coalesce(NEW.title,'')    || char(10) ||
              coalesce(NEW.body,'')     || char(10) ||
              coalesce(NEW.language,'') || char(10) ||
              coalesce(NEW.url,'')      || char(10) ||
              coalesce((SELECT group_concat(tag, ' ') FROM tags WHERE item_id = NEW.id), '')
           )
     WHERE id = NEW.id;
END;

CREATE TRIGGER trg_tags_blob_ai
AFTER INSERT ON tags
BEGIN
    UPDATE items
       SET search_blob = lower(
              coalesce(title,'')    || char(10) ||
              coalesce(body,'')     || char(10) ||
              coalesce(language,'') || char(10) ||
              coalesce(url,'')      || char(10) ||
              coalesce((SELECT group_concat(tag, ' ') FROM tags WHERE item_id = NEW.item_id), '')
           )
     WHERE id = NEW.item_id;
END;

CREATE TRIGGER trg_tags_blob_ad
AFTER DELETE ON tags
BEGIN
    UPDATE items
       SET search_blob = lower(
              coalesce(title,'')    || char(10) ||
              coalesce(body,'')     || char(10) ||
              coalesce(language,'') || char(10) ||
              coalesce(url,'')      || char(10) ||
              coalesce((SELECT group_concat(tag, ' ') FROM tags WHERE item_id = OLD.item_id), '')
           )
     WHERE id = OLD.item_id;
END;
