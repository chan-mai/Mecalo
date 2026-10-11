// 方言別のスキーマ定義(goose形式)
package migrations

import "embed"

//go:embed sqlite/*.sql
var SQLite embed.FS

//go:embed postgres/*.sql
var Postgres embed.FS
