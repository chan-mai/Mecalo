package sqlstore

import (
	"context"
	"database/sql"

	"github.com/chan-mai/mecalo/backend/internal/domain/calendar"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/meal"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/domain/recipe"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

type recipes repos

func (r recipes) Save(ctx context.Context, rev food.RevisionID, res recipe.Result) error {
	_, err := r.q.exec(ctx,
		"INSERT INTO recipe_details (revision_id, raw_grams, cooked_grams, servings) VALUES (?, ?, ?, ?)",
		rev, decimalArg(res.RawGrams), decimalArg(res.CookedGrams), decimalArg(res.Servings))
	if err != nil {
		return err
	}
	for i, l := range res.Lines {
		_, err := r.q.exec(ctx, `
			INSERT INTO recipe_ingredients
				(revision_id, position, food_id, ingredient_revision_id, name, quantity_amount, quantity_unit, grams)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			rev, i, l.FoodID, l.RevisionID, l.Name, decimalArg(l.Quantity.Amount), l.Quantity.Unit, decimalArg(l.Grams))
		if err != nil {
			return err
		}
	}
	return nil
}

func (r recipes) Get(ctx context.Context, rev food.RevisionID) (recipe.Result, error) {
	var res recipe.Result
	err := r.q.queryRow(ctx, "SELECT raw_grams, cooked_grams, servings FROM recipe_details WHERE revision_id = ?", rev).
		Scan(decimalCol{&res.RawGrams}, decimalCol{&res.CookedGrams}, decimalCol{&res.Servings})
	if err != nil {
		return recipe.Result{}, notFound(err)
	}
	rows, err := r.q.query(ctx, `
		SELECT food_id, ingredient_revision_id, name, quantity_amount, quantity_unit, grams
		FROM recipe_ingredients WHERE revision_id = ? ORDER BY position`, rev)
	if err != nil {
		return recipe.Result{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var l recipe.Line
		if err := rows.Scan(&l.FoodID, &l.RevisionID, &l.Name, decimalCol{&l.Quantity.Amount}, &l.Quantity.Unit, decimalCol{&l.Grams}); err != nil {
			return recipe.Result{}, err
		}
		res.Lines = append(res.Lines, l)
	}
	return res, rows.Err()
}

type meals repos

const entryColumns = `e.id, e.user_id, e.date, e.slot, e.food_id, e.food_revision_id, e.food_name, e.brand,
	e.dataset_id, e.source_ref, e.quantity_amount, e.quantity_unit, e.grams, e.client_ref, e.created_at, e.updated_at`

func scanEntry(row interface{ Scan(...any) error }) (meal.Entry, error) {
	var e meal.Entry
	err := row.Scan(&e.ID, &e.UserID, dateCol{&e.Date}, &e.Slot, &e.FoodID, &e.FoodRevisionID, &e.FoodName, &e.Brand,
		&e.Source.DatasetID, &e.Source.Reference, decimalCol{&e.Quantity.Amount}, &e.Quantity.Unit, decimalCol{&e.Grams},
		&e.ClientRef, timeCol{&e.CreatedAt}, timeCol{&e.UpdatedAt})
	return e, notFound(err)
}

func (r meals) Create(ctx context.Context, e meal.Entry) error {
	_, err := r.q.exec(ctx, `
		INSERT INTO meal_entries (id, user_id, date, slot, food_id, food_revision_id, food_name, brand,
			dataset_id, source_ref, quantity_amount, quantity_unit, grams, client_ref, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.UserID, dateArg(e.Date), e.Slot, e.FoodID, e.FoodRevisionID, e.FoodName, e.Brand,
		e.Source.DatasetID, e.Source.Reference, decimalArg(e.Quantity.Amount), e.Quantity.Unit, decimalArg(e.Grams),
		e.ClientRef, r.d.timeValue(e.CreatedAt), r.d.timeValue(e.UpdatedAt))
	if err != nil {
		return err
	}
	return r.insertNutrients(ctx, e)
}

func (r meals) Update(ctx context.Context, e meal.Entry) error {
	err := affectedOne(r.q.exec(ctx, `
		UPDATE meal_entries SET date = ?, slot = ?, food_id = ?, food_revision_id = ?, food_name = ?, brand = ?,
			dataset_id = ?, source_ref = ?, quantity_amount = ?, quantity_unit = ?, grams = ?, updated_at = ?
		WHERE id = ? AND user_id = ?`,
		dateArg(e.Date), e.Slot, e.FoodID, e.FoodRevisionID, e.FoodName, e.Brand,
		e.Source.DatasetID, e.Source.Reference, decimalArg(e.Quantity.Amount), e.Quantity.Unit, decimalArg(e.Grams),
		r.d.timeValue(e.UpdatedAt), e.ID, e.UserID))
	if err != nil {
		return err
	}
	if _, err := r.q.exec(ctx, "DELETE FROM meal_entry_nutrients WHERE entry_id = ?", e.ID); err != nil {
		return err
	}
	return r.insertNutrients(ctx, e)
}

func (r meals) insertNutrients(ctx context.Context, e meal.Entry) error {
	for code, v := range e.Nutrients {
		_, err := r.q.exec(ctx,
			"INSERT INTO meal_entry_nutrients (entry_id, code, amount, status, coverage) VALUES (?, ?, ?, ?, ?)",
			e.ID, code, decimalArg(v.Amount), v.Status, decimalArg(v.Coverage))
		if err != nil {
			return err
		}
	}
	return nil
}

func (r meals) Get(ctx context.Context, owner user.ID, id meal.ID) (meal.Entry, error) {
	return r.one(ctx, "e.id = ? AND e.user_id = ?", id, owner)
}

func (r meals) FindByClientRef(ctx context.Context, owner user.ID, ref string) (meal.Entry, error) {
	return r.one(ctx, "e.user_id = ? AND e.client_ref = ?", owner, ref)
}

func (r meals) one(ctx context.Context, where string, args ...any) (meal.Entry, error) {
	list, err := r.list(ctx, where, args...)
	if err != nil {
		return meal.Entry{}, err
	}
	if len(list) == 0 {
		return meal.Entry{}, port.ErrNotFound
	}
	return list[0], nil
}

func (r meals) Delete(ctx context.Context, owner user.ID, id meal.ID) error {
	return affectedOne(r.q.exec(ctx, "DELETE FROM meal_entries WHERE id = ? AND user_id = ?", id, owner))
}

func (r meals) ListRange(ctx context.Context, owner user.ID, from, to calendar.Date) ([]meal.Entry, error) {
	return r.list(ctx, "e.user_id = ? AND e.date >= ? AND e.date <= ? ORDER BY e.date, e.created_at, e.id",
		owner, dateArg(from), dateArg(to))
}

func (r meals) Recent(ctx context.Context, owner user.ID, limit int) ([]meal.Entry, error) {
	return r.list(ctx, `e.id IN (
			SELECT id FROM (
				SELECT id, food_id, created_at,
					ROW_NUMBER() OVER (PARTITION BY food_id ORDER BY created_at DESC, id DESC) AS rn
				FROM meal_entries WHERE user_id = ?
			) latest WHERE rn = 1
		)
		ORDER BY e.created_at DESC, e.id DESC LIMIT ?`, owner, limit)
}

func (r meals) Frequent(ctx context.Context, owner user.ID, since calendar.Date, limit int) ([]port.FoodUsage, error) {
	rows, err := r.q.query(ctx, `
		SELECT c.food_id, c.n, e.quantity_amount, e.quantity_unit
		FROM (
			SELECT food_id, count(*) AS n, max(created_at) AS last_at
			FROM meal_entries WHERE user_id = ? AND date >= ?
			GROUP BY food_id
		) c
		JOIN meal_entries e ON e.user_id = ? AND e.food_id = c.food_id AND e.created_at = c.last_at
		ORDER BY c.n DESC, c.last_at DESC
		LIMIT ?`, owner, dateArg(since), owner, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []port.FoodUsage
	seen := map[food.ID]bool{}
	for rows.Next() {
		var u port.FoodUsage
		if err := rows.Scan(&u.FoodID, &u.Count, decimalCol{&u.LastQuantity.Amount}, &u.LastQuantity.Unit); err != nil {
			return nil, err
		}
		if seen[u.FoodID] {
			continue
		}
		seen[u.FoodID] = true
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r meals) list(ctx context.Context, where string, args ...any) ([]meal.Entry, error) {
	var out []meal.Entry
	err := r.q.each(ctx, "SELECT "+entryColumns+" FROM meal_entries e WHERE "+where, args, func(rows *sql.Rows) error {
		e, err := scanEntry(rows)
		e.Nutrients = nutrient.Profile{}
		out = append(out, e)
		return err
	})
	if err != nil || len(out) == 0 {
		return out, err
	}

	byID := make(map[meal.ID]*meal.Entry, len(out))
	ids := make([]any, 0, len(out))
	for i := range out {
		byID[out[i].ID] = &out[i]
		ids = append(ids, out[i].ID)
	}
	err = inChunks(ids, func(chunk []any) error {
		return r.q.each(ctx,
			"SELECT entry_id, code, amount, status, coverage FROM meal_entry_nutrients WHERE entry_id IN ("+placeholders(len(chunk))+")",
			chunk,
			func(rows *sql.Rows) error {
				var (
					id   meal.ID
					code nutrient.Code
					v    nutrient.Value
				)
				if err := rows.Scan(&id, &code, decimalCol{&v.Amount}, &v.Status, decimalCol{&v.Coverage}); err != nil {
					return err
				}
				byID[id].Nutrients[code] = v
				return nil
			})
	})
	return out, err
}
