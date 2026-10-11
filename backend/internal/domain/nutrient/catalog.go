// 栄養素の定義と栄養値の計算
package nutrient

type Code string

type Unit string

const (
	UnitKcal      Unit = "kcal"
	UnitGram      Unit = "g"
	UnitMilligram Unit = "mg"
	UnitMicrogram Unit = "ug"
)

type Group string

const (
	GroupEnergy  Group = "energy"
	GroupMacro   Group = "macro"
	GroupMineral Group = "mineral"
	GroupVitamin Group = "vitamin"
)

type Definition struct {
	Code  Code
	Unit  Unit
	Group Group
	// 日本食品標準成分表の成分識別子
	Infoods string
}

const (
	Energy          Code = "energy"
	Protein         Code = "protein"
	Fat             Code = "fat"
	Carbohydrate    Code = "carbohydrate"
	Fiber           Code = "fiber"
	SaltEquivalent  Code = "salt_equivalent"
	Cholesterol     Code = "cholesterol"
	Sodium          Code = "sodium"
	Potassium       Code = "potassium"
	Calcium         Code = "calcium"
	Magnesium       Code = "magnesium"
	Phosphorus      Code = "phosphorus"
	Iron            Code = "iron"
	Zinc            Code = "zinc"
	Copper          Code = "copper"
	Manganese       Code = "manganese"
	Iodine          Code = "iodine"
	Selenium        Code = "selenium"
	Chromium        Code = "chromium"
	Molybdenum      Code = "molybdenum"
	VitaminA        Code = "vitamin_a"
	VitaminD        Code = "vitamin_d"
	VitaminE        Code = "vitamin_e"
	VitaminK        Code = "vitamin_k"
	VitaminB1       Code = "vitamin_b1"
	VitaminB2       Code = "vitamin_b2"
	Niacin          Code = "niacin_equivalent"
	VitaminB6       Code = "vitamin_b6"
	VitaminB12      Code = "vitamin_b12"
	Folate          Code = "folate"
	PantothenicAcid Code = "pantothenic_acid"
	Biotin          Code = "biotin"
	VitaminC        Code = "vitamin_c"
)

// 表示順を兼ねる
var catalog = []Definition{
	{Energy, UnitKcal, GroupEnergy, "ENERC_KCAL"},
	{Protein, UnitGram, GroupMacro, "PROT-"},
	{Fat, UnitGram, GroupMacro, "FAT-"},
	{Carbohydrate, UnitGram, GroupMacro, "CHOCDF-"},
	{Fiber, UnitGram, GroupMacro, "FIB-"},
	{SaltEquivalent, UnitGram, GroupMacro, "NACL_EQ"},
	{Cholesterol, UnitMilligram, GroupMacro, "CHOLE"},
	{Sodium, UnitMilligram, GroupMineral, "NA"},
	{Potassium, UnitMilligram, GroupMineral, "K"},
	{Calcium, UnitMilligram, GroupMineral, "CA"},
	{Magnesium, UnitMilligram, GroupMineral, "MG"},
	{Phosphorus, UnitMilligram, GroupMineral, "P"},
	{Iron, UnitMilligram, GroupMineral, "FE"},
	{Zinc, UnitMilligram, GroupMineral, "ZN"},
	{Copper, UnitMilligram, GroupMineral, "CU"},
	{Manganese, UnitMilligram, GroupMineral, "MN"},
	{Iodine, UnitMicrogram, GroupMineral, "ID"},
	{Selenium, UnitMicrogram, GroupMineral, "SE"},
	{Chromium, UnitMicrogram, GroupMineral, "CR"},
	{Molybdenum, UnitMicrogram, GroupMineral, "MO"},
	{VitaminA, UnitMicrogram, GroupVitamin, "VITA_RAE"},
	{VitaminD, UnitMicrogram, GroupVitamin, "VITD"},
	{VitaminE, UnitMilligram, GroupVitamin, "TOCPHA"},
	{VitaminK, UnitMicrogram, GroupVitamin, "VITK"},
	{VitaminB1, UnitMilligram, GroupVitamin, "THIA"},
	{VitaminB2, UnitMilligram, GroupVitamin, "RIBF"},
	{Niacin, UnitMilligram, GroupVitamin, "NE"},
	{VitaminB6, UnitMilligram, GroupVitamin, "VITB6A"},
	{VitaminB12, UnitMicrogram, GroupVitamin, "VITB12"},
	{Folate, UnitMicrogram, GroupVitamin, "FOL"},
	{PantothenicAcid, UnitMilligram, GroupVitamin, "PANTAC"},
	{Biotin, UnitMicrogram, GroupVitamin, "BIOT"},
	{VitaminC, UnitMilligram, GroupVitamin, "VITC"},
}

var byCode = func() map[Code]Definition {
	m := make(map[Code]Definition, len(catalog))
	for _, d := range catalog {
		m[d.Code] = d
	}
	return m
}()

func Catalog() []Definition {
	out := make([]Definition, len(catalog))
	copy(out, catalog)
	return out
}

func Lookup(code Code) (Definition, bool) {
	d, ok := byCode[code]
	return d, ok
}

func ByInfoods(tag string) (Definition, bool) {
	for _, d := range catalog {
		if d.Infoods == tag {
			return d, true
		}
	}
	return Definition{}, false
}
