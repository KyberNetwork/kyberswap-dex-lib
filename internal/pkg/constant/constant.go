package constant

import (
	"math"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
)

const OneHundredPercent = 100

const (
	EmptyHex = "0x"

	DebugHeader = "x-debug"

	AdditionalCostMessageL1Fee = "L1 fee that pays for rolls up cost"
)

type CtxLoggerKeyType string

const (
	// PermitBytesLength The permit can only be empty or 32 * 7 bytes
	// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/974c6c248fd536292c3a9eac7306c62f8bace4da/contracts/dependency/Permitable.sol#L34
	PermitBytesLength = 32 * 7
	maxDecimals       = 60
)

var (
	BoneFloat, _   = new(big.Float).SetString("1000000000000000000")
	Zero           = new(big.Int)
	Ten            = new(big.Float).SetFloat64(10)
	tenPowDecimals []*big.Float
	tenPowInt      []*big.Int
)

func TenPowDecimals(decimal uint8) *big.Float {
	if decimal < maxDecimals {
		return tenPowDecimals[decimal]
	}
	return new(big.Float).SetFloat64(math.Pow10(int(decimal)))
}

var DexUseSwapLimit = []string{
	pooltypes.PoolTypes.KyberPMM,
	pooltypes.PoolTypes.Synthetix,
	pooltypes.PoolTypes.NativeV1,
	pooltypes.PoolTypes.LimitOrder,
	pooltypes.PoolTypes.Dexalot,
	pooltypes.PoolTypes.RingSwap,
	pooltypes.PoolTypes.MxTrading,
	pooltypes.PoolTypes.LO1inch,
	pooltypes.PoolTypes.OneBit,
}

func init() {
	tenPowDecimals = make([]*big.Float, maxDecimals)
	tenPowDecimals[0] = new(big.Float).SetFloat64(1)
	tenPowInt = make([]*big.Int, maxDecimals)
	tenPowInt[0] = big.NewInt(1)
	for i := 1; i < maxDecimals; i++ {
		tenPowDecimals[i] = new(big.Float).Mul(tenPowDecimals[i-1], Ten)
		tenPowInt[i] = new(big.Int).Mul(tenPowInt[i-1], big.NewInt(10))
	}
}
