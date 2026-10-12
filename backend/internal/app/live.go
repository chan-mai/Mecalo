package app

import (
	"sync/atomic"

	"github.com/chan-mai/mecalo/backend/internal/port"
)

// 実行中に差し替える設定と外部取得元
type Live struct {
	Options
	// nilは外部照会無効
	Lookup   port.ProductLookup
	Finders  []port.FoodFinder
	Catalogs []port.Catalog
}

type live struct {
	p atomic.Pointer[Live]
}

func newLive(d Deps, opt Options) *live {
	l := &live{}
	l.p.Store(&Live{Options: opt, Lookup: d.Lookup, Finders: d.Finders, Catalogs: d.Catalogs})
	return l
}

func (l *live) get() *Live {
	return l.p.Load()
}

// 設定と外部取得元の差し替え, 実行中の処理は旧設定のまま
func (a *App) Reconfigure(v Live) {
	a.live.p.Store(&v)
}
