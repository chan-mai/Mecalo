package sqlstore

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/domain/calendar"
)

// 方言ごとに異なる列型を共通の型で読み出すScanner群

type timeCol struct{ t *time.Time }

func (c timeCol) Scan(src any) error {
	switch v := src.(type) {
	case time.Time:
		*c.t = v
		return nil
	case string:
		return c.parse(v)
	case []byte:
		return c.parse(string(v))
	}
	return fmt.Errorf("unsupported time value %T", src)
}

func (c timeCol) parse(s string) error {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return err
	}
	*c.t = t
	return nil
}

type nullTimeCol struct{ t **time.Time }

func (c nullTimeCol) Scan(src any) error {
	if src == nil {
		*c.t = nil
		return nil
	}
	var t time.Time
	if err := (timeCol{&t}).Scan(src); err != nil {
		return err
	}
	*c.t = &t
	return nil
}

// NULLはゼロ値
type zeroTimeCol struct{ t *time.Time }

func (c zeroTimeCol) Scan(src any) error {
	if src == nil {
		*c.t = time.Time{}
		return nil
	}
	return timeCol(c).Scan(src)
}

type dateCol struct{ d *calendar.Date }

func (c dateCol) Scan(src any) error {
	var s string
	switch v := src.(type) {
	case time.Time:
		s = v.Format("2006-01-02")
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		return fmt.Errorf("unsupported date value %T", src)
	}
	if len(s) > 10 {
		s = s[:10]
	}
	d, err := calendar.Parse(s)
	if err != nil {
		return err
	}
	*c.d = d
	return nil
}

type decimalCol struct{ d *decimal.Decimal }

func (c decimalCol) Scan(src any) error {
	var err error
	switch v := src.(type) {
	case string:
		*c.d, err = decimal.NewFromString(v)
	case []byte:
		*c.d, err = decimal.NewFromString(string(v))
	case float64:
		*c.d = decimal.NewFromFloat(v)
	case int64:
		*c.d = decimal.NewFromInt(v)
	default:
		err = fmt.Errorf("unsupported decimal value %T", src)
	}
	return err
}

type nullDecimalCol struct{ d **decimal.Decimal }

func (c nullDecimalCol) Scan(src any) error {
	if src == nil {
		*c.d = nil
		return nil
	}
	var d decimal.Decimal
	if err := (decimalCol{&d}).Scan(src); err != nil {
		return err
	}
	*c.d = &d
	return nil
}

type boolCol struct{ b *bool }

func (c boolCol) Scan(src any) error {
	switch v := src.(type) {
	case bool:
		*c.b = v
	case int64:
		*c.b = v != 0
	case string:
		b, err := strconv.ParseBool(v)
		if err != nil {
			return err
		}
		*c.b = b
	default:
		return fmt.Errorf("unsupported bool value %T", src)
	}
	return nil
}

func decimalArg(d decimal.Decimal) driver.Value {
	return d.String()
}

func nullDecimalArg(d *decimal.Decimal) driver.Value {
	if d == nil {
		return nil
	}
	return d.String()
}

func dateArg(d calendar.Date) driver.Value {
	return d.String()
}
