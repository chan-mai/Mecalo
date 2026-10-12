package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

// 外部取得元の同期と検索
type Sources struct {
	d       Deps
	imports *Imports
	live    *live
	// 同期中・待機中の取得元と進捗, 値は*syncProgress
	running sync.Map
}

const (
	// 連続して一覧にない回数, 到達で販売終了
	missLimit       = 2
	externalTimeout = 8 * time.Second
	externalTTL     = 24 * time.Hour
	externalLimit   = 30
	// 一覧にある商品の詳細の再取得間隔
	refetchAfter = 30 * 24 * time.Hour
)

type SourceKind string

const (
	SourceLookup  SourceKind = "lookup"
	SourceFinder  SourceKind = "finder"
	SourceCatalog SourceKind = "catalog"
)

type SourceInfo struct {
	Dataset food.Dataset
	Kind    SourceKind
	Status  port.SourceStatus
	Running bool
	// 同期中のみ
	Progress SyncRun
}

type SyncRun struct {
	// ゼロ値は待機中
	StartedAt time.Time
	// 確認した商品数
	Items int
}

type syncProgress struct {
	started atomic.Int64
	items   atomic.Int64
}

func (p *syncProgress) snapshot() SyncRun {
	var out SyncRun
	if n := p.started.Load(); n != 0 {
		out.StartedAt = time.Unix(0, n)
	}
	out.Items = int(p.items.Load())
	return out
}

// 同期中・待機中として登録, 登録済みはfalse
func (s *Sources) reserve(id food.DatasetID) bool {
	_, busy := s.running.LoadOrStore(id, &syncProgress{})
	return !busy
}

func (s *Sources) List(ctx context.Context) ([]SourceInfo, error) {
	statuses, err := s.d.Store.Sources().ListStatus(ctx)
	if err != nil {
		return nil, err
	}
	byID := map[food.DatasetID]port.SourceStatus{}
	for _, st := range statuses {
		byID[st.DatasetID] = st
	}
	cur := s.live.get()
	var out []SourceInfo
	add := func(src port.FoodSource, kind SourceKind) {
		ds := src.Dataset()
		info := SourceInfo{Dataset: ds, Kind: kind, Status: byID[ds.ID]}
		if p, ok := s.running.Load(ds.ID); ok {
			info.Running = true
			info.Progress = p.(*syncProgress).snapshot()
		}
		out = append(out, info)
	}
	if cur.Lookup != nil {
		add(cur.Lookup, SourceLookup)
	}
	for _, f := range cur.Finders {
		add(f, SourceFinder)
	}
	for _, c := range cur.Catalogs {
		add(c, SourceCatalog)
	}
	return out, nil
}

type SyncResult struct {
	ImportStats
	Skipped int
	// 詳細を取得できなかった商品
	Failed       int
	Discontinued int
	// 取得元側で変更なし
	NotModified bool
}

func (s *Sources) catalog(id food.DatasetID) (port.Catalog, bool) {
	for _, c := range s.live.get().Catalogs {
		if c.Dataset().ID == id {
			return c, true
		}
	}
	return nil, false
}

// 管理者のみ
func (s *Sources) ListFor(ctx context.Context, actor user.User) ([]SourceInfo, error) {
	if !actor.IsAdmin() {
		return nil, ErrForbidden
	}
	return s.List(ctx)
}

// バックグラウンドで同期開始, 同期中はErrConflict
func (s *Sources) StartSync(ctx context.Context, actor user.User, id food.DatasetID) error {
	if !actor.IsAdmin() {
		return ErrForbidden
	}
	if _, ok := s.catalog(id); !ok {
		return ErrNotFound
	}
	if !s.reserve(id) {
		return ErrConflict
	}
	go func() {
		defer s.running.Delete(id)
		_, _ = s.sync(context.WithoutCancel(ctx), id)
	}()
	return nil
}

// 一覧の取込, 一覧から消えた商品は販売終了
func (s *Sources) Sync(ctx context.Context, id food.DatasetID) (SyncResult, error) {
	if _, ok := s.catalog(id); !ok {
		return SyncResult{}, ErrNotFound
	}
	if !s.reserve(id) {
		return SyncResult{}, ErrConflict
	}
	defer s.running.Delete(id)
	return s.sync(ctx, id)
}

// 同期の経過の通知
type SyncPhase string

const (
	SyncStarted  SyncPhase = "started"
	SyncProgress SyncPhase = "progress"
	SyncFinished SyncPhase = "finished"
	SyncFailed   SyncPhase = "failed"
)

type SyncEvent struct {
	Source food.DatasetID
	Phase  SyncPhase
	// 一覧で確認した商品数
	Items  int
	Result SyncResult
	Err    error
}

// 進捗を通知する商品数の間隔
const progressEvery = 200

func (s *Sources) report(e SyncEvent) {
	if s.d.OnSync != nil {
		s.d.OnSync(e)
	}
}

func (s *Sources) sync(ctx context.Context, id food.DatasetID) (SyncResult, error) {
	if p, ok := s.running.Load(id); ok {
		p.(*syncProgress).started.Store(s.d.Clock.Now().UnixNano())
	}
	s.report(SyncEvent{Source: id, Phase: SyncStarted})
	res, items, err := s.syncOnce(ctx, id)
	if err != nil {
		s.report(SyncEvent{Source: id, Phase: SyncFailed, Items: items, Result: res, Err: err})
		return res, err
	}
	s.report(SyncEvent{Source: id, Phase: SyncFinished, Items: items, Result: res})
	return res, nil
}

func (s *Sources) syncOnce(ctx context.Context, id food.DatasetID) (SyncResult, int, error) {
	c, _ := s.catalog(id)
	prev, err := s.d.Store.Sources().Status(ctx, id)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return SyncResult{}, 0, err
	}
	started := s.d.Clock.Now()
	ds := c.Dataset()
	ds.ImportedAt = started

	fresh, err := s.d.Store.Sources().FetchedSince(ctx, id, started.Add(-refetchAfter))
	if err != nil {
		return SyncResult{}, 0, err
	}
	sink := &syncSink{ctx: ctx, s: s, ds: ds, at: started, fresh: map[food.ID]bool{}}
	if p, ok := s.running.Load(id); ok {
		sink.progress = p.(*syncProgress)
	}
	for _, fid := range fresh {
		sink.fresh[fid] = true
	}
	state, err := c.Sync(ctx, port.CatalogState{Cursor: prev.Cursor}, sink)
	if err == nil {
		err = sink.flush(ctx)
	}
	res, items := sink.res, sink.items
	status := port.SourceStatus{DatasetID: id, Cursor: prev.Cursor, SyncedAt: started, Items: prev.Items}
	if err != nil {
		status.Error = err.Error()
		if serr := s.d.Store.Sources().SaveStatus(ctx, status); serr != nil {
			return res, items, serr
		}
		return res, items, err
	}

	res.NotModified = state.Unchanged
	if !state.Unchanged && !state.Partial {
		gone, err := s.d.Store.Sources().MarkMissing(ctx, id, started, missLimit)
		if err != nil {
			return res, items, err
		}
		for _, fid := range gone {
			if err := s.d.Store.Foods().SetDiscontinued(ctx, fid, &started); err != nil {
				return res, items, err
			}
		}
		res.Discontinued = len(gone)
		status.Items = items
	}
	if state.Partial {
		status.Items = prev.Items + res.Created
	}
	status.Cursor = state.Cursor
	return res, items, s.d.Store.Sources().SaveStatus(ctx, status)
}

type syncSink struct {
	ctx   context.Context
	s     *Sources
	ds    food.Dataset
	at    time.Time
	fresh map[food.ID]bool
	batch []SystemFood
	seen  []food.ID
	res   SyncResult
	items int
	// nilは進捗の公開なし
	progress *syncProgress
}

var _ port.CatalogSink = (*syncSink)(nil)

func (k *syncSink) Emit(sf SystemFood) error {
	// 不正な商品は読み飛ばし
	if !k.s.imports.valid(k.ds, sf) {
		k.res.Skipped++
		return nil
	}
	k.count()
	k.batch = append(k.batch, sf)
	if len(k.batch) >= importBatchSize {
		return k.flush(k.ctx)
	}
	return nil
}

func (k *syncSink) Fresh(id food.ID) bool {
	return k.fresh[id]
}

func (k *syncSink) Seen(id food.ID) error {
	k.count()
	k.seen = append(k.seen, id)
	return nil
}

// 登録済みの食品は確認済みとして扱い販売終了の判定から除外
func (k *syncSink) Failed(id food.ID, _ error) error {
	k.res.Failed++
	k.count()
	k.seen = append(k.seen, id)
	return nil
}

func (k *syncSink) count() {
	k.items++
	if k.progress != nil {
		k.progress.items.Store(int64(k.items))
	}
	if k.items%progressEvery == 0 {
		k.s.report(SyncEvent{Source: k.ds.ID, Phase: SyncProgress, Items: k.items, Result: k.res})
	}
}

func (k *syncSink) flush(ctx context.Context) error {
	src := k.s.d.Store.Sources()
	if len(k.seen) > 0 {
		if err := src.MarkSeen(ctx, k.ds.ID, k.seen, k.at, false); err != nil {
			return err
		}
		k.seen = k.seen[:0]
	}
	var fetched []food.ID
	st, err := k.s.imports.upsert(ctx, k.ds, k.batch, func(fid food.ID) { fetched = append(fetched, fid) })
	k.res.Created += st.Created
	k.res.Updated += st.Updated
	k.res.Unchanged += st.Unchanged
	k.batch = k.batch[:0]
	if err != nil {
		return err
	}
	return src.MarkSeen(ctx, k.ds.ID, fetched, k.at, true)
}

// 同期中でない全取得元を順にバックグラウンドで同期, 全て同期中ならErrConflict
func (s *Sources) StartSyncAll(ctx context.Context, actor user.User) error {
	if !actor.IsAdmin() {
		return ErrForbidden
	}
	var ids []food.DatasetID
	for _, c := range s.live.get().Catalogs {
		id := c.Dataset().ID
		if s.reserve(id) {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return ErrConflict
	}
	go func() {
		ctx := context.WithoutCancel(ctx)
		for _, id := range ids {
			_, _ = s.sync(ctx, id)
			s.running.Delete(id)
		}
	}()
	return nil
}

// 前回同期から間隔が過ぎた取得元のみ同期, 同期の失敗は状態とOnSyncへ記録
func (s *Sources) SyncDue(ctx context.Context, interval time.Duration) error {
	for _, c := range s.live.get().Catalogs {
		id := c.Dataset().ID
		st, err := s.d.Store.Sources().Status(ctx, id)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
		if err == nil && s.d.Clock.Now().Sub(st.SyncedAt) < interval {
			continue
		}
		_, _ = s.Sync(ctx, id)
	}
	return nil
}

// 外部検索の1件
type ExternalHit struct {
	Food    SystemFood
	Dataset food.Dataset
	// 保存済みの場合の食品ID
	SavedID food.ID
}

type ExternalResult struct {
	Hits []ExternalHit
	// 取得できなかった取得元
	Failed []food.DatasetID
}

// 利用者操作時のみ実行, 結果は一定時間保持
func (s *Sources) Search(ctx context.Context, query string) (ExternalResult, error) {
	query = strings.TrimSpace(query)
	key := food.SearchKey(query)
	if key == "" {
		return ExternalResult{}, invalid(errors.New("query is required"))
	}
	finders := s.live.get().Finders
	if len(finders) == 0 {
		return ExternalResult{}, ErrDisabled
	}
	var out ExternalResult
	for _, f := range finders {
		ds := f.Dataset()
		foods, err := s.find(ctx, f, query, key)
		if err != nil {
			out.Failed = append(out.Failed, ds.ID)
			continue
		}
		for _, sf := range foods {
			hit := ExternalHit{Food: sf, Dataset: ds}
			if fd, _, err := s.d.Store.Foods().FindBySource(ctx, ds.ID, sf.Reference); err == nil {
				hit.SavedID = fd.ID
			} else if !errors.Is(err, ErrNotFound) {
				return ExternalResult{}, err
			}
			out.Hits = append(out.Hits, hit)
		}
	}
	return out, nil
}

func (s *Sources) find(ctx context.Context, f port.FoodFinder, query, key string) ([]SystemFood, error) {
	id := f.Dataset().ID
	now := s.d.Clock.Now()
	if body, err := s.d.Store.Sources().CacheGet(ctx, id, "find:"+key, now); err == nil {
		var foods []SystemFood
		if err := json.Unmarshal(body, &foods); err == nil {
			return foods, nil
		}
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	tctx, cancel := context.WithTimeout(ctx, externalTimeout)
	defer cancel()
	foods, err := f.Find(tctx, query, externalLimit)
	if err != nil {
		return nil, err
	}
	expires := now.Add(externalTTL)
	body, err := json.Marshal(foods)
	if err != nil {
		return nil, err
	}
	if err := s.d.Store.Sources().CachePut(ctx, id, "find:"+key, body, expires); err != nil {
		return nil, err
	}
	for _, sf := range foods {
		item, err := json.Marshal(sf)
		if err != nil {
			return nil, err
		}
		if err := s.d.Store.Sources().CachePut(ctx, id, "item:"+sf.Reference, item, expires); err != nil {
			return nil, err
		}
	}
	return foods, nil
}

// 外部検索で選んだ食品をシステム食品として保存
func (s *Sources) Save(ctx context.Context, id food.DatasetID, reference string) (food.ID, error) {
	var finder port.FoodFinder
	for _, f := range s.live.get().Finders {
		if f.Dataset().ID == id {
			finder = f
		}
	}
	if finder == nil {
		return "", ErrNotFound
	}
	body, err := s.d.Store.Sources().CacheGet(ctx, id, "item:"+reference, s.d.Clock.Now())
	if err != nil {
		return "", err
	}
	var sf SystemFood
	if err := json.Unmarshal(body, &sf); err != nil {
		return "", err
	}
	var saved food.ID
	if _, err := s.imports.upsert(ctx, finder.Dataset(), []SystemFood{sf}, func(fid food.ID) { saved = fid }); err != nil {
		return "", err
	}
	return saved, nil
}

// 期限切れの外部検索結果の削除
func (s *Sources) PurgeCache(ctx context.Context) error {
	return s.d.Store.Sources().CachePurge(ctx, s.d.Clock.Now())
}
