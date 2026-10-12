package app

import (
	"context"
	"errors"
	"slices"
	"sort"
	"strings"
	"unicode"

	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

// 手動登録食品の出典データセット
var UserDataset = food.Dataset{
	ID:   "user",
	Kind: food.DatasetUser,
	Name: "User entry",
}

type Foods struct {
	d       Deps
	imports *Imports
}

type FoodView struct {
	Food     food.Food
	Revision food.Revision
	Favorite bool
}

type FoodInput struct {
	Name      string
	Brand     string
	Aliases   []string
	Barcode   string
	Nutrients nutrient.Profile
	Basis     food.Basis
	Portions  []food.Portion
}

const (
	searchLimitMax   = 100
	searchCandidates = 3
	searchLinkedMax  = 5
	frequentDays     = 60
)

func (s *Foods) Search(ctx context.Context, uid user.ID, query string, kinds []food.Kind, limit, offset int) ([]FoodView, error) {
	words := food.SearchTerms(query)
	if len(words) == 0 {
		return []FoodView{}, nil
	}
	if limit <= 0 || limit > searchLimitMax {
		limit = 30
	}

	terms := s.searchTerms(words)
	q := port.FoodSearchQuery{UserID: uid, Terms: terms, Kinds: kinds, Limit: limit * searchCandidates, Offset: offset}
	hits, err := s.d.Store.Foods().Search(ctx, q)
	if err != nil {
		return nil, err
	}
	// 該当なしの場合は漢字1文字ずつの読みで再検索(豚肉→ぶた+にく, ご飯→めし)
	rankKeys := []string{words[0]}
	if len(hits) == 0 && offset == 0 {
		if chars := s.kanjiTerms(words); len(chars) > 0 {
			q.Terms = chars
			if hits, err = s.d.Store.Foods().Search(ctx, q); err != nil {
				return nil, err
			}
			for _, alts := range chars {
				rankKeys = append(rankKeys, alts...)
			}
		}
	}
	favs, err := s.favoriteSet(ctx, uid)
	if err != nil {
		return nil, err
	}
	since := today(s.d.Clock).AddDays(-frequentDays)
	usage, err := s.d.Store.Meals().Frequent(ctx, uid, since, searchLimitMax)
	if err != nil {
		return nil, err
	}
	counts := map[food.ID]int{}
	for _, u := range usage {
		counts[u.FoodID] = u.Count
	}

	views := make([]FoodView, 0, len(hits))
	seen := map[food.ID]bool{}
	for _, h := range hits {
		views = append(views, FoodView{Food: h.Food, Revision: h.Revision, Favorite: favs[h.Food.ID]})
		seen[h.Food.ID] = true
	}

	// 同じ検索語で過去に選んだ食品は文字列が一致しなくても候補に含める
	linked := map[food.ID]int{}
	if offset == 0 {
		ids, err := s.d.Store.QueryLinks().Linked(ctx, uid, food.SearchKey(query), searchLinkedMax)
		if err != nil {
			return nil, err
		}
		for i, id := range ids {
			linked[id] = len(ids) - i
			if seen[id] {
				continue
			}
			f, rev, err := s.d.Store.Foods().Get(ctx, uid, id)
			if errors.Is(err, ErrNotFound) {
				continue
			}
			if err != nil {
				return nil, err
			}
			if f.Deleted() || !kindAllowed(kinds, f.Kind) {
				continue
			}
			views = append(views, FoodView{Food: f, Revision: rev, Favorite: favs[id]})
		}
	}

	priority, err := s.priorities(ctx)
	if err != nil {
		return nil, err
	}
	views = dedupe(views, priority)

	score := func(v FoodView) int {
		sc := counts[v.Food.ID]*10 + linked[v.Food.ID]*200 + (len(sourceOrder)-priority(v))*5
		if v.Favorite {
			sc += 50
		}
		name := food.SearchKey(v.Revision.Name)
		if name == words[0] {
			sc += 100
		}
		// 語単位の一致, 末尾一致(普通牛乳←牛乳)を前方一致より優先
		best := 0
		for _, token := range strings.Fields(name) {
			for _, key := range rankKeys {
				switch {
				case token == key:
					best = max(best, 60)
				case strings.HasSuffix(token, key):
					best = max(best, 30)
				case strings.HasPrefix(token, key):
					best = max(best, 10)
				}
			}
		}
		return sc + best
	}
	sort.SliceStable(views, func(i, j int) bool {
		return score(views[i]) > score(views[j])
	})
	if len(views) > limit {
		views = views[:limit]
	}

	if offset == 0 {
		if err := s.d.Store.SearchHistory().Record(ctx, uid, strings.TrimSpace(query), s.d.Clock.Now()); err != nil {
			return nil, err
		}
	}
	return views, nil
}

// 語ごとの表記と読みの検索キー
func (s *Foods) searchTerms(words []string) [][]string {
	terms := make([][]string, 0, len(words))
	for _, w := range words {
		alts := []string{w}
		if s.d.Reader != nil {
			if r := food.SearchKey(s.d.Reader.Reading(w)); r != "" && r != w {
				alts = append(alts, r)
			}
		}
		terms = append(terms, alts)
	}
	return terms
}

// 検索語を漢字1文字ずつ(表記と読み)と, 2文字以上のかな等の連続へ分割
func (s *Foods) kanjiTerms(words []string) [][]string {
	if s.d.Reader == nil {
		return nil
	}
	var out [][]string
	hasKanji := false
	for _, w := range words {
		var run []rune
		flush := func() {
			if len(run) >= 2 {
				out = append(out, []string{string(run)})
			}
			run = run[:0]
		}
		for _, r := range w {
			if !unicode.Is(unicode.Han, r) {
				run = append(run, r)
				continue
			}
			flush()
			hasKanji = true
			c := string(r)
			alts := []string{c}
			if reading := food.SearchKey(s.d.Reader.Reading(c)); reading != "" && reading != c {
				alts = append(alts, reading)
			}
			out = append(out, alts)
		}
		flush()
	}
	if !hasKanji {
		return nil
	}
	return out
}

// 同一商品の代表を選ぶ順, 先頭ほど優先
var sourceOrder = []food.DatasetKind{food.DatasetUser, food.DatasetOfficial, food.DatasetTabecal, food.DatasetMEXT, food.DatasetOpenFoodFacts}

// 小さいほど優先
func (s *Foods) priorities(ctx context.Context) (func(FoodView) int, error) {
	datasets, err := s.d.Store.Datasets().List(ctx)
	if err != nil {
		return nil, err
	}
	rank := map[food.DatasetID]int{}
	for _, ds := range datasets {
		rank[ds.ID] = len(sourceOrder)
		if i := slices.Index(sourceOrder, ds.Kind); i >= 0 {
			rank[ds.ID] = i
		}
	}
	return func(v FoodView) int {
		if !v.Food.IsSystem() {
			return 0
		}
		if r, ok := rank[v.Revision.Source.DatasetID]; ok {
			return r
		}
		return len(sourceOrder)
	}, nil
}

// JAN, またはブランドと名前が同じシステム食品を1件へ
func dedupe(views []FoodView, priority func(FoodView) int) []FoodView {
	out := make([]FoodView, 0, len(views))
	at := map[string]int{}
	for _, v := range views {
		key := ""
		switch {
		case v.Food.Barcode != "":
			key = "b:" + v.Food.Barcode
		case v.Food.IsSystem() && v.Revision.Brand != "":
			key = "n:" + food.SearchKey(v.Revision.Brand) + "|" + food.SearchKey(v.Revision.Name)
		}
		if key == "" {
			out = append(out, v)
			continue
		}
		if i, ok := at[key]; ok {
			if priority(v) < priority(out[i]) {
				out[i] = v
			}
			continue
		}
		at[key] = len(out)
		out = append(out, v)
	}
	return out
}

func kindAllowed(kinds []food.Kind, k food.Kind) bool {
	return len(kinds) == 0 || slices.Contains(kinds, k)
}

// 検索結果から選んだ食品を検索語と対応付け
func (s *Foods) RecordSelection(ctx context.Context, uid user.ID, query string, id food.ID) error {
	key := food.SearchKey(query)
	if key == "" {
		return invalid(errors.New("query is required"))
	}
	if _, _, err := s.d.Store.Foods().Get(ctx, uid, id); err != nil {
		return err
	}
	return s.d.Store.QueryLinks().Link(ctx, uid, key, id, s.d.Clock.Now())
}

func (s *Foods) Get(ctx context.Context, uid user.ID, id food.ID) (FoodView, error) {
	f, rev, err := s.d.Store.Foods().Get(ctx, uid, id)
	if err != nil {
		return FoodView{}, err
	}
	favs, err := s.favoriteSet(ctx, uid)
	if err != nil {
		return FoodView{}, err
	}
	return FoodView{Food: f, Revision: rev, Favorite: favs[f.ID]}, nil
}

func (s *Foods) Revisions(ctx context.Context, uid user.ID, id food.ID) ([]food.Revision, error) {
	return s.d.Store.Foods().ListRevisions(ctx, uid, id)
}

func (s *Foods) ListOwned(ctx context.Context, uid user.ID) ([]FoodView, error) {
	hits, err := s.d.Store.Foods().ListOwned(ctx, uid)
	if err != nil {
		return nil, err
	}
	favs, err := s.favoriteSet(ctx, uid)
	if err != nil {
		return nil, err
	}
	views := make([]FoodView, 0, len(hits))
	for _, h := range hits {
		views = append(views, FoodView{Food: h.Food, Revision: h.Revision, Favorite: favs[h.Food.ID]})
	}
	return views, nil
}

func (s *Foods) Create(ctx context.Context, uid user.ID, in FoodInput) (FoodView, error) {
	now := s.d.Clock.Now()
	f := food.Food{
		ID:        food.ID(s.d.IDs.NewID()),
		OwnerID:   uid,
		Kind:      food.KindCustom,
		Barcode:   normalizedBarcode(in.Barcode),
		CreatedAt: now,
	}
	rev := s.newRevision(f.ID, 1, in)
	if err := rev.Validate(); err != nil {
		return FoodView{}, invalid(err)
	}
	f.CurrentRevisionID = rev.ID

	err := s.d.Store.InTx(ctx, func(r port.Repositories) error {
		if err := r.Datasets().Upsert(ctx, UserDataset); err != nil {
			return err
		}
		return r.Foods().Create(ctx, f, rev)
	})
	if err != nil {
		return FoodView{}, err
	}
	return FoodView{Food: f, Revision: rev}, nil
}

// 新リビジョンとして追加し既存記録のみ不変
func (s *Foods) Update(ctx context.Context, uid user.ID, id food.ID, in FoodInput) (FoodView, error) {
	var view FoodView
	err := s.d.Store.InTx(ctx, func(r port.Repositories) error {
		f, cur, err := r.Foods().Get(ctx, uid, id)
		if err != nil {
			return err
		}
		if !f.EditableBy(uid) || f.Deleted() || f.Kind == food.KindRecipe {
			return ErrForbidden
		}
		rev := s.newRevision(f.ID, cur.Number+1, in)
		if err := rev.Validate(); err != nil {
			return invalid(err)
		}
		if err := r.Foods().AddRevision(ctx, rev); err != nil {
			return err
		}
		f.CurrentRevisionID = rev.ID
		view = FoodView{Food: f, Revision: rev}
		return nil
	})
	return view, err
}

func (s *Foods) Delete(ctx context.Context, uid user.ID, id food.ID) error {
	return s.d.Store.InTx(ctx, func(r port.Repositories) error {
		f, _, err := r.Foods().Get(ctx, uid, id)
		if err != nil {
			return err
		}
		if !f.EditableBy(uid) {
			return ErrForbidden
		}
		if err := r.Favorites().Remove(ctx, uid, id); err != nil {
			return err
		}
		return r.Foods().SoftDelete(ctx, id, s.d.Clock.Now())
	})
}

// ローカルDB優先, 未登録かつ外部照会が有効な場合のみ照会
func (s *Foods) ByBarcode(ctx context.Context, uid user.ID, code string) (FoodView, error) {
	code, ok := food.NormalizeBarcode(code)
	if !ok {
		return FoodView{}, invalid(errors.New("invalid barcode"))
	}
	f, rev, err := s.d.Store.Foods().FindByBarcode(ctx, uid, code)
	if err == nil {
		return FoodView{Food: f, Revision: rev}, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return FoodView{}, err
	}
	if s.d.Lookup == nil {
		return FoodView{}, ErrNotFound
	}

	sf, err := s.d.Lookup.Lookup(ctx, code)
	if err != nil {
		return FoodView{}, err
	}
	var id food.ID
	if _, err := s.imports.upsert(ctx, s.d.Lookup.Dataset(), []SystemFood{sf}, func(fid food.ID) { id = fid }); err != nil {
		return FoodView{}, err
	}
	return s.Get(ctx, uid, id)
}

// 不正な桁数は入力のまま保存
func normalizedBarcode(code string) string {
	if n, ok := food.NormalizeBarcode(code); ok {
		return n
	}
	return strings.TrimSpace(code)
}

func (s *Foods) SetFavorite(ctx context.Context, uid user.ID, id food.ID, on bool) error {
	if _, _, err := s.d.Store.Foods().Get(ctx, uid, id); err != nil {
		return err
	}
	if on {
		return s.d.Store.Favorites().Add(ctx, uid, id, s.d.Clock.Now())
	}
	return s.d.Store.Favorites().Remove(ctx, uid, id)
}

func (s *Foods) Favorites(ctx context.Context, uid user.ID) ([]FoodView, error) {
	ids, err := s.d.Store.Favorites().List(ctx, uid)
	if err != nil {
		return nil, err
	}
	views := make([]FoodView, 0, len(ids))
	for _, id := range ids {
		f, rev, err := s.d.Store.Foods().Get(ctx, uid, id)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if f.Deleted() {
			continue
		}
		views = append(views, FoodView{Food: f, Revision: rev, Favorite: true})
	}
	return views, nil
}

func (s *Foods) SearchHistory(ctx context.Context, uid user.ID, limit int) ([]string, error) {
	return s.d.Store.SearchHistory().List(ctx, uid, limit)
}

func (s *Foods) ClearSearchHistory(ctx context.Context, uid user.ID) error {
	return s.d.Store.SearchHistory().Clear(ctx, uid)
}

func (s *Foods) Datasets(ctx context.Context) ([]food.Dataset, error) {
	return s.d.Store.Datasets().List(ctx)
}

func (s *Foods) newRevision(id food.ID, number int, in FoodInput) food.Revision {
	return food.Revision{
		ID:        food.RevisionID(s.d.IDs.NewID()),
		FoodID:    id,
		Number:    number,
		Name:      strings.TrimSpace(in.Name),
		Brand:     strings.TrimSpace(in.Brand),
		Aliases:   food.NormalizeAliases(in.Aliases),
		Nutrients: in.Nutrients,
		Basis:     in.Basis,
		Portions:  in.Portions,
		Source:    food.Source{DatasetID: UserDataset.ID},
		CreatedAt: s.d.Clock.Now(),
	}
}

func (s *Foods) favoriteSet(ctx context.Context, uid user.ID) (map[food.ID]bool, error) {
	ids, err := s.d.Store.Favorites().List(ctx, uid)
	if err != nil {
		return nil, err
	}
	set := make(map[food.ID]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set, nil
}
