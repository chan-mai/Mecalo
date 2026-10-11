package reading

import "testing"

func TestReading(t *testing.T) {
	k := New()
	tests := map[string]string{
		"人参":      "ニンジン",
		"にんじん":    "ニンジン",
		"豚肉 ロース":  "ブタニク ロース",
		"コーンフレーク": "コーンフレーク",
	}
	for in, want := range tests {
		if got := k.Reading(in); got != want {
			t.Errorf("Reading(%q) = %q, want %q", in, got, want)
		}
	}
}
