package catalog

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
)

func amount(t *testing.T, p nutrient.Profile, code nutrient.Code, want string) {
	t.Helper()
	v, ok := p[code]
	if want == "" {
		if ok {
			t.Errorf("%s = %s, want missing", code, v.Amount)
		}
		return
	}
	if !ok || !v.Amount.Equal(decimal.RequireFromString(want)) {
		t.Errorf("%s = %v (%v), want %s", code, v.Amount, ok, want)
	}
}

func TestParseLabel(t *testing.T) {
	basis, found, v := ParseLabel("1袋(240g)当たり エネルギー322kcal たんぱく質11.0g 脂質5.8g 炭水化物58.1g 糖質55.0g 食物繊維3.1g 食塩相当量2.4g カリウム230mg リン151mg")
	if !found || basis.Unit != "1袋" || !basis.Grams.Equal(decimal.NewFromInt(240)) {
		t.Errorf("basis = %+v", basis)
	}
	p := v.Profile()
	amount(t, p, nutrient.Energy, "322")
	amount(t, p, nutrient.Protein, "11")
	amount(t, p, nutrient.Carbohydrate, "58.1")
	amount(t, p, nutrient.SaltEquivalent, "2.4")
	amount(t, p, nutrient.Potassium, "230")
	amount(t, p, nutrient.Phosphorus, "151")
}

func TestParseLabelVariants(t *testing.T) {
	basis, _, v := ParseLabel("100mlあたり エネルギー 32(Kcal)、たんぱく質 0(g)、脂質 0(g)、炭水化物 8(g)、食塩相当量 0.02(g)")
	if basis.PerUnit() {
		t.Errorf("basis = %+v, want per 100", basis)
	}
	p := v.Profile()
	amount(t, p, nutrient.Energy, "32")
	amount(t, p, nutrient.Protein, "0")
	amount(t, p, nutrient.SaltEquivalent, "0.02")

	basis, found, v := ParseLabel("熱量：1,005kcal、たんぱく質：4.6g、脂質：-、炭水化物：35.9g（糖質：33.6g、食物繊維：2.3g）、食塩相当量：0.89g")
	if found || basis.PerUnit() {
		t.Errorf("basis without text = %+v, %v", basis, found)
	}
	p = v.Profile()
	amount(t, p, nutrient.Energy, "1005")
	amount(t, p, nutrient.Fat, "")
	amount(t, p, nutrient.Carbohydrate, "35.9")
	amount(t, p, nutrient.Fiber, "2.3")

	_, _, v = ParseLabel("エネルギー 250kcal たんぱく質 10g 脂質 0~1g 糖質 30g 食物繊維 2g ナトリウム 393mg")
	p = v.Profile()
	amount(t, p, nutrient.Fat, "0.5")
	if p[nutrient.Fat].Status != nutrient.StatusEstimated {
		t.Error("range should be estimated")
	}
	amount(t, p, nutrient.Carbohydrate, "32")
	amount(t, p, nutrient.SaltEquivalent, "1")
}

func TestBasisOf(t *testing.T) {
	if b := BasisOf("1個", ""); b.Unit != "1個" || !b.Grams.IsZero() {
		t.Errorf("1個 = %+v", b)
	}
	if b := BasisOf("100g", ""); b.PerUnit() {
		t.Errorf("100g = %+v", b)
	}
	if b := BasisOf("50g", ""); b.Unit != "50g" || !b.Grams.Equal(decimal.NewFromInt(50)) {
		t.Errorf("50g = %+v", b)
	}
	if b := BasisOf("", ""); b.Unit != "1食" {
		t.Errorf("empty = %+v", b)
	}
}

func TestNormalize(t *testing.T) {
	if got := Normalize("⽩⽶ ＡＢＣ　１２３ ビッグマック®"); got != "白米 ABC 123 ビッグマック" {
		t.Errorf("Normalize = %q", got)
	}
}
