// 時刻, ID採番
package system

import (
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type Clock struct {
	loc atomic.Pointer[time.Location]
}

func NewClock(loc *time.Location) *Clock {
	c := &Clock{}
	c.SetLocation(loc)
	return c
}

func (c *Clock) SetLocation(loc *time.Location) {
	c.loc.Store(loc)
}

func (c *Clock) Now() time.Time {
	return time.Now().In(c.loc.Load())
}

// UUIDv7
type UUIDs struct{}

func (UUIDs) NewID() string {
	return uuid.Must(uuid.NewV7()).String()
}
