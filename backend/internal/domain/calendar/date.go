// タイムゾーンなし暦日
package calendar

import (
	"fmt"
	"time"
)

const layout = "2006-01-02"

// "YYYY-MM-DD" 形式の暦日
type Date struct {
	t time.Time
}

func Parse(s string) (Date, error) {
	t, err := time.Parse(layout, s)
	if err != nil {
		return Date{}, fmt.Errorf("invalid date %q: %w", s, err)
	}
	return Date{t: t}, nil
}

func MustParse(s string) Date {
	d, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return d
}

func Of(year int, month time.Month, day int) Date {
	return Date{t: time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

// 指定ロケーションでの暦日
func In(t time.Time, loc *time.Location) Date {
	y, m, d := t.In(loc).Date()
	return Of(y, m, d)
}

func (d Date) String() string {
	return d.t.Format(layout)
}

// UTC 0時の時刻
func (d Date) Time() time.Time {
	return d.t
}

func (d Date) IsZero() bool {
	return d.t.IsZero()
}

func (d Date) AddDays(n int) Date {
	return Date{t: d.t.AddDate(0, 0, n)}
}

func (d Date) Equal(o Date) bool {
	return d.t.Equal(o.t)
}

func (d Date) Before(o Date) bool {
	return d.t.Before(o.t)
}

func (d Date) After(o Date) bool {
	return d.t.After(o.t)
}

// dからoまでの日数
func (d Date) DaysUntil(o Date) int {
	return int(o.t.Sub(d.t).Hours() / 24)
}

// dからoまでの満年数
func (d Date) YearsUntil(o Date) int {
	years := o.t.Year() - d.t.Year()
	if o.t.Month() < d.t.Month() || (o.t.Month() == d.t.Month() && o.t.Day() < d.t.Day()) {
		years--
	}
	return years
}

// 月曜始まりの週初日
func (d Date) StartOfWeek() Date {
	offset := (int(d.t.Weekday()) + 6) % 7
	return d.AddDays(-offset)
}
