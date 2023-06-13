package syncswapstable

import (
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

var (
	DefaultGas = Gas{Swap: 0}
	MaxFee     = utils.NewBig("100000")
)
