package uniswaplo

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type makerAndAsset = string

func newMakerAndAsset(maker, makerAsset string) makerAndAsset {
	return fmt.Sprintf("%v:%v", maker, makerAsset)
}

type PoolSimulator struct {
	pool.Pool

	// static extra fields
	token0 string
	token1 string

	// extra fields
	takeToken0Orders []*DutchOrder
	takeToken1Orders []*DutchOrder

	takeToken0OrdersMapping map[string]int
	takeToken1OrdersMapping map[string]int

	reactorAddress string
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var numTokens = len(entityPool.Tokens)
	var tokens = make([]string, numTokens)
	var reserves = make([]*big.Int, numTokens)
	if numTokens != 2 {
		return nil, fmt.Errorf("pool's number of tokens should equal 2")
	}
	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	takeToken0OrdersMapping := make(map[string]int, len(extra.TakeToken0Orders))
	takeToken1OrdersMapping := make(map[string]int, len(extra.TakeToken1Orders))

	for i, takeToken0Order := range extra.TakeToken0Orders {
		takeToken0OrdersMapping[takeToken0Order.OrderHash] = i
	}

	for i, takeToken1Order := range extra.TakeToken1Orders {
		takeToken1OrdersMapping[takeToken1Order.OrderHash] = i
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				SwapFee:    integer.Zero(),
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reserves,
				Checked:    false,
			},
		},
		token0:                  staticExtra.Token0,
		token1:                  staticExtra.Token1,
		takeToken0Orders:        extra.TakeToken0Orders,
		takeToken1Orders:        extra.TakeToken1Orders,
		takeToken0OrdersMapping: takeToken0OrdersMapping,
		takeToken1OrdersMapping: takeToken1OrdersMapping,
		reactorAddress:          staticExtra.ReactorAddress,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut

	swapSide := p.getSwapSide(tokenAmountIn.Token)
	if swapSide == SwapSideUnknown {
		return nil, ErrTokenInNotSupported
	}

	orders := p.getOrdersBySwapSide(swapSide)
	if len(orders) == 0 {
		return nil, ErrNoOrderAvailable
	}

	totalAmountOut := number.Set(number.Zero)
	remainingAmountIn := number.SetFromBig(tokenAmountIn.Amount)

	swapInfo := SwapInfo{
		AmountIn:     tokenAmountIn.Amount.String(),
		SwapSide:     swapSide,
		FilledOrders: []*DutchOrder{},
	}

}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {

}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return p.reactorAddress
}

func (p *PoolSimulator) CalculateLimit() map[string]*big.Int {

}

func (p *PoolSimulator) getSwapSide(tokenIn string) SwapSide {
	if strings.EqualFold(tokenIn, p.token0) {
		return SwapSideTakeToken0
	}

	if strings.EqualFold(tokenIn, p.token1) {
		return SwapSideTakeToken1
	}

	return SwapSideUnknown
}

func (p *PoolSimulator) getOrdersBySwapSide(swapSide SwapSide) []*Order {
	if swapSide == SwapSideTakeToken0 {
		return p.takeToken0Orders
	}

	return p.takeToken1Orders
}
