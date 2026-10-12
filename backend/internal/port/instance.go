package port

import "context"

// 管理者がWebUIから変更した設定, キーは環境変数名
type InstanceSettingRepository interface {
	List(ctx context.Context) (map[string]string, error)
	Put(ctx context.Context, values map[string]string) error
	// keysが空なら全件
	Delete(ctx context.Context, keys []string) error
}
