package official

import (
	"github.com/chan-mai/mecalo/backend/internal/adapter/webfetch"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

// MECALO_SOURCESで指定する名前と取得元
type Entry struct {
	Name string
	New  func(*webfetch.Client) port.Catalog
}

// 公式サイトの取得元, 追加時はここへ登録
var Registry = []Entry{
	{"topvalu", func(f *webfetch.Client) port.Catalog { return &Topvalu{Fetch: f} }},
	{"familymart", func(f *webfetch.Client) port.Catalog { return &FamilyMart{Fetch: f} }},
	{"seven-eleven", func(f *webfetch.Client) port.Catalog { return &SevenEleven{Fetch: f} }},
	{"lawson", func(f *webfetch.Client) port.Catalog { return &Lawson{Fetch: f} }},
	{"mcdonalds", func(f *webfetch.Client) port.Catalog { return &McDonalds{Fetch: f} }},
	{"starbucks", func(f *webfetch.Client) port.Catalog { return &Starbucks{Fetch: f} }},
	{"mos", func(f *webfetch.Client) port.Catalog { return &Mos{Fetch: f} }},
	{"ringerhut", func(f *webfetch.Client) port.Catalog { return RingerHut(f) }},
	{"ootoya", func(f *webfetch.Client) port.Catalog { return Ootoya(f) }},
	{"skylark", func(f *webfetch.Client) port.Catalog { return &Skylark{Fetch: f} }},
	{"saizeriya", func(f *webfetch.Client) port.Catalog { return &Saizeriya{Fetch: f} }},
	{"sushiro", func(f *webfetch.Client) port.Catalog { return &Sushiro{Fetch: f} }},
	{"kurasushi", func(f *webfetch.Client) port.Catalog { return &KuraSushi{Fetch: f} }},
	{"godiva", func(f *webfetch.Client) port.Catalog { return &Godiva{Fetch: f} }},
	{"kourakuen", func(f *webfetch.Client) port.Catalog { return &Kourakuen{Fetch: f} }},
	{"matsunoya", func(f *webfetch.Client) port.Catalog { return &Matsunoya{Fetch: f} }},
	{"kiyoken", func(f *webfetch.Client) port.Catalog { return &Kiyoken{Fetch: f} }},
	{"cozycorner", func(f *webfetch.Client) port.Catalog { return &CozyCorner{Fetch: f} }},
	{"horai551", func(f *webfetch.Client) port.Catalog { return &Horai551{Fetch: f} }},
	{"dennys", func(f *webfetch.Client) port.Catalog { return Dennys(f) }},
	{"steakdon", func(f *webfetch.Client) port.Catalog { return SteakDon(f) }},
	{"tonden", func(f *webfetch.Client) port.Catalog { return Tonden(f) }},
	{"burgerking", func(f *webfetch.Client) port.Catalog { return BurgerKing(f) }},
	{"bigboy", func(f *webfetch.Client) port.Catalog { return &BigBoy{Fetch: f} }},
	{"doutor", func(f *webfetch.Client) port.Catalog { return &Doutor{Fetch: f} }},
	{"saintmarc", func(f *webfetch.Client) port.Catalog { return &SaintMarc{Fetch: f} }},
}
