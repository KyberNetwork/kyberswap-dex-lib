package v3

import (
	"fmt"
	"math/big"
	"strconv"

	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswapv3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ticklens"
)

type (
	SwapInfo struct {
		RemainingAmountIn     *v3Utils.Int256  `json:"rAI,omitempty"`
		NextStateSqrtRatioX96 *v3Utils.Uint160 `json:"nSqrtRx96"`
		nextStateLiquidity    *v3Utils.Uint128
		nextStateTickCurrent  int
	}

	Extra struct {
		Liquidity    *big.Int `json:"liquidity"`
		SqrtPriceX96 *big.Int `json:"sqrtPriceX96"`
		Tick         *big.Int `json:"tick"`
		Ticks        []Tick   `json:"ticks"`
	}

	StaticExtra struct {
		TickSpacing      uint64    `json:"tickSpacing"`
		UnderlyingTokens [2]string `json:"underlyingTokens"`
	}

	ExtraTickU256 struct {
		Liquidity    *uint256.Int `json:"liquidity"`
		SqrtPriceX96 *uint256.Int `json:"sqrtPriceX96"`
		Tick         *int         `json:"tick"`
		Ticks        []TickU256   `json:"ticks"`
	}

	FetchRPCResult struct {
		BlockNumber uint64
		Liquidity   *big.Int
		Slot0       Slot0
		Reserves    [2]*big.Int
		StaticExtra StaticExtra
	}

	Slot0 struct {
		SqrtPriceX96               *big.Int
		Tick                       *big.Int
		ObservationIndex           uint16
		ObservationCardinality     uint16
		ObservationCardinalityNext uint16
		FeeProtocol                uint32
		Unlocked                   bool
	}

	PoolMeta struct {
		SwapFee     uint32       `json:"swapFee"`
		PriceLimit  *uint256.Int `json:"priceLimit"`
		BlockNumber uint64       `json:"blockNumber"`
	}

	Gas = uniswapv3.Gas

	Metadata = uniswapv3.Metadata

	Token = uniswapv3.Token

	SubgraphPool = uniswapv3.SubgraphPool

	TickResp = ticklens.TickResp

	SubgraphPoolTicks = uniswapv3.SubgraphPoolTicks

	Tick = uniswapv3.Tick

	TickU256 = uniswapv3.TickU256
)

func transformTickRespToTick(tickResp TickResp) (Tick, error) {
	tickIdx, err := strconv.Atoi(tickResp.TickIdx)
	if err != nil {
		return Tick{}, fmt.Errorf("can not convert tickIdx string to int, tick: %v", tickResp.TickIdx)
	}

	liquidityGross, ok := new(big.Int).SetString(tickResp.LiquidityGross, 10)
	if !ok {
		return Tick{}, fmt.Errorf("can not convert liquidityGross string to bigInt, tick: %v", tickResp.TickIdx)
	}

	liquidityNet, ok := new(big.Int).SetString(tickResp.LiquidityNet, 10)
	if !ok {
		return Tick{}, fmt.Errorf("can not convert liquidityNet string to bigInt, tick: %v", tickResp.TickIdx)
	}

	return Tick{
		Index:          tickIdx,
		LiquidityGross: liquidityGross,
		LiquidityNet:   liquidityNet,
	}, nil
}
