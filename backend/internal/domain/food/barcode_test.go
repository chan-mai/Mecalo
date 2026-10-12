package food

import "testing"

func TestNormalizeBarcode(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"4902777015927", "4902777015927", true},
		{" 4902777015927 ", "4902777015927", true},
		{"49698626", "49698626", true},
		{"0000049698626", "49698626", true},
		{"04902777015927", "4902777015927", true},
		{"00000049698626", "49698626", true},
		{"012345678905", "0012345678905", true},
		{"14902102151303", "14902102151303", true},
		{"4902-777", "", false},
		{"1234567", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		got, ok := NormalizeBarcode(tt.in)
		if got != tt.want || ok != tt.ok {
			t.Errorf("NormalizeBarcode(%q) = %q, %v, want %q, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestValidBarcode(t *testing.T) {
	for code, want := range map[string]bool{
		"4902777015927": true,
		"49698626":      true,
		"4909411084950": true,
		"4902777015928": false,
		"0000049698626": false,
	} {
		if got := ValidBarcode(code); got != want {
			t.Errorf("ValidBarcode(%q) = %v, want %v", code, got, want)
		}
	}
}
