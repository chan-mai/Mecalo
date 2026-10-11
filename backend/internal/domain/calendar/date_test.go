package calendar

import "testing"

func TestYearsUntil(t *testing.T) {
	birth := MustParse("1990-10-12")
	tests := map[string]int{
		"2026-10-11": 35,
		"2026-10-12": 36,
		"2027-01-01": 36,
	}
	for at, want := range tests {
		if got := birth.YearsUntil(MustParse(at)); got != want {
			t.Errorf("YearsUntil(%s) = %d, want %d", at, got, want)
		}
	}
}

func TestStartOfWeek(t *testing.T) {
	tests := map[string]string{
		"2026-10-12": "2026-10-12",
		"2026-10-18": "2026-10-12",
		"2026-10-14": "2026-10-12",
	}
	for in, want := range tests {
		if got := MustParse(in).StartOfWeek().String(); got != want {
			t.Errorf("StartOfWeek(%s) = %s, want %s", in, got, want)
		}
	}
}

func TestParseRejectsInvalid(t *testing.T) {
	for _, s := range []string{"", "2026-13-01", "2026/10/12"} {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) accepted", s)
		}
	}
}
