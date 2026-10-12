package food

import "time"

type DatasetID string

type DatasetKind string

const (
	DatasetMEXT          DatasetKind = "mext"
	DatasetOpenFoodFacts DatasetKind = "openfoodfacts"
	DatasetUser          DatasetKind = "user"
	DatasetTabecal       DatasetKind = "tabecal"
	// メーカー・小売・外食チェーンの公式サイト
	DatasetOfficial DatasetKind = "official"
)

// 食品データの取込元と版
type Dataset struct {
	ID      DatasetID
	Kind    DatasetKind
	Name    string
	Version string
	License string
	URL     string
	// 帰属表示文
	Attribution string
	ImportedAt  time.Time
}
