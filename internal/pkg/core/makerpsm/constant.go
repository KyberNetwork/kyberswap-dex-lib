package makerpsm

import (
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
)

const (
	DAIAddress = "0x6b175474e89094c44da98b954eedeac495271d0f"
)

var (
	DefaultGas = Gas{SellGem: 115000, BuyGem: 115000}
	WAD        = constant.BONE
)
