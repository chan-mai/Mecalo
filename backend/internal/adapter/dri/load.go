// 食事摂取基準YAMLの読込
package dri

import (
	"fmt"

	"github.com/shopspring/decimal"
	"go.yaml.in/yaml/v3"

	"github.com/chan-mai/mecalo/backend/internal/domain/body"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/domain/target"
)

type document struct {
	Edition string `yaml:"edition"`
	Title   string `yaml:"title"`
	URL     string `yaml:"url"`
	Bands   []struct {
		Label string `yaml:"label"`
		Min   int    `yaml:"min"`
		Max   int    `yaml:"max"`
	} `yaml:"bands"`
	Energy struct {
		BMRPerKg struct {
			Male   map[string]decimal.Decimal `yaml:"male"`
			Female map[string]decimal.Decimal `yaml:"female"`
		} `yaml:"bmr_per_kg"`
		PAL struct {
			Bands map[string]map[string]*decimal.Decimal `yaml:"bands"`
		} `yaml:"pal"`
	} `yaml:"energy"`
	Nutrients map[string]nutrientDoc `yaml:"nutrients"`
}

type nutrientDoc struct {
	Unit   string                       `yaml:"unit"`
	Source string                       `yaml:"source"`
	Note   string                       `yaml:"note"`
	Groups map[string]map[string]values `yaml:",inline"`
}

type values struct {
	EAR         *decimal.Decimal  `yaml:"ear"`
	RDA         *decimal.Decimal  `yaml:"rda"`
	AI          *decimal.Decimal  `yaml:"ai"`
	UL          *decimal.Decimal  `yaml:"ul"`
	DGMin       *decimal.Decimal  `yaml:"dg_min"`
	DGMax       *decimal.Decimal  `yaml:"dg_max"`
	DGEnergyPct []decimal.Decimal `yaml:"dg_energy_pct"`
	Note        string            `yaml:"note"`
}

func Load(src []byte) (target.Reference, error) {
	var doc document
	if err := yaml.Unmarshal(src, &doc); err != nil {
		return target.Reference{}, fmt.Errorf("parse dri: %w", err)
	}

	ref := target.Reference{
		Edition:   doc.Edition,
		Title:     doc.Title,
		URL:       doc.URL,
		BMRPerKg:  map[body.Sex]map[string]decimal.Decimal{body.SexMale: doc.Energy.BMRPerKg.Male, body.SexFemale: doc.Energy.BMRPerKg.Female},
		PAL:       map[string]map[body.ActivityLevel]decimal.Decimal{},
		Nutrients: map[nutrient.Code]target.NutrientReference{},
	}
	for _, b := range doc.Bands {
		ref.Bands = append(ref.Bands, target.AgeBand{Label: b.Label, Min: b.Min, Max: b.Max})
	}
	for band, levels := range doc.Energy.PAL.Bands {
		m := map[body.ActivityLevel]decimal.Decimal{}
		for level, v := range levels {
			if v != nil {
				m[body.ActivityLevel(level)] = *v
			}
		}
		ref.PAL[band] = m
	}

	for code, n := range doc.Nutrients {
		def, ok := nutrient.Lookup(nutrient.Code(code))
		if !ok {
			return target.Reference{}, fmt.Errorf("unknown nutrient %q", code)
		}
		if string(def.Unit) != n.Unit {
			return target.Reference{}, fmt.Errorf("%s: unit %q does not match catalog %q", code, n.Unit, def.Unit)
		}
		nr := target.NutrientReference{Source: n.Source, Note: n.Note, Values: map[target.Group]map[string]target.Indicators{}}
		for group, bands := range n.Groups {
			g := target.Group(group)
			switch g {
			case target.GroupMale, target.GroupFemale, target.GroupFemaleMenstruating:
			default:
				return target.Reference{}, fmt.Errorf("%s: unknown group %q", code, group)
			}
			byBand := map[string]target.Indicators{}
			for band, v := range bands {
				ind := target.Indicators{EAR: v.EAR, RDA: v.RDA, AI: v.AI, UL: v.UL, DGMin: v.DGMin, DGMax: v.DGMax}
				if len(v.DGEnergyPct) == 2 {
					ind.DGEnergyPctMin = &v.DGEnergyPct[0]
					ind.DGEnergyPctMax = &v.DGEnergyPct[1]
				}
				byBand[band] = ind
			}
			nr.Values[g] = byBand
		}
		ref.Nutrients[def.Code] = nr
	}
	return ref, nil
}
