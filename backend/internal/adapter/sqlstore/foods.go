package sqlstore

import (
	"context"
	"database/sql"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

type datasets repos

func (r datasets) Upsert(ctx context.Context, d food.Dataset) error {
	if d.ImportedAt.IsZero() {
		d.ImportedAt = time.Now()
	}
	_, err := r.q.exec(ctx, `
		INSERT INTO datasets (id, kind, name, version, license, url, attribution, imported_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			kind = excluded.kind, name = excluded.name, version = excluded.version,
			license = excluded.license, url = excluded.url, attribution = excluded.attribution,
			imported_at = excluded.imported_at`,
		d.ID, d.Kind, d.Name, d.Version, d.License, d.URL, d.Attribution, r.d.timeValue(d.ImportedAt))
	return err
}

const datasetColumns = "id, kind, name, version, license, url, attribution, imported_at"

func scanDataset(row interface{ Scan(...any) error }) (food.Dataset, error) {
	var d food.Dataset
	err := row.Scan(&d.ID, &d.Kind, &d.Name, &d.Version, &d.License, &d.URL, &d.Attribution, timeCol{&d.ImportedAt})
	return d, notFound(err)
}

func (r datasets) Get(ctx context.Context, id food.DatasetID) (food.Dataset, error) {
	return scanDataset(r.q.queryRow(ctx, "SELECT "+datasetColumns+" FROM datasets WHERE id = ?", id))
}

func (r datasets) List(ctx context.Context) ([]food.Dataset, error) {
	rows, err := r.q.query(ctx, "SELECT "+datasetColumns+" FROM datasets ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []food.Dataset
	for rows.Next() {
		d, err := scanDataset(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

type foods repos

const (
	foodColumns     = "f.id, f.owner_id, f.kind, f.barcode, f.current_revision_id, f.created_at, f.deleted_at, f.discontinued_at"
	revisionColumns = "r.id, r.food_id, r.number, r.name, r.brand, r.aliases, r.basis_unit, r.basis_grams, r.dataset_id, r.source_ref, r.source_url, r.retrieved_at, r.created_at"
	visibleTo       = "(f.owner_id IS NULL OR f.owner_id = ?)"
)

func scanFoodAndRevision(row interface{ Scan(...any) error }) (food.Food, food.Revision, error) {
	var (
		f     food.Food
		rev   food.Revision
		owner sql.NullString
	)
	err := row.Scan(
		&f.ID, &owner, &f.Kind, &f.Barcode, &f.CurrentRevisionID, timeCol{&f.CreatedAt}, nullTimeCol{&f.DeletedAt}, nullTimeCol{&f.DiscontinuedAt},
		&rev.ID, &rev.FoodID, &rev.Number, &rev.Name, &rev.Brand, aliasesCol{&rev.Aliases}, &rev.Basis.Unit, decimalCol{&rev.Basis.Grams},
		&rev.Source.DatasetID, &rev.Source.Reference, &rev.Source.URL, zeroTimeCol{&rev.Source.RetrievedAt}, timeCol{&rev.CreatedAt},
	)
	f.OwnerID = user.ID(owner.String)
	return f, rev, notFound(err)
}

func scanRevision(row interface{ Scan(...any) error }) (food.Revision, error) {
	var rev food.Revision
	err := row.Scan(&rev.ID, &rev.FoodID, &rev.Number, &rev.Name, &rev.Brand, aliasesCol{&rev.Aliases}, &rev.Basis.Unit, decimalCol{&rev.Basis.Grams},
		&rev.Source.DatasetID, &rev.Source.Reference, &rev.Source.URL, zeroTimeCol{&rev.Source.RetrievedAt}, timeCol{&rev.CreatedAt})
	return rev, notFound(err)
}

func ownerArg(id user.ID) any {
	if id == "" {
		return nil
	}
	return string(id)
}

func (r foods) Create(ctx context.Context, f food.Food, rev food.Revision) error {
	_, err := r.q.exec(ctx, `
		INSERT INTO foods (id, owner_id, kind, barcode, current_revision_id, created_at, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, NULL)`,
		f.ID, ownerArg(f.OwnerID), f.Kind, f.Barcode, rev.ID, r.d.timeValue(f.CreatedAt))
	if err != nil {
		return err
	}
	return r.insertRevision(ctx, rev)
}

func (r foods) AddRevision(ctx context.Context, rev food.Revision) error {
	// 現行リビジョンのみ保持
	if _, err := r.q.exec(ctx,
		"DELETE FROM food_search_grams WHERE revision_id IN (SELECT current_revision_id FROM foods WHERE id = ?)", rev.FoodID); err != nil {
		return err
	}
	if err := r.insertRevision(ctx, rev); err != nil {
		return err
	}
	return affectedOne(r.q.exec(ctx, "UPDATE foods SET current_revision_id = ? WHERE id = ?", rev.ID, rev.FoodID))
}

func (r foods) insertRevision(ctx context.Context, rev food.Revision) error {
	k := r.idx.keys(rev.Name, rev.Brand, rev.Aliases)
	_, err := r.q.exec(ctx, `
		INSERT INTO food_revisions (id, food_id, number, name, brand, aliases, search_key, reading_key, compact_key, compact_reading, gram_count,
			basis_unit, basis_grams, dataset_id, source_ref, source_url, retrieved_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rev.ID, rev.FoodID, rev.Number, rev.Name, rev.Brand, aliasesArg(rev.Aliases), k.key, k.reading, k.compact, k.compactReading, len(k.grams),
		rev.Basis.Unit, decimalArg(rev.Basis.Grams),
		rev.Source.DatasetID, rev.Source.Reference, rev.Source.URL, nullTimeArg(r.d, rev.Source.RetrievedAt), r.d.timeValue(rev.CreatedAt))
	if err != nil {
		return err
	}
	if err := insertGrams(ctx, r.q, string(rev.ID), k.grams); err != nil {
		return err
	}
	for code, v := range rev.Nutrients {
		_, err := r.q.exec(ctx,
			"INSERT INTO food_nutrients (revision_id, code, amount, status, coverage) VALUES (?, ?, ?, ?, ?)",
			rev.ID, code, decimalArg(v.Amount), v.Status, decimalArg(v.Coverage))
		if err != nil {
			return err
		}
	}
	for i, p := range rev.Portions {
		_, err := r.q.exec(ctx,
			"INSERT INTO food_portions (revision_id, position, unit, grams) VALUES (?, ?, ?, ?)",
			rev.ID, i, p.Unit, decimalArg(p.Grams))
		if err != nil {
			return err
		}
	}
	return nil
}

func (r foods) Get(ctx context.Context, viewer user.ID, id food.ID) (food.Food, food.Revision, error) {
	return r.one(ctx, "f.id = ? AND "+visibleTo, id, ownerArg(viewer))
}

func (r foods) FindByBarcode(ctx context.Context, viewer user.ID, barcode string) (food.Food, food.Revision, error) {
	return r.one(ctx,
		"f.barcode = ? AND f.deleted_at IS NULL AND "+visibleTo+" ORDER BY CASE WHEN f.owner_id IS NULL THEN 1 ELSE 0 END, f.created_at DESC",
		barcode, ownerArg(viewer))
}

func (r foods) FindBySource(ctx context.Context, dataset food.DatasetID, reference string) (food.Food, food.Revision, error) {
	return r.one(ctx,
		"f.owner_id IS NULL AND r.dataset_id = ? AND r.source_ref = ? ORDER BY f.created_at",
		dataset, reference)
}

func (r foods) one(ctx context.Context, where string, args ...any) (food.Food, food.Revision, error) {
	f, rev, err := scanFoodAndRevision(r.q.queryRow(ctx,
		"SELECT "+foodColumns+", "+revisionColumns+" FROM foods f JOIN food_revisions r ON r.id = f.current_revision_id WHERE "+where+" LIMIT 1",
		args...))
	if err != nil {
		return food.Food{}, food.Revision{}, err
	}
	revs := []*food.Revision{&rev}
	if err := r.loadDetails(ctx, revs); err != nil {
		return food.Food{}, food.Revision{}, err
	}
	return f, rev, nil
}

func (r foods) GetRevision(ctx context.Context, id food.RevisionID) (food.Revision, error) {
	rev, err := scanRevision(r.q.queryRow(ctx, "SELECT "+revisionColumns+" FROM food_revisions r WHERE r.id = ?", id))
	if err != nil {
		return food.Revision{}, err
	}
	if err := r.loadDetails(ctx, []*food.Revision{&rev}); err != nil {
		return food.Revision{}, err
	}
	return rev, nil
}

func (r foods) ListRevisions(ctx context.Context, viewer user.ID, id food.ID) ([]food.Revision, error) {
	var out []food.Revision
	err := r.q.each(ctx,
		"SELECT "+revisionColumns+" FROM food_revisions r JOIN foods f ON f.id = r.food_id WHERE f.id = ? AND "+visibleTo+" ORDER BY r.number",
		[]any{id, ownerArg(viewer)},
		func(rows *sql.Rows) error {
			rev, err := scanRevision(rows)
			out = append(out, rev)
			return err
		})
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, port.ErrNotFound
	}
	ptrs := make([]*food.Revision, len(out))
	for i := range out {
		ptrs[i] = &out[i]
	}
	return out, r.loadDetails(ctx, ptrs)
}

func (r foods) Search(ctx context.Context, q port.FoodSearchQuery) ([]port.FoodHit, error) {
	if len(q.Terms) == 0 || len(q.Terms[0]) == 0 {
		return nil, nil
	}
	where := "f.deleted_at IS NULL AND f.discontinued_at IS NULL AND " + visibleTo
	args := []any{ownerArg(q.UserID)}
	for _, alternatives := range q.Terms {
		conds := make([]string, 0, len(alternatives)*4)
		for _, key := range alternatives {
			like := "%" + escapeLike(key) + "%"
			conds = append(conds, `r.search_key LIKE ? ESCAPE '\'`, `r.reading_key LIKE ? ESCAPE '\'`)
			args = append(args, like, like)
			// 誤一致対策 短いのは除外
			if utf8.RuneCountInString(key) >= compactMinRunes {
				conds = append(conds, `r.compact_key LIKE ? ESCAPE '\'`, `r.compact_reading LIKE ? ESCAPE '\'`)
				args = append(args, like, like)
			}
		}
		where += " AND (" + strings.Join(conds, " OR ") + ")"
	}
	first := q.Terms[0][0]
	if len(q.Kinds) > 0 {
		where += " AND f.kind IN (" + placeholders(len(q.Kinds)) + ")"
		for _, k := range q.Kinds {
			args = append(args, k)
		}
	}
	args = append(args, first, escapeLike(first)+"%", q.Limit, q.Offset)
	return r.hits(ctx, where+`
		ORDER BY CASE WHEN r.search_key = ? THEN 0 WHEN r.search_key LIKE ? ESCAPE '\' THEN 1 ELSE 2 END,
			length(r.search_key), r.name, f.id
		LIMIT ? OFFSET ?`, args...)
}

func (r foods) SearchSimilar(ctx context.Context, q port.SimilarQuery) ([]port.SimilarHit, error) {
	if len(q.Grams) == 0 || q.Limit <= 0 {
		return nil, nil
	}
	args := make([]any, 0, len(q.Grams)+6)
	for _, g := range q.Grams {
		args = append(args, g)
	}
	args = append(args, max(q.MinShared, 1), q.Limit*4, ownerArg(q.UserID))
	where := "f.deleted_at IS NULL AND f.discontinued_at IS NULL AND " + visibleTo
	if len(q.Kinds) > 0 {
		where += " AND f.kind IN (" + placeholders(len(q.Kinds)) + ")"
		for _, k := range q.Kinds {
			args = append(args, k)
		}
	}
	args = append(args, q.Limit)
	var out []port.SimilarHit
	err := r.q.each(ctx, `
		SELECT `+foodColumns+", "+revisionColumns+`, r.gram_count, x.shared
		FROM (
			SELECT revision_id, COUNT(*) AS shared FROM food_search_grams
			WHERE gram IN (`+placeholders(len(q.Grams))+`)
			GROUP BY revision_id HAVING COUNT(*) >= ?
			ORDER BY shared DESC LIMIT ?
		) x
		JOIN food_revisions r ON r.id = x.revision_id
		JOIN foods f ON f.current_revision_id = r.id
		WHERE `+where+`
		ORDER BY x.shared DESC, r.gram_count, f.id
		LIMIT ?`,
		args,
		func(rows *sql.Rows) error {
			var h port.SimilarHit
			f, rev, err := scanFoodAndRevision(rowsWithExtra{rows, []any{&h.Grams, &h.Shared}})
			h.Food, h.Revision = f, rev
			out = append(out, h)
			return err
		})
	if err != nil {
		return nil, err
	}
	ptrs := make([]*food.Revision, len(out))
	for i := range out {
		ptrs[i] = &out[i].Revision
	}
	return out, r.loadDetails(ctx, ptrs)
}

// 末尾に追加の列を持つ行の読込
type rowsWithExtra struct {
	rows  *sql.Rows
	extra []any
}

func (r rowsWithExtra) Scan(dest ...any) error {
	return r.rows.Scan(append(dest, r.extra...)...)
}

// 空白を除いた検索キーで照合する語の最小文字数
const compactMinRunes = 4

func (r foods) ListOwned(ctx context.Context, owner user.ID) ([]port.FoodHit, error) {
	return r.hits(ctx, "f.owner_id = ? ORDER BY f.created_at", owner)
}

func (r foods) hits(ctx context.Context, where string, args ...any) ([]port.FoodHit, error) {
	var out []port.FoodHit
	err := r.q.each(ctx,
		"SELECT "+foodColumns+", "+revisionColumns+" FROM foods f JOIN food_revisions r ON r.id = f.current_revision_id WHERE "+where,
		args,
		func(rows *sql.Rows) error {
			f, rev, err := scanFoodAndRevision(rows)
			out = append(out, port.FoodHit{Food: f, Revision: rev})
			return err
		})
	if err != nil {
		return nil, err
	}
	ptrs := make([]*food.Revision, len(out))
	for i := range out {
		ptrs[i] = &out[i].Revision
	}
	return out, r.loadDetails(ctx, ptrs)
}

func (r foods) SetDiscontinued(ctx context.Context, id food.ID, at *time.Time) error {
	if at == nil {
		return affectedOne(r.q.exec(ctx, "UPDATE foods SET discontinued_at = NULL WHERE id = ?", id))
	}
	_, err := r.q.exec(ctx, "UPDATE foods SET discontinued_at = ? WHERE id = ? AND discontinued_at IS NULL", r.d.timeValue(*at), id)
	return err
}

func (r foods) SoftDelete(ctx context.Context, id food.ID, at time.Time) error {
	return affectedOne(r.q.exec(ctx, "UPDATE foods SET deleted_at = ? WHERE id = ?", r.d.timeValue(at), id))
}

// 栄養値と単位をまとめて読込
func (r foods) loadDetails(ctx context.Context, revs []*food.Revision) error {
	if len(revs) == 0 {
		return nil
	}
	byID := make(map[food.RevisionID]*food.Revision, len(revs))
	args := make([]any, 0, len(revs))
	for _, rev := range revs {
		rev.Nutrients = nutrient.Profile{}
		rev.Portions = nil
		if _, dup := byID[rev.ID]; !dup {
			args = append(args, rev.ID)
		}
		byID[rev.ID] = rev
	}
	err := inChunks(args, func(chunk []any) error {
		in := placeholders(len(chunk))
		err := r.q.each(ctx,
			"SELECT revision_id, code, amount, status, coverage FROM food_nutrients WHERE revision_id IN ("+in+")", chunk,
			func(rows *sql.Rows) error {
				var (
					id   food.RevisionID
					code nutrient.Code
					v    nutrient.Value
				)
				if err := rows.Scan(&id, &code, decimalCol{&v.Amount}, &v.Status, decimalCol{&v.Coverage}); err != nil {
					return err
				}
				byID[id].Nutrients[code] = v
				return nil
			})
		if err != nil {
			return err
		}
		return r.q.each(ctx,
			"SELECT revision_id, unit, grams FROM food_portions WHERE revision_id IN ("+in+") ORDER BY revision_id, position", chunk,
			func(rows *sql.Rows) error {
				var (
					id food.RevisionID
					p  food.Portion
				)
				if err := rows.Scan(&id, &p.Unit, decimalCol{&p.Grams}); err != nil {
					return err
				}
				byID[id].Portions = append(byID[id].Portions, p)
				return nil
			})
	})
	if err != nil {
		return err
	}
	// 同一リビジョンが複数回現れた場合の共有
	for _, rev := range revs {
		if src := byID[rev.ID]; src != rev {
			rev.Nutrients = src.Nutrients
			rev.Portions = src.Portions
		}
	}
	return nil
}

const chunkSize = 500

// IN句のパラメータ数上限対策
func inChunks(ids []any, fn func(chunk []any) error) error {
	for start := 0; start < len(ids); start += chunkSize {
		end := min(start+chunkSize, len(ids))
		if err := fn(ids[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
