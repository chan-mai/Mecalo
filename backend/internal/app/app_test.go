package app_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/data"
	"github.com/chan-mai/mecalo/backend/internal/adapter/dri"
	"github.com/chan-mai/mecalo/backend/internal/adapter/password"
	"github.com/chan-mai/mecalo/backend/internal/adapter/reading"
	"github.com/chan-mai/mecalo/backend/internal/adapter/sqlstore"
	"github.com/chan-mai/mecalo/backend/internal/adapter/system"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/body"
	"github.com/chan-mai/mecalo/backend/internal/domain/calendar"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/meal"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/domain/target"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

// MECALO_TEST_POSTGRES_DSN設定時はCockroachDB / PostgreSQLでも実行
func eachStore(t *testing.T, fn func(t *testing.T, s port.Store)) {
	t.Run("sqlite", func(t *testing.T) {
		s, err := sqlstore.OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "test.db"), sqlstore.WithReader(sharedReader))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { s.Close() })
		fn(t, s)
	})
	dsn := os.Getenv("MECALO_TEST_POSTGRES_DSN")
	if dsn == "" {
		return
	}
	t.Run("postgres", func(t *testing.T) {
		s := openPostgres(t, dsn)
		fn(t, s)
	})
}

// 辞書の読込は1回のみ
var sharedReader = reading.New()

var dbSeq struct {
	sync.Mutex
	n int
}

func openPostgres(t *testing.T, dsn string) port.Store {
	t.Helper()
	ctx := context.Background()
	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()

	dbSeq.Lock()
	dbSeq.n++
	name := fmt.Sprintf("mecalo_test_%d_%d", time.Now().UnixNano(), dbSeq.n)
	dbSeq.Unlock()
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		a, err := sql.Open("pgx", dsn)
		if err == nil {
			_, _ = a.ExecContext(ctx, "DROP DATABASE IF EXISTS "+name+" CASCADE")
			a.Close()
		}
	})

	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + name
	s, err := sqlstore.OpenPostgres(ctx, u.String(), sqlstore.WithReader(sharedReader))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

type fixedClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fixedClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(time.Millisecond)
	return c.now
}

type env struct {
	app   *app.App
	store port.Store
	ctx   context.Context
	alice user.User
	bob   user.User
}

var jst = time.FixedZone("JST", 9*60*60)

func newEnv(t *testing.T, s port.Store) *env {
	t.Helper()
	return newEnvWith(t, s, nil)
}

// 外部取得元等の依存を追加
func newEnvWith(t *testing.T, s port.Store, opt func(*app.Deps)) *env {
	t.Helper()
	ref, err := dri.Load(data.DRI2025)
	if err != nil {
		t.Fatal(err)
	}
	deps := app.Deps{
		Store:     s,
		Reader:    sharedReader,
		Clock:     &fixedClock{now: time.Date(2026, 10, 12, 8, 0, 0, 0, jst)},
		IDs:       system.UUIDs{},
		Reference: ref,
		Hasher:    password.Fast(),
	}
	if opt != nil {
		opt(&deps)
	}
	a := app.New(deps, app.Options{RegistrationOpen: true})
	e := &env{app: a, store: s, ctx: context.Background()}

	sess, err := a.Accounts.Setup(e.ctx, app.SignUpInput{LoginName: "alice", Password: "password-a"})
	if err != nil {
		t.Fatal(err)
	}
	e.alice = sess.User
	sess, err = a.Accounts.SignUp(e.ctx, app.SignUpInput{LoginName: "bob", Password: "password-b"})
	if err != nil {
		t.Fatal(err)
	}
	e.bob = sess.User
	return e
}

func d(s string) decimal.Decimal {
	return decimal.RequireFromString(s)
}

func grams(s string) food.Quantity {
	return food.Quantity{Amount: d(s), Unit: food.UnitGram}
}

var today = calendar.MustParse("2026-10-12")

func (e *env) createFood(t *testing.T, owner user.User, name, kcal string) app.FoodView {
	t.Helper()
	v, err := e.app.Foods.Create(e.ctx, owner.ID, app.FoodInput{
		Name: name,
		Nutrients: nutrient.Profile{
			nutrient.Energy:  nutrient.Measured(d(kcal)),
			nutrient.Protein: nutrient.Measured(d("10")),
		},
		Portions: []food.Portion{{Unit: "個", Grams: d("50")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func (e *env) dayEnergy(t *testing.T, owner user.User, date calendar.Date) decimal.Decimal {
	t.Helper()
	rep, err := e.app.Reports.Daily(e.ctx, owner.ID, date)
	if err != nil {
		t.Fatal(err)
	}
	return rep.Summary.Total.Get(nutrient.Energy).Amount
}

func TestPastRecordsDoNotChange(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		e := newEnv(t, s)
		f := e.createFood(t, e.alice, "自家製パン", "250")

		entry, err := e.app.Meals.Add(e.ctx, e.alice.ID, app.AddEntryInput{
			Date: today, Slot: meal.SlotBreakfast, FoodID: f.Food.ID, Quantity: food.Quantity{Amount: d("2"), Unit: "個"},
		})
		if err != nil {
			t.Fatal(err)
		}
		if got := e.dayEnergy(t, e.alice, today); !got.Equal(d("250")) {
			t.Fatalf("energy = %s", got)
		}

		_, err = e.app.Foods.Update(e.ctx, e.alice.ID, f.Food.ID, app.FoodInput{
			Name:      "自家製パン 改",
			Nutrients: nutrient.Profile{nutrient.Energy: nutrient.Measured(d("400"))},
		})
		if err != nil {
			t.Fatal(err)
		}
		if got := e.dayEnergy(t, e.alice, today); !got.Equal(d("250")) {
			t.Errorf("energy changed after food update: %s", got)
		}

		// 単位「個」が新リビジョンにないためグラムで再計算
		re, err := e.app.Meals.Recalculate(e.ctx, e.alice.ID, entry.ID)
		if err != nil {
			t.Fatal(err)
		}
		if re.Quantity.Unit != food.UnitGram || !re.Nutrients.Get(nutrient.Energy).Amount.Equal(d("400")) {
			t.Errorf("recalculated = %+v", re)
		}
		if re.Nutrients.Get(nutrient.Protein).Known() {
			t.Error("protein missing in new revision must be unknown")
		}

		if err := e.app.Foods.Delete(e.ctx, e.alice.ID, f.Food.ID); err != nil {
			t.Fatal(err)
		}
		if got := e.dayEnergy(t, e.alice, today); !got.Equal(d("400")) {
			t.Errorf("energy changed after food delete: %s", got)
		}

		copied, err := e.app.Meals.Copy(e.ctx, e.alice.ID, app.CopyInput{FromDate: today, ToDate: today.AddDays(1)})
		if err != nil {
			t.Fatal(err)
		}
		if len(copied) != 1 || !copied[0].Nutrients.Get(nutrient.Energy).Amount.Equal(d("400")) {
			t.Errorf("copy of deleted food = %+v", copied)
		}
	})
}

func TestUserIsolation(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		e := newEnv(t, s)
		f := e.createFood(t, e.alice, "秘密のおやつ", "100")
		entry, err := e.app.Meals.Add(e.ctx, e.alice.ID, app.AddEntryInput{
			Date: today, Slot: meal.SlotSnack, FoodID: f.Food.ID, Quantity: grams("100"),
		})
		if err != nil {
			t.Fatal(err)
		}

		if _, err := e.app.Foods.Get(e.ctx, e.bob.ID, f.Food.ID); !errors.Is(err, app.ErrNotFound) {
			t.Errorf("bob read alice food: %v", err)
		}
		if _, err := e.app.Meals.Add(e.ctx, e.bob.ID, app.AddEntryInput{
			Date: today, Slot: meal.SlotSnack, FoodID: f.Food.ID, Quantity: grams("100"),
		}); !errors.Is(err, app.ErrNotFound) {
			t.Errorf("bob recorded alice food: %v", err)
		}
		if _, err := e.app.Foods.Update(e.ctx, e.bob.ID, f.Food.ID, app.FoodInput{Name: "x"}); !errors.Is(err, app.ErrNotFound) {
			t.Errorf("bob updated alice food: %v", err)
		}
		if _, err := e.app.Meals.Get(e.ctx, e.bob.ID, entry.ID); !errors.Is(err, app.ErrNotFound) {
			t.Errorf("bob read alice entry: %v", err)
		}
		q := grams("1")
		if _, err := e.app.Meals.Update(e.ctx, e.bob.ID, entry.ID, app.UpdateEntryInput{Quantity: &q}); !errors.Is(err, app.ErrNotFound) {
			t.Errorf("bob updated alice entry: %v", err)
		}
		if err := e.app.Meals.Delete(e.ctx, e.bob.ID, entry.ID); !errors.Is(err, app.ErrNotFound) {
			t.Errorf("bob deleted alice entry: %v", err)
		}
		if got := e.dayEnergy(t, e.bob, today); !got.IsZero() {
			t.Errorf("bob sees energy %s", got)
		}
		hits, err := e.app.Foods.Search(e.ctx, e.bob.ID, "おやつ", nil, 10, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(hits) != 0 {
			t.Errorf("bob found alice food: %+v", hits)
		}
	})
}

func TestSearchSystemFoods(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		e := newEnv(t, s)
		ds := food.Dataset{ID: "mext-test", Kind: food.DatasetMEXT, Name: "test", ImportedAt: time.Now()}
		err := s.InTx(e.ctx, func(r port.Repositories) error {
			if err := r.Datasets().Upsert(e.ctx, ds); err != nil {
				return err
			}
			names := []string{"こめ　［水稲めし］　精白米　うるち米", "コーンフレーク", "ヨーグルト　全脂無糖", "そらまめ　しょうゆ豆", "\uFF08にんじん類\uFF09　にんじん　根　生", "えんばく　オートミール", "＜畜肉類＞　ぶた　［大型種肉］　ロース　脂身つき　生"}
			for i, name := range names {
				id := food.ID(fmt.Sprintf("sys-%d", i))
				var aliases []string
				if i == 5 {
					aliases = []string{"オート", "オーツ"}
				}
				rev := food.Revision{
					ID: food.RevisionID(fmt.Sprintf("sys-rev-%d", i)), FoodID: id, Number: 1, Name: name, Aliases: aliases,
					Nutrients: nutrient.Profile{nutrient.Energy: nutrient.Measured(d("100"))},
					Source:    food.Source{DatasetID: ds.ID, Reference: fmt.Sprintf("%05d", i)},
					CreatedAt: time.Now(),
				}
				f := food.Food{ID: id, Kind: food.KindStandard, CurrentRevisionID: rev.ID, CreatedAt: time.Now()}
				if err := r.Foods().Create(e.ctx, f, rev); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}

		search := func(u user.User, query string) []string {
			t.Helper()
			hits, err := e.app.Foods.Search(e.ctx, u.ID, query, nil, 10, 0)
			if err != nil {
				t.Fatal(err)
			}
			ids := make([]string, 0, len(hits))
			for _, h := range hits {
				ids = append(ids, string(h.Food.ID))
			}
			return ids
		}

		for query, want := range map[string]string{
			"コメ":     "sys-0",
			"めし":     "sys-0",
			"めし 精白米": "sys-0",
			"こーん":    "sys-1",
			"ﾖｰｸﾞﾙﾄ": "sys-2",
			"人参":     "sys-4",
			"オーツ":    "sys-5",
			"豚肉":     "sys-6",
			"豚ロース":   "sys-6",
			"ご飯":     "sys-0",
		} {
			if got := search(e.bob, query); len(got) != 1 || got[0] != want {
				t.Errorf("search %q = %v, want [%s]", query, got, want)
			}
		}

		// 表記も読みも一致しない語は, 選択の記録後に候補へ含める
		if got := search(e.bob, "ライス"); len(got) != 0 {
			t.Errorf("search before selection = %v", got)
		}
		if err := e.app.Foods.RecordSelection(e.ctx, e.bob.ID, "ライス", "sys-0"); err != nil {
			t.Fatal(err)
		}
		if got := search(e.bob, "ライス"); len(got) != 1 || got[0] != "sys-0" {
			t.Errorf("search after selection = %v", got)
		}
		if got := search(e.alice, "ライス"); len(got) != 0 {
			t.Errorf("selection leaked to another user: %v", got)
		}

		history, err := e.app.Foods.SearchHistory(e.ctx, e.bob.ID, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(history) != 10 {
			t.Errorf("history = %v", history)
		}
	})
}

func TestRecipeAndIdempotentAdd(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		e := newEnv(t, s)
		meat := e.createFood(t, e.alice, "豚肉", "200")
		roux := e.createFood(t, e.alice, "ルウ", "500")

		rv, err := e.app.Recipes.Create(e.ctx, e.alice.ID, app.RecipeInput{
			Name: "カレー",
			Ingredients: []app.IngredientInput{
				{FoodID: meat.Food.ID, Quantity: grams("300")},
				{FoodID: roux.Food.ID, Quantity: food.Quantity{Amount: d("2"), Unit: "個"}},
			},
			CookedGrams: d("1000"),
			Servings:    d("4"),
		})
		if err != nil {
			t.Fatal(err)
		}
		got, err := e.app.Recipes.Get(e.ctx, e.alice.ID, rv.Food.Food.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Recipe.Lines) != 2 || !got.Recipe.CookedGrams.Equal(d("1000")) {
			t.Errorf("recipe = %+v", got.Recipe)
		}

		in := app.AddEntryInput{
			Date: today, Slot: meal.SlotDinner, FoodID: rv.Food.Food.ID,
			Quantity: food.Quantity{Amount: d("1"), Unit: "人前"}, ClientRef: "client-1",
		}
		first, err := e.app.Meals.Add(e.ctx, e.alice.ID, in)
		if err != nil {
			t.Fatal(err)
		}
		second, err := e.app.Meals.Add(e.ctx, e.alice.ID, in)
		if err != nil {
			t.Fatal(err)
		}
		if first.ID != second.ID {
			t.Error("client ref did not deduplicate")
		}
		// (600 + 500) kcal / 4人前
		if got := e.dayEnergy(t, e.alice, today); !got.Equal(d("275")) {
			t.Errorf("energy = %s", got)
		}
	})
}

func TestDailyReportWithTargets(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		e := newEnv(t, s)
		rep, err := e.app.Reports.Daily(e.ctx, e.alice.ID, today)
		if err != nil {
			t.Fatal(err)
		}
		if rep.Targets != nil {
			t.Error("targets without profile")
		}

		_, err = e.app.Settings.Save(e.ctx, e.alice.ID, app.SettingsInput{
			EffectiveFrom: today.AddDays(-30),
			Profile: body.Profile{
				Sex: body.SexMale, BirthDate: calendar.MustParse("1990-04-01"), HeightCm: d("170"),
				Activity: body.ActivityModerate, Goal: body.GoalLose, EnergyAdjustKcal: d("-300"),
			},
			Overrides: map[nutrient.Code]target.Override{nutrient.VitaminC: {Min: ptr(d("150"))}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := e.app.Weights.Save(e.ctx, e.alice.ID, body.Weight{Date: today.AddDays(-2), Kg: d("70")}); err != nil {
			t.Fatal(err)
		}

		// 記録前は目標のみ, 判定なし
		rep, err = e.app.Reports.Daily(e.ctx, e.alice.ID, today)
		if err != nil {
			t.Fatal(err)
		}
		if rep.Targets == nil || len(rep.Evaluations) != 0 {
			t.Errorf("before records: targets=%v evaluations=%d", rep.Targets != nil, len(rep.Evaluations))
		}
		empty, err := e.app.Reports.Range(e.ctx, e.alice.ID, today.StartOfWeek(), today.StartOfWeek().AddDays(6))
		if err != nil {
			t.Fatal(err)
		}
		if empty.RecordedDays != 0 || len(empty.AverageEvaluations) != 0 {
			t.Errorf("empty week: recorded=%d evaluations=%d", empty.RecordedDays, len(empty.AverageEvaluations))
		}

		f := e.createFood(t, e.alice, "おにぎり", "180")
		if _, err := e.app.Meals.Add(e.ctx, e.alice.ID, app.AddEntryInput{
			Date: today, Slot: meal.SlotLunch, FoodID: f.Food.ID, Quantity: grams("100"),
		}); err != nil {
			t.Fatal(err)
		}

		rep, err = e.app.Reports.Daily(e.ctx, e.alice.ID, today)
		if err != nil {
			t.Fatal(err)
		}
		if rep.Targets == nil {
			t.Fatal("targets missing")
		}
		// 22.5 * 70 * 1.75 - 300
		if !rep.Targets.EnergyKcal.Equal(d("2456.25")) {
			t.Errorf("energy target = %s", rep.Targets.EnergyKcal)
		}
		levels := map[nutrient.Code]target.Level{}
		for _, ev := range rep.Evaluations {
			levels[ev.Code] = ev.Level
		}
		if levels[nutrient.Energy] != target.LevelLow {
			t.Errorf("energy level = %s", levels[nutrient.Energy])
		}
		if levels[nutrient.VitaminC] != target.LevelUnknown {
			t.Errorf("vitamin c level = %s", levels[nutrient.VitaminC])
		}

		week, err := e.app.Reports.Range(e.ctx, e.alice.ID, today.StartOfWeek(), today.StartOfWeek().AddDays(6))
		if err != nil {
			t.Fatal(err)
		}
		if len(week.Days) != 7 || week.RecordedDays != 1 {
			t.Errorf("week days=%d recorded=%d", len(week.Days), week.RecordedDays)
		}
	})
}

func TestExportImport(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		e := newEnv(t, s)
		f := e.createFood(t, e.alice, "納豆", "190")
		if _, err := e.app.Meals.Add(e.ctx, e.alice.ID, app.AddEntryInput{
			Date: today, Slot: meal.SlotBreakfast, FoodID: f.Food.ID, Quantity: grams("50"),
		}); err != nil {
			t.Fatal(err)
		}
		if err := e.app.Foods.SetFavorite(e.ctx, e.alice.ID, f.Food.ID, true); err != nil {
			t.Fatal(err)
		}
		bundle, err := e.app.Exports.Export(e.ctx, e.alice)
		if err != nil {
			t.Fatal(err)
		}
		if len(bundle.Foods) != 1 || len(bundle.Meals) != 1 || len(bundle.Favorites) != 1 {
			t.Fatalf("bundle = %+v", bundle)
		}

		// 別インスタンスへの移行を想定し全データ削除後に取込
		if err := e.app.Accounts.DeleteUser(e.ctx, e.alice, e.bob.ID); err != nil {
			t.Fatal(err)
		}
		if err := s.Users().Delete(e.ctx, e.alice.ID); err != nil {
			t.Fatal(err)
		}
		sess, err := e.app.Accounts.Setup(e.ctx, app.SignUpInput{LoginName: "carol", Password: "password-c"})
		if err != nil {
			t.Fatal(err)
		}
		res, err := e.app.Exports.Import(e.ctx, sess.User.ID, bundle)
		if err != nil {
			t.Fatal(err)
		}
		if res.Foods != 1 || res.Meals != 1 {
			t.Errorf("import result = %+v", res)
		}
		if got := e.dayEnergy(t, sess.User, today); !got.Equal(d("95")) {
			t.Errorf("energy = %s", got)
		}
	})
}

func TestAccounts(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		e := newEnv(t, s)
		if _, err := e.app.Accounts.Setup(e.ctx, app.SignUpInput{LoginName: "mallory", Password: "password-m"}); !errors.Is(err, app.ErrForbidden) {
			t.Errorf("second setup: %v", err)
		}
		if _, err := e.app.Accounts.LoginLocal(e.ctx, "alice", "nope"); !errors.Is(err, app.ErrUnauthorized) {
			t.Errorf("wrong password: %v", err)
		}
		sess, err := e.app.Accounts.LoginLocal(e.ctx, "ALICE", "password-a")
		if err != nil {
			t.Fatal(err)
		}
		u, err := e.app.Accounts.Authenticate(e.ctx, sess.Token)
		if err != nil || u.ID != e.alice.ID {
			t.Fatalf("authenticate = %v, %v", u, err)
		}
		if err := e.app.Accounts.ChangePassword(e.ctx, u, "password-a", "new-password"); err != nil {
			t.Fatal(err)
		}
		if _, err := e.app.Accounts.LoginLocal(e.ctx, "alice", "new-password"); err != nil {
			t.Errorf("login with new password: %v", err)
		}
		if err := e.app.Accounts.Logout(e.ctx, sess.Token); err != nil {
			t.Fatal(err)
		}
		if _, err := e.app.Accounts.Authenticate(e.ctx, sess.Token); !errors.Is(err, app.ErrUnauthorized) {
			t.Errorf("token valid after logout: %v", err)
		}
		if _, err := e.app.Accounts.ListUsers(e.ctx, e.bob); !errors.Is(err, app.ErrForbidden) {
			t.Errorf("member listed users: %v", err)
		}
	})
}

func ptr[T any](v T) *T {
	return &v
}
