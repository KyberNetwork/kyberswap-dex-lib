package cl

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	DexType = "pancake-infinity-cl"

	graphFirstLimit = 1000

	maxChangedTicks = 10

	clPoolManagerMethodGetLiquidity    = "getLiquidity"
	clPoolManagerMethodGetSlot0        = "getSlot0"
	clPoolManagerMethodGetPoolTickInfo = "getPoolTickInfo"

	OFFSET_TICK_SPACING = 16
)

var (
	Q96 = new(big.Int).Lsh(bignumber.One, 96)

	ErrUnsupportedHook      = errors.New("unsupported hook")
	ErrTooManyChangedTickes = errors.New("too many changed ticks")
)
