package sqlstore

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// SQLの方言差分
type dialect struct {
	name string
	// ? を方言のプレースホルダ変換
	rebind func(query string) string
	// 時刻の書込値
	timeValue func(t time.Time) any
	// 再試行すべきトランザクションエラー
	retryable func(err error) bool
}

// 固定幅のため文字列比較で時刻順
const sqliteTimeLayout = "2006-01-02T15:04:05.000000000Z07:00"

var sqliteDialect = dialect{
	name:   "sqlite",
	rebind: func(q string) string { return q },
	timeValue: func(t time.Time) any {
		return t.UTC().Format(sqliteTimeLayout)
	},
	retryable: func(error) bool { return false },
}

var postgresDialect = dialect{
	name:   "postgres",
	rebind: rebindDollar,
	timeValue: func(t time.Time) any {
		return t.UTC()
	},
	retryable: func(err error) bool {
		var pgErr *pgconn.PgError
		return errors.As(err, &pgErr) && pgErr.Code == "40001"
	},
}

func rebindDollar(q string) string {
	var b strings.Builder
	n := 0
	inQuote := false
	for _, r := range q {
		switch {
		case r == '\'':
			inQuote = !inQuote
			b.WriteRune(r)
		case r == '?' && !inQuote:
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
