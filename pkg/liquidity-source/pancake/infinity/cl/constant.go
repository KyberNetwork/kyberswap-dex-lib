package cl

import (
	"errors"
	"math/big"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	DexType = "pancake-infinity-cl"

	graphFirstLimit = 1000
	maxChangedTicks = 10
	tickChunkSize   = 100

	_OFFSET_TICK_SPACING = 16

	_MASK12 = 0xfff
)

const (
	getPoolTickInfoMethod = "getPoolTickInfo"
)

var (
	Q96     = new(big.Int).Lsh(bignumber.One, 96)
	_MASK24 = uint256.NewInt(0xffffff)

	ErrTooManyChangedTickes = errors.New("too many changed ticks")
)
