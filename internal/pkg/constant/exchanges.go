package constant

import (
	"time"
)

var (
	DefaultDeadlineInMinute time.Duration = time.Minute * 20
	MaximumSlippage         int64         = 2000
	MaxAmountInUSD          float64       = 100000000
)
