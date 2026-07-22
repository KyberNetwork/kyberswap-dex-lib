package machima

import (
	"math/big"

	"github.com/pkg/errors"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeMachima

	// Machima is a Uniswap V3 fork. Pools are launched by ClankNow at a single fee tier, but
	// fee/tickSpacing are still read per pool so an additional tier cannot silently mis-price.
	// These are only the fallback when the on-chain read fails.
	defaultFee                = 10000 // 1%, in UniV3 FeeAmount units (hundredths of a bip)
	defaultTickSpacing uint64 = 200   // UniV3 tick spacing for the 1% tier

	// AntiSniperWindowSeconds is the window after pool deployment during which the Machima
	// router rejects swaps.
	AntiSniperWindowSeconds = 600

	bpsDenominator = 10000

	graphFirstLimit = 1000
	rpcChunkSize    = 100
)

const (
	methodFee                = "fee"
	methodTickSpacing        = "tickSpacing"
	methodGetTokenTax        = "getTokenTax"
	methodPoolDeploymentTime = "poolDeploymentTime"
	methodXmaSellPriceLimit  = "xmaSellSqrtPriceLimit"
)

var (
	// defaultGas covers the aggregator router hop on top of the underlying V3 pool swap.
	defaultGas = uniswapv3.Gas{BaseGas: 350000, CrossInitTickGas: 100000}

	ErrAntiSniperActive   = errors.New("pool is in anti-sniper window")
	ErrInvalidPair        = errors.New("invalid pair: exactly one side must be a counter asset")
	ErrOverflow           = errors.New("bigInt overflow uint256")
	ErrZeroAmount         = errors.New("zero amount")
	ErrTaxTooHigh         = errors.New("tax bps >= 100%")
	ErrUnexpectedSwapInfo = errors.New("underlying v3 simulator returned unexpected swapInfo")

	bigBpsDenominator = big.NewInt(bpsDenominator)
)
