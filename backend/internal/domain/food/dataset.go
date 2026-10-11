package food

import "time"

type DatasetID string

type DatasetKind string

const (
	DatasetMEXT          DatasetKind = "mext"
	DatasetOpenFoodFacts DatasetKind = "openfoodfacts"
	DatasetUser          DatasetKind = "user"
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
