package mext

import (
	"os"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
)

func TestParseValue(t *testing.T) {
	tests := []struct {
		in     string
		status nutrient.Status
		amount string
	}{
		{"13.5", nutrient.StatusMeasured, "13.5"},
		{"1,452", nutrient.StatusMeasured, "1452"},
		{"(11.3)", nutrient.StatusEstimated, "11.3"},
		{"(0)", nutrient.StatusEstimated, "0"},
		{"Tr", nutrient.StatusTrace, "0"},
		{"(Tr)", nutrient.StatusTrace, "0"},
		{"-", nutrient.StatusMissing, "0"},
		{"", nutrient.StatusMissing, "0"},
	}
	for _, tt := range tests {
		v, err := ParseValue(tt.in)
		if err != nil {
			t.Fatalf("%q: %v", tt.in, err)
		}
		if v.Status != tt.status || v.Amount.String() != tt.amount {
			t.Errorf("%q = %s %s, want %s %s", tt.in, v.Status, v.Amount, tt.status, tt.amount)
		}
	}
	if _, err := ParseValue("abc"); err == nil {
		t.Error("garbage accepted")
	}
}

func TestParseAliases(t *testing.T) {
	got := ParseAliases("※ 耳の割合\uFF1A\u002045 %\n別名\uFF1Aサンドイッチ用食パン\n別名\uFF1A\u0020オート、オーツ\n食物繊維\uFF1AAOAC2011.25法")
	want := []string{"サンドイッチ用食パン", "オート", "オーツ"}
	if len(got) != len(want) {
		t.Fatalf("aliases = %q", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("aliases = %q", got)
		}
	}
}

// MECALO_TEST_MEXT_XLSXに本表Excelのパスを指定した場合のみ実行
func TestParseFile(t *testing.T) {
	path := os.Getenv("MECALO_TEST_MEXT_XLSX")
	if path == "" {
		t.Skip("MECALO_TEST_MEXT_XLSX not set")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	table, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Foods) < 2000 || table.UpdatedAt == "" {
		t.Fatalf("foods=%d updated=%q", len(table.Foods), table.UpdatedAt)
	}
	withAliases := 0
	for _, f := range table.Foods {
		if len(f.Aliases) > 0 {
			withAliases++
		}
	}
	if withAliases < 500 {
		t.Errorf("foods with aliases = %d", withAliases)
	}
	first := table.Foods[0]
	if first.Reference != "01001" || !first.Nutrients.Get(nutrient.Energy).Amount.Equal(decimal.NewFromInt(343)) {
		t.Errorf("first = %+v", first)
	}
	t.Logf("foods=%d updated=%s first=%s", len(table.Foods), table.UpdatedAt, first.Name)
}
