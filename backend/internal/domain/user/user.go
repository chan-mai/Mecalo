// 利用者
package user

import "time"

type ID string

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

type User struct {
	ID          ID
	DisplayName string
	Role        Role
	CreatedAt   time.Time
	Onboarded   bool
}

func (u User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// APIトークンで許可する操作
type Scope string

const (
	ScopeRead  Scope = "read"
	ScopeWrite Scope = "write"
)

func (s Scope) Valid() bool {
	return s == ScopeRead || s == ScopeWrite
}

// writeはreadを含む
func (s Scope) Allows(need Scope) bool {
	return s == ScopeWrite || s == need
}
