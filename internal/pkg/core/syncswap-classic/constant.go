package syncswapclassic

import (
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

var (
	DefaultGas = Gas{Swap: 300000}
	MaxFee     = utils.NewBig("100000")
)
