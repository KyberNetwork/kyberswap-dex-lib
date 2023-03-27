package time

import (
	"time"
)

var NowFunc = func() time.Time { return time.Now() }
