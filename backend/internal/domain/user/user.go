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
}

func (u User) IsAdmin() bool {
	return u.Role == RoleAdmin
}
