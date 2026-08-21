package uniswapv3

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ticklens"
)

type Gas struct {
	BaseGas          int64
	CrossInitTickGas int64

	// CrossEmptyWordGas prices a swap-loop iteration that walks one tick-bitmap word without
	// crossing an initialized tick: a cold SLOAD of the word (2100) plus the tick-price math the
	// step runs regardless. Leaving it at zero keeps the old model, which charged such iterations
	// nothing at all and so priced a swap through a thin pool as if it were free.
	CrossEmptyWordGas int64
}

type SwapInfo struct {
	RemainingAmountIn     *uint256.Int `json:"rAI,omitempty"`
	NextStateSqrtRatioX96 *uint256.Int `json:"nSqrtRx96"`
	NextStateLiquidity    uint256.Int  `json:"-"`
	NextStateTickCurrent  int          `json:"nT"`
}

type Metadata struct {
	LastCreatedAtTimestamp *big.Int `json:"lastCreatedAtTimestamp"`
}

type Token struct {
	Address  string `json:"id"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals string `json:"decimals"`
}

type SubgraphPool struct {
	ID                 string `json:"id"`
	FeeTier            string `json:"feeTier"`
	CreatedAtTimestamp string `json:"createdAtTimestamp"`
	Token0             Token  `json:"token0"`
	Token1             Token  `json:"token1"`
}

type TickResp = ticklens.TickResp

type StaticExtra struct {
	PoolId string `json:"poolId"`
}

type Tick struct {
	Index          int      `json:"index"`
	LiquidityGross *big.Int `json:"liquidityGross"`
	LiquidityNet   *big.Int `json:"liquidityNet"`
}

type TickU256 struct {
	Index          int          `json:"index"`
	LiquidityGross *uint256.Int `json:"liquidityGross"`
	LiquidityNet   *int256.Int  `json:"liquidityNet"`
}

type Extra struct {
	Liquidity    *big.Int `json:"liquidity"`
	SqrtPriceX96 *big.Int `json:"sqrtPriceX96"`
	TickSpacing  uint64   `json:"tickSpacing"`
	Tick         *big.Int `json:"tick"`
	Ticks        []Tick   `json:"ticks"`

	BuyRestrictedToken string `json:"buyRestrictedToken,omitempty"` // for pons-fun
}

type ExtraTickU256 struct {
	Liquidity    *uint256.Int `json:"liquidity"`
	SqrtPriceX96 *uint256.Int `json:"sqrtPriceX96"`
	TickSpacing  uint64       `json:"tickSpacing"`
	Tick         *int         `json:"tick"`
	Ticks        []TickU256   `json:"ticks"`

	BuyRestrictedToken string `json:"buyRestrictedToken,omitempty"`
}

// SimulatorConfig holds construction-time options for NewPoolSimulatorWithExtra.
// It is not stored in the pool simulator struct.
type SimulatorConfig struct {
	AllowEmptyTicks bool
}

// Slot0 is the resolved slot0() result, after trying the standard/slipstream/solidly raw
// shapes (see slot0RawStandard/slot0RawSlipstream/slot0RawSolidly and FetchRPCData).
type Slot0 struct {
	SqrtPriceX96 *big.Int
	Tick         *big.Int
	Unlocked     bool
	// Fee is only non-nil when the pool's slot0() embeds its own fee (solidly-v3 shape);
	// for every other shape the fee comes from fee()/currentFee() instead (see FetchRPCData).
	Fee *big.Int
}

// slot0RawStandard/slot0RawSlipstream/slot0RawSolidly are the raw ethrpc.Call.UnpackABI
// decode targets, tried in that order (longest to shortest) against every pool's slot0()
// return - see abis.Slot0SlipstreamABI/abis.Slot0SolidlyABI and the comment on
// poolCreatedEventIDWithFee in constant.go for why the order matters: a shorter shape can
// silently misdecode a longer pool's trailing fields instead of erroring.
type slot0RawStandard struct {
	SqrtPriceX96               *big.Int
	Tick                       *big.Int
	ObservationIndex           uint16
	ObservationCardinality     uint16
	ObservationCardinalityNext uint16
	FeeProtocol                uint32
	Unlocked                   bool
}

type slot0RawSlipstream struct {
	SqrtPriceX96               *big.Int
	Tick                       *big.Int
	ObservationIndex           uint16
	ObservationCardinality     uint16
	ObservationCardinalityNext uint16
	Unlocked                   bool
}

type slot0RawSolidly struct {
	SqrtPriceX96 *big.Int
	Tick         *big.Int
	Fee          *big.Int
	Unlocked     bool
}

type slot0RawKatana struct {
	SqrtPriceX96               *big.Int
	Tick                       *big.Int
	ObservationIndex           uint16
	ObservationCardinality     uint16
	ObservationCardinalityNext uint16
	FeeProtocol                uint32
	Extra                      *big.Int
	Unlocked                   bool
}

type preGenesisPool struct {
	ID string `json:"id"`
}

type FetchRPCResult struct {
	Liquidity   *big.Int `json:"liquidity"`
	Slot0       Slot0    `json:"slot0"`
	TickSpacing *big.Int `json:"tickSpacing"`
	// Fee is the resolved swap fee: currentFee() if present (dynamic-fee pools such as
	// ramses-v2's dynamic flavor and nuri-v2 additionally expose a stale fee() that must
	// not be used), else fee() (static pools, and dynamic pools like slipstream that only
	// expose a single always-current fee() method), else Slot0.Fee (solidly-v3, which has
	// neither method and embeds fee in slot0()).
	Fee      *big.Int `json:"fee"`
	Reserve0 *big.Int `json:"reserve0"`
	Reserve1 *big.Int `json:"reserve1"`

	// pons-fun on robinhood check
	BuyRestrictedToken string `json:"buyRestrictedToken,omitempty"`
}

type TicksResp struct {
	LiquidityGross *big.Int
	LiquidityNet   *big.Int
}

type PoolMeta struct {
	SwapFee     uint32       `json:"swapFee"`
	PriceLimit  *uint256.Int `json:"priceLimit"`
	BlockNumber uint64       `json:"blockNumber"`
}

func transformTickRespToTick(tickResp TickResp) (Tick, error) {
	liquidityGross, ok := new(big.Int).SetString(tickResp.LiquidityGross, 10)
	if !ok {
		return Tick{}, fmt.Errorf("can not convert liquidityGross string to bigInt, tick: %v", tickResp.TickIdx)
	}

	liquidityNet, ok := new(big.Int).SetString(tickResp.LiquidityNet, 10)
	if !ok {
		return Tick{}, fmt.Errorf("can not convert liquidityNet string to bigInt, tick: %v", tickResp.TickIdx)
	}

	tickIdx, err := strconv.Atoi(tickResp.TickIdx)
	if err != nil {
		return Tick{}, fmt.Errorf("can not convert tickIdx string to int, tick: %v", tickResp.TickIdx)
	}

	return Tick{
		Index:          tickIdx,
		LiquidityGross: liquidityGross,
		LiquidityNet:   liquidityNet,
	}, nil
}
