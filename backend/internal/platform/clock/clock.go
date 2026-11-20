package clock

import (
	"time"

	"github.com/endale-dev/vigil/internal/service"
)

type System struct{}

func (System) Now() time.Time {
	return time.Now()
}

var _ service.Clock = System{}
