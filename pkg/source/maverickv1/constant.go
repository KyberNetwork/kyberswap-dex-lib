package maverickv1

import (
	"math/big"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	DexTypeMaverickV1     = "maverick-v1"
	graphQLRequestTimeout = 20 * time.Second

	poolMethodFee         = "fee"
	poolMethodGetState    = "getState"
	poolMethodBinBalanceA = "binBalanceA"
	poolMethodBinBalanceB = "binBalanceB"
	poolMethodTokenAScale = "tokenAScale"
	poolMethodTokenBScale = "tokenBScale"

	poolMethodGetBin = "getBin"
)

var (
	DefaultGas              = Gas{Swap: 125000}
	zeroBI                  = big.NewInt(0)
	zeroString              = "0"
	defaultTokenWeight uint = 50
)

var (
	MaxTick                     = 460540
	MaxSwapIterationCalculation = 50
	OffsetMask                  = big.NewInt(255)
	Kinds                       = big.NewInt(4)
	Mask                        = big.NewInt(15)
	BitMask                     = new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil), big.NewInt(1))
	WordSize                    = big.NewInt(256)
	One                         = bignumber.BONE
	Unit                        = bignumber.BONE
)
