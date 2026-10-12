package openfoodfacts

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/chan-mai/mecalo/backend/internal/adapter/webfetch"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

const DefaultDeltaBase = "https://static.openfoodfacts.org/data/delta"

// 日次差分ファイルの取込, 初回は全量ダンプの取込が前提
type Delta struct {
	Base    string
	Country string
	Fetch   *webfetch.Client
}

var _ port.Catalog = (*Delta)(nil)

var deltaName = regexp.MustCompile(`^openfoodfacts_products_(\d+)_(\d+)\.json\.gz$`)

func (d *Delta) Dataset() food.Dataset {
	return Dataset()
}

// Cursorは取込済みの差分の終了時刻(Unix秒)
func (d *Delta) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	base := strings.TrimRight(d.Base, "/")
	if base == "" {
		base = DefaultDeltaBase
	}
	idx, err := d.Fetch.Get(ctx, base+"/index.txt", webfetch.Validators{})
	if err != nil {
		return prev, err
	}
	done, _ := strconv.ParseInt(prev.Cursor, 10, 64)

	type file struct {
		name string
		end  int64
	}
	var files []file
	for _, line := range strings.Split(string(idx.Body), "\n") {
		m := deltaName.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		end, _ := strconv.ParseInt(m[2], 10, 64)
		if end > done {
			files = append(files, file{name: m[0], end: end})
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].end < files[j].end })

	state := port.CatalogState{Cursor: prev.Cursor, Partial: true, Unchanged: len(files) == 0}
	for _, f := range files {
		body, err := d.Fetch.Open(ctx, base+"/"+f.name)
		if err != nil {
			return state, err
		}
		err = ReadDump(body, d.country(), sink.Emit)
		body.Close()
		if err != nil {
			return state, err
		}
		state.Cursor = strconv.FormatInt(f.end, 10)
	}
	return state, nil
}

func (d *Delta) country() string {
	if d.Country == "" {
		return "japan"
	}
	return d.Country
}
