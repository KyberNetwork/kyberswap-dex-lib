package gmx

import (
	"math/big"
)

type IFastPriceFeed interface {
	GetPrice(token string, refPrice *big.Int, maximise bool) *big.Int
}
