// 時刻, ID採番
package system

import (
	"time"

	"github.com/google/uuid"
)

type Clock struct {
	Location *time.Location
}

func (c Clock) Now() time.Time {
	return time.Now().In(c.Location)
}

// UUIDv7
type UUIDs struct{}

func (UUIDs) NewID() string {
	return uuid.Must(uuid.NewV7()).String()
}
