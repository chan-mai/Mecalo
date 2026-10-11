// argon2idによるパスワードハッシュ
package password

import (
	"github.com/alexedwards/argon2id"

	"github.com/chan-mai/mecalo/backend/internal/port"
)

type Argon2id struct {
	params *argon2id.Params
}

var _ port.PasswordHasher = Argon2id{}

// OWASP推奨の最小構成(m=19MiB, t=2, p=1)
func Default() Argon2id {
	return Argon2id{params: &argon2id.Params{Memory: 19 * 1024, Iterations: 2, Parallelism: 1, SaltLength: 16, KeyLength: 32}}
}

// テスト用の軽量設定
func Fast() Argon2id {
	return Argon2id{params: &argon2id.Params{Memory: 1024, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}}
}

func (a Argon2id) Hash(password string) (string, error) {
	return argon2id.CreateHash(password, a.params)
}

func (a Argon2id) Verify(hash, password string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}
