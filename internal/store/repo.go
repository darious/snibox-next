package store

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

func (r *Repo) DB() *sql.DB { return r.db }

var ErrNotFound = errors.New("item not found")

const timeFormat = time.RFC3339Nano

func formatTime(t time.Time) string { return t.UTC().Format(timeFormat) }

func parseTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC(), nil
	}
	return time.Parse(time.RFC3339, s)
}

func nullStr(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

// Create inserts the item and its tags. The caller's item ID, CreatedAt,
// UpdatedAt are used as-is (caller is responsible for setting them).
func (r *Repo) Create(ctx context.Context, it *Item) error {
	if err := it.Validate(); err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := insertItemTx(ctx, tx, it); err != nil {
		return err
	}
	return tx.Commit()
}

func insertItemTx(ctx context.Context, tx *sql.Tx, it *Item) error {
	_, err := tx.ExecContext(ctx, `
        INSERT INTO items(id, title, body, type, language, url, pinned, archived, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		it.ID, it.Title, it.Body, string(it.Type), nullStr(it.Language), nullStr(it.URL),
		boolToInt(it.Pinned), boolToInt(it.Archived),
		formatTime(it.CreatedAt), formatTime(it.UpdatedAt))
	if err != nil {
		return err
	}
	for _, t := range it.Tags {
		if _, err := tx.ExecContext(ctx, `INSERT INTO tags(item_id, tag) VALUES (?, ?)`, it.ID, t); err != nil {
			return err
		}
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Get fetches an item by id (including tags).
func (r *Repo) Get(ctx context.Context, id string) (*Item, error) {
	row := r.db.QueryRowContext(ctx, `
        SELECT id, title, body, type, language, url, pinned, archived, created_at, updated_at
        FROM items WHERE id = ?`, id)
	it, err := scanItem(row)
	if err != nil {
		return nil, err
	}
	tags, err := r.tagsForItem(ctx, id)
	if err != nil {
		return nil, err
	}
	it.Tags = tags
	return it, nil
}

func (r *Repo) tagsForItem(ctx context.Context, id string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT tag FROM tags WHERE item_id = ? ORDER BY tag`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

type rowScanner interface {
	Scan(...any) error
}

func scanItem(s rowScanner) (*Item, error) {
	var (
		it     Item
		typ    string
		lang   sql.NullString
		urlStr sql.NullString
		pinned int
		arch   int
		ca, ua string
	)
	if err := s.Scan(&it.ID, &it.Title, &it.Body, &typ, &lang, &urlStr, &pinned, &arch, &ca, &ua); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	it.Type = ItemType(typ)
	if lang.Valid {
		s := lang.String
		it.Language = &s
	}
	if urlStr.Valid {
		s := urlStr.String
		it.URL = &s
	}
	it.Pinned = pinned != 0
	it.Archived = arch != 0
	var err error
	if it.CreatedAt, err = parseTime(ca); err != nil {
		return nil, err
	}
	if it.UpdatedAt, err = parseTime(ua); err != nil {
		return nil, err
	}
	return &it, nil
}

// Update replaces mutable fields of an existing item. Tags are replaced wholesale.
// CreatedAt is preserved; UpdatedAt is bumped to now.
func (r *Repo) Update(ctx context.Context, it *Item) error {
	if err := it.Validate(); err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	it.UpdatedAt = time.Now().UTC()
	res, err := tx.ExecContext(ctx, `
        UPDATE items SET title=?, body=?, type=?, language=?, url=?, pinned=?, archived=?, updated_at=?
        WHERE id = ?`,
		it.Title, it.Body, string(it.Type), nullStr(it.Language), nullStr(it.URL),
		boolToInt(it.Pinned), boolToInt(it.Archived),
		formatTime(it.UpdatedAt), it.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM tags WHERE item_id = ?`, it.ID); err != nil {
		return err
	}
	for _, t := range it.Tags {
		if _, err := tx.ExecContext(ctx, `INSERT INTO tags(item_id, tag) VALUES (?, ?)`, it.ID, t); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repo) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM items WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// TogglePin / ToggleArchive flip the corresponding flag and bump updated_at.
func (r *Repo) TogglePin(ctx context.Context, id string) (*Item, error) {
	return r.toggleFlag(ctx, id, "pinned")
}
func (r *Repo) ToggleArchive(ctx context.Context, id string) (*Item, error) {
	return r.toggleFlag(ctx, id, "archived")
}

func (r *Repo) toggleFlag(ctx context.Context, id, col string) (*Item, error) {
	if col != "pinned" && col != "archived" {
		return nil, fmt.Errorf("invalid flag %s", col)
	}
	now := formatTime(time.Now().UTC())
	res, err := r.db.ExecContext(ctx, fmt.Sprintf(
		`UPDATE items SET %s = 1 - %s, updated_at = ? WHERE id = ?`, col, col),
		now, id)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	return r.Get(ctx, id)
}

func (r *Repo) AddTag(ctx context.Context, id, tag string) error {
	tag = NormaliseTag(tag)
	if err := ValidateTag(tag); err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM items WHERE id = ?`, id).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO tags(item_id, tag) VALUES (?, ?)`, id, tag); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE items SET updated_at = ? WHERE id = ?`, formatTime(time.Now().UTC()), id); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repo) RemoveTag(ctx context.Context, id, tag string) error {
	tag = NormaliseTag(tag)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `DELETE FROM tags WHERE item_id = ? AND tag = ?`, id, tag)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	if _, err := tx.ExecContext(ctx, `UPDATE items SET updated_at = ? WHERE id = ?`, formatTime(time.Now().UTC()), id); err != nil {
		return err
	}
	return tx.Commit()
}

// TagCount is a tag name and the number of non-archived items using it.
type TagCount struct {
	Tag   string
	Count int
}

func (r *Repo) TagCounts(ctx context.Context) ([]TagCount, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT t.tag, COUNT(*) AS n
        FROM tags t
        JOIN items i ON i.id = t.item_id
        WHERE i.archived = 0
        GROUP BY t.tag
        ORDER BY t.tag`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TagCount
	for rows.Next() {
		var tc TagCount
		if err := rows.Scan(&tc.Tag, &tc.Count); err != nil {
			return nil, err
		}
		out = append(out, tc)
	}
	return out, rows.Err()
}

// Counts returns view-level counts for the nav rail.
type Counts struct {
	All, Snippets, Notes, Links, Pinned, Archive int
}

func (r *Repo) Counts(ctx context.Context) (Counts, error) {
	var c Counts
	rows, err := r.db.QueryContext(ctx, `
        SELECT
            SUM(CASE WHEN archived = 0                       THEN 1 ELSE 0 END),
            SUM(CASE WHEN archived = 0 AND type = 'snippet'  THEN 1 ELSE 0 END),
            SUM(CASE WHEN archived = 0 AND type = 'note'     THEN 1 ELSE 0 END),
            SUM(CASE WHEN archived = 0 AND type = 'link'     THEN 1 ELSE 0 END),
            SUM(CASE WHEN archived = 0 AND pinned = 1        THEN 1 ELSE 0 END),
            SUM(CASE WHEN archived = 1                       THEN 1 ELSE 0 END)
        FROM items`)
	if err != nil {
		return c, err
	}
	defer rows.Close()
	if !rows.Next() {
		return c, nil
	}
	var a, s, n, l, p, ar sql.NullInt64
	if err := rows.Scan(&a, &s, &n, &l, &p, &ar); err != nil {
		return c, err
	}
	c.All = int(a.Int64)
	c.Snippets = int(s.Int64)
	c.Notes = int(n.Int64)
	c.Links = int(l.Int64)
	c.Pinned = int(p.Int64)
	c.Archive = int(ar.Int64)
	return c, nil
}

// Filter narrows a List query. Zero value = "All view".
type Filter struct {
	View     string   // all | snippets | notes | links | pinned | archive | search
	Query    string   // search text
	Tags     []string // AND semantics
	Type     string   // optional snippet|note|link
	Language string   // optional, snippets only
	Sort     string   // updated (default) | created | title
	Cursor   string   // opaque
	Limit    int      // default 50
}

const defaultLimit = 50

// List returns a page of items matching the filter, plus the next-page cursor.
func (r *Repo) List(ctx context.Context, f Filter) ([]*Item, string, error) {
	if f.Limit <= 0 {
		f.Limit = defaultLimit
	}

	var (
		where []string
		args  []any
	)
	switch f.View {
	case "", "all":
		where = append(where, "i.archived = 0")
	case "snippets":
		where = append(where, "i.archived = 0", "i.type = 'snippet'")
	case "notes":
		where = append(where, "i.archived = 0", "i.type = 'note'")
	case "links":
		where = append(where, "i.archived = 0", "i.type = 'link'")
	case "pinned":
		where = append(where, "i.archived = 0", "i.pinned = 1")
	case "archive":
		where = append(where, "i.archived = 1")
	case "search":
		where = append(where, "i.archived = 0")
	default:
		return nil, "", fmt.Errorf("unknown view %q", f.View)
	}

	if f.Type != "" {
		where = append(where, "i.type = ?")
		args = append(args, f.Type)
	}
	if f.Language != "" {
		where = append(where, "i.language = ?")
		args = append(args, f.Language)
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		q = strings.ToLower(q)
		if len(q) < 2 {
			where = append(where, "lower(i.title) LIKE ?")
			args = append(args, "%"+q+"%")
		} else {
			where = append(where, "i.search_blob LIKE ?")
			args = append(args, "%"+q+"%")
		}
	}
	if len(f.Tags) > 0 {
		// AND semantics: each tag must exist for the item.
		for _, t := range f.Tags {
			where = append(where, `EXISTS (SELECT 1 FROM tags WHERE item_id = i.id AND tag = ?)`)
			args = append(args, NormaliseTag(t))
		}
	}

	orderCol, orderDir := "i.updated_at", "DESC"
	switch f.Sort {
	case "", "updated":
	case "created":
		orderCol = "i.created_at"
	case "title":
		orderCol, orderDir = "lower(i.title)", "ASC"
	default:
		return nil, "", fmt.Errorf("unknown sort %q", f.Sort)
	}

	// Pinned float-to-top except in archive view.
	pinnedFirst := f.View != "archive"

	if f.Cursor != "" {
		cv, cid, err := decodeCursor(f.Cursor)
		if err != nil {
			return nil, "", fmt.Errorf("bad cursor: %w", err)
		}
		op := "<"
		if orderDir == "ASC" {
			op = ">"
		}
		where = append(where, fmt.Sprintf("(%s %s ? OR (%s = ? AND i.id %s ?))", orderCol, op, orderCol, op))
		args = append(args, cv, cv, cid)
	}

	query := "SELECT i.id, i.title, i.body, i.type, i.language, i.url, i.pinned, i.archived, i.created_at, i.updated_at FROM items i"
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	if pinnedFirst {
		query += " ORDER BY i.pinned DESC, " + orderCol + " " + orderDir + ", i.id " + orderDir
	} else {
		query += " ORDER BY " + orderCol + " " + orderDir + ", i.id " + orderDir
	}
	query += fmt.Sprintf(" LIMIT %d", f.Limit+1)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var items []*Item
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, "", err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(items) > f.Limit {
		last := items[f.Limit-1]
		items = items[:f.Limit]
		var v string
		switch f.Sort {
		case "created":
			v = formatTime(last.CreatedAt)
		case "title":
			v = strings.ToLower(last.Title)
		default:
			v = formatTime(last.UpdatedAt)
		}
		nextCursor = encodeCursor(v, last.ID)
	}

	if err := r.attachTags(ctx, items); err != nil {
		return nil, "", err
	}
	return items, nextCursor, nil
}

func (r *Repo) attachTags(ctx context.Context, items []*Item) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, 0, len(items))
	idx := make(map[string]*Item, len(items))
	placeholders := make([]string, 0, len(items))
	args := make([]any, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
		idx[it.ID] = it
		placeholders = append(placeholders, "?")
		args = append(args, it.ID)
	}
	q := fmt.Sprintf(`SELECT item_id, tag FROM tags WHERE item_id IN (%s) ORDER BY tag`,
		strings.Join(placeholders, ","))
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var iid, tag string
		if err := rows.Scan(&iid, &tag); err != nil {
			return err
		}
		if it, ok := idx[iid]; ok {
			it.Tags = append(it.Tags, tag)
		}
	}
	// Ensure tag slice exists (avoid nil for templ).
	for _, it := range items {
		sort.Strings(it.Tags)
	}
	return rows.Err()
}

func encodeCursor(v, id string) string {
	return base64.URLEncoding.EncodeToString([]byte(v + "\x00" + id))
}

func decodeCursor(c string) (string, string, error) {
	b, err := base64.URLEncoding.DecodeString(c)
	if err != nil {
		return "", "", err
	}
	parts := strings.SplitN(string(b), "\x00", 2)
	if len(parts) != 2 {
		return "", "", errors.New("malformed")
	}
	return parts[0], parts[1], nil
}

// ListAll streams every item (used for export). Caller closes the rows.
func (r *Repo) ListAll(ctx context.Context) ([]*Item, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT id, title, body, type, language, url, pinned, archived, created_at, updated_at
        FROM items ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*Item
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.attachTags(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}

// ReplaceAll wipes the table and inserts all items inside a single transaction.
// Used by importer "replace" mode. Preserves caller-provided timestamps.
func (r *Repo) ReplaceAll(ctx context.Context, items []*Item) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM items`); err != nil {
		return err
	}
	for _, it := range items {
		if err := insertItemTx(ctx, tx, it); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Upsert merges items by id. Each item's timestamps are used as-is.
func (r *Repo) Upsert(ctx context.Context, items []*Item) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, it := range items {
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO items(id, title, body, type, language, url, pinned, archived, created_at, updated_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            ON CONFLICT(id) DO UPDATE SET
                title=excluded.title, body=excluded.body, type=excluded.type,
                language=excluded.language, url=excluded.url,
                pinned=excluded.pinned, archived=excluded.archived,
                created_at=excluded.created_at, updated_at=excluded.updated_at`,
			it.ID, it.Title, it.Body, string(it.Type), nullStr(it.Language), nullStr(it.URL),
			boolToInt(it.Pinned), boolToInt(it.Archived),
			formatTime(it.CreatedAt), formatTime(it.UpdatedAt)); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM tags WHERE item_id = ?`, it.ID); err != nil {
			return err
		}
		for _, t := range it.Tags {
			if _, err := tx.ExecContext(ctx, `INSERT INTO tags(item_id, tag) VALUES (?, ?)`, it.ID, t); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}
