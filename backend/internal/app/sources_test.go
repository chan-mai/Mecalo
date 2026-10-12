package app_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/meal"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

var chainDataset = food.Dataset{ID: "testchain", Kind: food.DatasetOfficial, Name: "Test chain"}

type fakeCatalog struct {
	foods []app.SystemFood
	// 詳細を取得できなかった扱いにする食品
	failing []food.ID
	// Freshな食品をSeenで通知した回数
	seen int
}

func (c *fakeCatalog) Dataset() food.Dataset { return chainDataset }

func (c *fakeCatalog) Sync(_ context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	for _, id := range c.failing {
		if err := sink.Failed(id, errors.New("unavailable")); err != nil {
			return prev, err
		}
	}
	for _, sf := range c.foods {
		if sink.Fresh(sf.ID) {
			c.seen++
			if err := sink.Seen(sf.ID); err != nil {
				return prev, err
			}
			continue
		}
		if err := sink.Emit(sf); err != nil {
			return prev, err
		}
	}
	return port.CatalogState{Cursor: "next"}, nil
}

func chainFood(ref, name, kcal string) app.SystemFood {
	return app.SystemFood{
		ID:        food.SystemID("testchain", ref),
		Kind:      food.KindProduct,
		Name:      name,
		Brand:     "テストバーガー",
		Reference: ref,
		Nutrients: nutrient.Profile{nutrient.Energy: nutrient.Measured(d(kcal)), nutrient.Protein: nutrient.Measured(d("20"))},
		Basis:     food.Basis{Unit: "1個"},
	}
}

func TestCatalogSync(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		cat := &fakeCatalog{foods: []app.SystemFood{
			chainFood("1", "テストバーガー", "500"),
			chainFood("2", "テストポテト", "300"),
			{ID: "testchain-3", Name: "壊れた商品", Reference: "3", Nutrients: nutrient.Profile{nutrient.Energy: nutrient.Measured(d("-1"))}},
		}}
		var events []app.SyncEvent
		e := newEnvWith(t, s, func(d *app.Deps) {
			d.Catalogs = []port.Catalog{cat}
			d.OnSync = func(ev app.SyncEvent) { events = append(events, ev) }
		})

		res, err := e.app.Sources.Sync(e.ctx, chainDataset.ID)
		if err != nil {
			t.Fatal(err)
		}
		if res.Created != 2 || res.Skipped != 1 {
			t.Fatalf("first sync = %+v", res)
		}
		if len(events) != 2 || events[0].Phase != app.SyncStarted || events[1].Phase != app.SyncFinished || events[1].Items != 2 || events[1].Result.Created != 2 {
			t.Errorf("events = %+v", events)
		}

		// 重量不明の1個あたりの値を記録
		entry, err := e.app.Meals.Add(e.ctx, e.alice.ID, app.AddEntryInput{
			Date: today, Slot: meal.SlotLunch, FoodID: "testchain-1", Quantity: food.Quantity{Amount: d("2"), Unit: "1個"},
		})
		if err != nil {
			t.Fatal(err)
		}
		if !entry.Grams.IsZero() || !entry.Nutrients.Get(nutrient.Energy).Amount.Equal(d("1000")) {
			t.Fatalf("entry = %+v", entry)
		}
		if _, err := e.app.Meals.Add(e.ctx, e.alice.ID, app.AddEntryInput{
			Date: today, Slot: meal.SlotLunch, FoodID: "testchain-1", Quantity: grams("100"),
		}); err == nil {
			t.Error("grams of unknown-weight food accepted")
		}

		// 詳細取得から期間内は一覧での確認のみ
		if _, err := e.app.Sources.Sync(e.ctx, chainDataset.ID); err != nil {
			t.Fatal(err)
		}
		if cat.seen != 2 {
			t.Errorf("seen = %d, want 2", cat.seen)
		}

		// 2回続けて一覧にない商品は販売終了
		cat.foods = cat.foods[1:2]
		for range 2 {
			if _, err := e.app.Sources.Sync(e.ctx, chainDataset.ID); err != nil {
				t.Fatal(err)
			}
		}
		hits, err := e.app.Foods.Search(e.ctx, e.alice.ID, "テストバーガー", nil, 10, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, h := range hits {
			if h.Food.ID == "testchain-1" {
				t.Error("discontinued food in search results")
			}
		}
		if got := e.dayEnergy(t, e.alice, today); !got.Equal(d("1000")) {
			t.Errorf("energy after discontinuation = %s", got)
		}
		v, err := e.app.Foods.Get(e.ctx, e.alice.ID, "testchain-1")
		if err != nil || !v.Food.Discontinued() {
			t.Fatalf("discontinued food = %+v, %v", v.Food, err)
		}

		// 再掲載で販売終了を解除
		cat.foods = append(cat.foods, chainFood("1", "テストバーガー", "520"))
		res, err = e.app.Sources.Sync(e.ctx, chainDataset.ID)
		if err != nil {
			t.Fatal(err)
		}
		v, err = e.app.Foods.Get(e.ctx, e.alice.ID, "testchain-1")
		if err != nil || v.Food.Discontinued() || v.Revision.Number != 2 {
			t.Errorf("revived food = %+v rev %d, %v (sync %+v)", v.Food, v.Revision.Number, err, res)
		}
		if got := e.dayEnergy(t, e.alice, today); !got.Equal(d("1000")) {
			t.Errorf("energy after revision = %s", got)
		}

		// 取得に失敗した商品は販売終了の対象外
		cat.foods, cat.failing = cat.foods[:1], []food.ID{"testchain-1", "testchain-unknown"}
		for range 3 {
			res, err = e.app.Sources.Sync(e.ctx, chainDataset.ID)
			if err != nil {
				t.Fatal(err)
			}
		}
		if res.Failed != 2 || res.Discontinued != 0 {
			t.Errorf("failed sync = %+v", res)
		}
		if v, _ := e.app.Foods.Get(e.ctx, e.alice.ID, "testchain-1"); v.Food.Discontinued() {
			t.Error("failed food discontinued")
		}

		list, err := e.app.Sources.List(e.ctx)
		if err != nil || len(list) != 1 || list[0].Status.Cursor != "next" || list[0].Status.Items != 3 {
			t.Errorf("sources = %+v, %v", list, err)
		}
	})
}

func TestImportMatchesLegacyID(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		e := newEnv(t, s)
		ds := food.Dataset{ID: "openfoodfacts", Kind: food.DatasetOpenFoodFacts, Name: "Open Food Facts"}
		legacy := app.SystemFood{
			ID: "0f8c2d4e-legacy", Kind: food.KindProduct, Barcode: "4901234567894", Name: "緑茶", Reference: "4901234567894",
			Nutrients: nutrient.Profile{nutrient.Energy: nutrient.Measured(d("0"))},
		}
		if _, err := e.app.Imports.Upsert(e.ctx, ds, []app.SystemFood{legacy}); err != nil {
			t.Fatal(err)
		}
		current := legacy
		current.ID = "off-4901234567894"
		current.Nutrients = nutrient.Profile{nutrient.Energy: nutrient.Measured(d("1"))}
		st, err := e.app.Imports.Upsert(e.ctx, ds, []app.SystemFood{current})
		if err != nil {
			t.Fatal(err)
		}
		if st.Created != 0 || st.Updated != 1 {
			t.Errorf("stats = %+v", st)
		}
		v, err := e.app.Foods.ByBarcode(e.ctx, e.alice.ID, "04901234567894")
		if err != nil || v.Food.ID != legacy.ID {
			t.Errorf("by barcode = %+v, %v", v.Food, err)
		}
	})
}

type fakeFinder struct {
	calls int
}

func (f *fakeFinder) Dataset() food.Dataset {
	return food.Dataset{ID: "testfinder", Kind: food.DatasetTabecal, Name: "Test finder"}
}

func (f *fakeFinder) Find(_ context.Context, query string, _ int) ([]app.SystemFood, error) {
	f.calls++
	sf := chainFood("x1", query+" 並盛", "650")
	sf.ID = "testfinder-x1"
	return []app.SystemFood{sf}, nil
}

func TestExternalSearch(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		finder := &fakeFinder{}
		e := newEnvWith(t, s, func(d *app.Deps) { d.Finders = []port.FoodFinder{finder} })

		res, err := e.app.Sources.Search(e.ctx, "牛丼")
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Hits) != 1 || res.Hits[0].SavedID != "" || res.Hits[0].Food.Name != "牛丼 並盛" {
			t.Fatalf("hits = %+v", res.Hits)
		}
		id, err := e.app.Sources.Save(e.ctx, "testfinder", "x1")
		if err != nil || id != "testfinder-x1" {
			t.Fatalf("save = %s, %v", id, err)
		}
		res, err = e.app.Sources.Search(e.ctx, "牛丼")
		if err != nil {
			t.Fatal(err)
		}
		if finder.calls != 1 {
			t.Errorf("finder called %d times, want cached", finder.calls)
		}
		if res.Hits[0].SavedID != "testfinder-x1" {
			t.Errorf("saved id = %q", res.Hits[0].SavedID)
		}
		if _, err := e.app.Sources.Save(e.ctx, "testfinder", "unknown"); err == nil {
			t.Error("saved uncached item")
		}
	})
}

func TestSearchDedupe(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		e := newEnv(t, s)
		off := food.Dataset{ID: "openfoodfacts", Kind: food.DatasetOpenFoodFacts, Name: "Open Food Facts"}
		tv := food.Dataset{ID: "topvalu", Kind: food.DatasetOfficial, Name: "トップバリュ"}
		energy := nutrient.Profile{nutrient.Energy: nutrient.Measured(d("10"))}
		mk := func(id, ref string) app.SystemFood {
			return app.SystemFood{ID: food.ID(id), Kind: food.KindProduct, Barcode: "4549414730869", Name: "おいしい緑茶", Brand: "トップバリュ", Reference: ref, Nutrients: energy}
		}
		if _, err := e.app.Imports.Upsert(e.ctx, off, []app.SystemFood{mk("off-4549414730869", "4549414730869")}); err != nil {
			t.Fatal(err)
		}
		if _, err := e.app.Imports.Upsert(e.ctx, tv, []app.SystemFood{mk("topvalu-4549414730869", "4549414730869")}); err != nil {
			t.Fatal(err)
		}
		hits, err := e.app.Foods.Search(e.ctx, e.alice.ID, "緑茶", nil, 10, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(hits) != 1 || hits[0].Food.ID != "topvalu-4549414730869" {
			ids := []food.ID{}
			for _, h := range hits {
				ids = append(ids, h.Food.ID)
			}
			t.Errorf("hits = %v", ids)
		}
	})
}

func TestSimilarSearch(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		e := newEnv(t, s)
		ds := food.Dataset{ID: "chains", Kind: food.DatasetOfficial, Name: "chains"}
		energy := nutrient.Profile{nutrient.Energy: nutrient.Measured(d("100"))}
		var foods []app.SystemFood
		for i, name := range []string{"いきなり！ステーキ ワイルドステーキ", "ステーキ丼", "ファミマ・ザ・海苔弁当", "ハンバーガー", "きつねうどん"} {
			ref := fmt.Sprintf("%d", i)
			foods = append(foods, app.SystemFood{ID: food.SystemID("chains", ref), Kind: food.KindProduct, Name: name, Reference: ref, Nutrients: energy})
		}
		if _, err := e.app.Imports.Upsert(e.ctx, ds, foods); err != nil {
			t.Fatal(err)
		}
		search := func(q string) []string {
			t.Helper()
			hits, err := e.app.Foods.Search(e.ctx, e.alice.ID, q, nil, 10, 0)
			if err != nil {
				t.Fatal(err)
			}
			names := make([]string, 0, len(hits))
			for _, h := range hits {
				names = append(names, h.Revision.Name)
			}
			return names
		}
		for q, want := range map[string]string{
			"いきなりステーキ":     "いきなり！ステーキ ワイルドステーキ",
			"ステーキ いきなり":    "いきなり！ステーキ ワイルドステーキ",
			"ファミマザ海苔弁当":    "ファミマ・ザ・海苔弁当",
			"ハンバーガ":        "ハンバーガー",
			"いきなりすてーきわいるど": "いきなり！ステーキ ワイルドステーキ",
		} {
			if got := search(q); len(got) == 0 || got[0] != want {
				t.Errorf("search %q = %v, want first %q", q, got, want)
			}
		}
		if got := search("いきなりステーキ"); len(got) != 1 {
			t.Errorf("unrelated foods matched: %v", got)
		}
		if q := "カレーライス"; len(search(q)) != 0 {
			t.Errorf("search %q = %v", q, search(q))
		}
	})
}

// 同期の開始を止める一覧
type gatedCatalog struct {
	fakeCatalog
	gate chan struct{}
}

func (c *gatedCatalog) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	<-c.gate
	return c.fakeCatalog.Sync(ctx, prev, sink)
}

func TestStartSyncAll(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		cat := &gatedCatalog{fakeCatalog: fakeCatalog{foods: []app.SystemFood{chainFood("1", "テストバーガー", "500")}}, gate: make(chan struct{})}
		done := make(chan app.SyncEvent, 1)
		e := newEnvWith(t, s, func(d *app.Deps) {
			d.Catalogs = []port.Catalog{cat}
			d.OnSync = func(ev app.SyncEvent) {
				if ev.Phase == app.SyncFinished || ev.Phase == app.SyncFailed {
					done <- ev
				}
			}
		})
		if err := e.app.Sources.StartSyncAll(e.ctx, e.bob); !errors.Is(err, app.ErrForbidden) {
			t.Errorf("member: %v", err)
		}
		if err := e.app.Sources.StartSyncAll(e.ctx, e.alice); err != nil {
			t.Fatal(err)
		}
		if err := e.app.Sources.StartSyncAll(e.ctx, e.alice); !errors.Is(err, app.ErrConflict) {
			t.Errorf("while running: %v", err)
		}
		list, err := e.app.Sources.List(e.ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 1 || !list[0].Running {
			t.Errorf("running = %+v", list)
		}
		close(cat.gate)
		if ev := <-done; ev.Phase != app.SyncFinished || ev.Result.Created != 1 {
			t.Errorf("event = %+v", ev)
		}
	})
}
