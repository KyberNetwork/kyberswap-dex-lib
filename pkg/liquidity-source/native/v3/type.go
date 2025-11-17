package v3

import (
	"fmt"
	"math/big"
	"strconv"

	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ticklens"
)

type (
	SwapInfo struct {
		LpTokenIn             string           `json:"lpTokenIn"`
		LpTokenOut            string           `json:"lpTokenOut"`
		RemainingAmountIn     *v3Utils.Int256  `json:"rAI,omitempty"`
		NextStateSqrtRatioX96 *v3Utils.Uint160 `json:"nSqrtRx96"`
		nextStateLiquidity    *v3Utils.Uint128
		nextStateTickCurrent  int
	}

	Extra struct {
		Unlocked     bool     `json:"unlocked"`
		Liquidity    *big.Int `json:"liquidity"`
		SqrtPriceX96 *big.Int `json:"sqrtPriceX96"`
		Tick         *big.Int `json:"tick"`
		Ticks        []Tick   `json:"ticks"`
		Vaults       [2]Vault `json:"vaults"`
	}

	StaticExtra struct {
		TickSpacing        uint64 `json:"tickSpacing"`
		NeedScanUnderlying bool   `json:"scan,omitempty"`
	}

	StaticData struct {
		TickSpacing      uint64   `json:"tickSpacing"`
		UnderlyingTokens []string `json:"underlyingTokens"`
	}

	ExtraTickU256 struct {
		Unlocked     bool         `json:"unlocked"`
		Liquidity    *uint256.Int `json:"liquidity"`
		SqrtPriceX96 *uint256.Int `json:"sqrtPriceX96"`
		Tick         *int         `json:"tick"`
		Ticks        []TickU256   `json:"ticks"`
		Vaults       [2]Vault     `json:"vaults"`
	}

	FetchRPCResult struct {
		BlockNumber      uint64
		Liquidity        *big.Int
		Slot0            Slot0
		Reserves         [2]*big.Int
		UnderlyingTokens []common.Address
		Vaults           [2]Vault
	}

	TicksResp struct {
		LiquidityGross                 *big.Int
		LiquidityNet                   *big.Int
		FeeGrowthOutside0X128          *big.Int
		FeeGrowthOutside1X128          *big.Int
		TickCumulativeOutside          *big.Int
		SecondsPerLiquidityOutsideX128 *big.Int
		SecondsOutside                 uint32
		Initialized                    bool
	}

	VaultRPC struct {
		DepositPaused        bool
		RedeemPaused         bool
		MinDeposit           *big.Int
		ExchangeRate         *big.Int
		MinRedeemInterval    *big.Int
		RedeemCoolDownExempt bool
	}

	Vault struct {
		DepositPaused        bool
		RedeemPaused         bool
		MinDeposit           *uint256.Int
		ExchangeRate         *uint256.Int
		MinRedeemInterval    *uint256.Int
		RedeemCoolDownExempt bool
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
		SwapFee         uint32       `json:"swapFee"`
		PriceLimit      *uint256.Int `json:"priceLimit"`
		BlockNumber     uint64       `json:"blockNumber"`
		ApprovalAddress string       `json:"approvalAddress,omitempty"`
	}

	Token struct {
		Address  string `json:"id"`
		Symbol   string `json:"symbol"`
		Decimals string `json:"decimals"`
	}

	SubgraphPool struct {
		ID                 string `json:"id"`
		FeeTier            string `json:"feeTier"`
		PoolType           string `json:"poolType"`
		CreatedAtTimestamp string `json:"createdAtTimestamp"`
		LpToken0           Token  `json:"token0"`
		LpToken1           Token  `json:"token1"`
	}

	Gas = uniswapv3.Gas

	Metadata = uniswapv3.Metadata

	TickResp = ticklens.TickResp

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
