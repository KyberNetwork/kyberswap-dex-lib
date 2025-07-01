package uniswaplo

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/logger"
)

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

type PoolMetaInfo struct {
	ReactorAddress  string `json:"reactorAddress"`
	ApprovalAddress string `json:"approvalAddress"`
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
				Address:  strings.ToLower(entityPool.Address),
				SwapFee:  integer.Zero(),
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   tokens,
				Reserves: reserves,
				// Checked:    false,
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

	filledSwappers := make(map[common.Address]struct{})

	// Filling logic, note that this LO only supports full fill
	// Using greedy algo for simple way approach first,
	// but we also could use dynamic programming like knapsack algo
	for _, order := range orders {
		// skip zero order
		if order.Input.StartAmount.Cmp(number.Zero) == 0 {
			continue
		}

		orderTakingAmount := order.Outputs[0].StartAmount
		// Case 1: This order can not be enough to fill
		// orderAmount > remainingAmountIn, continue
		if orderTakingAmount.Cmp(remainingAmountIn) > 0 {
			continue
		}

		// skip filled swappers, we only support take only 1 swapper per batch
		// using same swapper for multiple orders is not supported, this way will highly having chances led to insufficent permit2 allowance
		if _, ok := filledSwappers[order.Swapper]; ok {
			continue
		}

		// Case 2: Fullfill this order
		// orderAmount == remainingAmountIn
		// add user making amount to totalAmountOut
		if orderTakingAmount.Cmp(remainingAmountIn) == 0 {
			totalAmountOut.Add(totalAmountOut, order.Input.StartAmount)
			swapInfo.FilledOrders = append(swapInfo.FilledOrders, order)
			remainingAmountIn.Sub(remainingAmountIn, orderTakingAmount)

			swapInfo.IsAmountInFulfilled = true
			filledSwappers[order.Swapper] = struct{}{}
			continue
		}

		// Case 3: order amount < remainingAmountIn
		// add the order to totalAmountOut
		// and update remainingAmountIn
		// remainingAmountIn = remainingAmountIn - orderTakingAmount
		// totalAmountOut = totalAmountOut + orderMakingAmount
		if orderTakingAmount.Cmp(remainingAmountIn) < 0 {
			remainingAmountIn.Sub(remainingAmountIn, orderTakingAmount)

			totalAmountOut.Add(totalAmountOut, order.Input.StartAmount)
			swapInfo.FilledOrders = append(swapInfo.FilledOrders, order)
			filledSwappers[order.Swapper] = struct{}{}
			continue
		}
	}

	if len(swapInfo.FilledOrders) == 0 {
		return nil, ErrCannotFulfillAmountIn
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: totalAmountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: integer.Zero(),
		},
		Gas:      p.estimateGas(len(swapInfo.FilledOrders)),
		SwapInfo: swapInfo,
		RemainingTokenAmountIn: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: remainingAmountIn.ToBig(),
		},
	}, nil
}

func (p *PoolSimulator) estimateGas(numberOfFilledOrders int) int64 {
	return p.estimateGasForExecutor(numberOfFilledOrders)
}

func (p *PoolSimulator) estimateGasForExecutor(numberOfFilledOrders int) int64 {
	return int64(BaseGas) + int64(numberOfFilledOrders)*int64(GasPerOrderExecutor)
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.Warn("failed to UpdateBalance for LO1inch pool, wrong swapInfo type")
		return
	}

	if swapInfo.SwapSide == SwapSideUnknown {
		return
	}

	for _, filledOrderInfo := range swapInfo.FilledOrders {
		var order *DutchOrder

		if swapInfo.SwapSide == SwapSideTakeToken0 {
			orderIndex, ok := p.takeToken0OrdersMapping[filledOrderInfo.OrderHash]
			if !ok {
				continue
			}

			order = p.takeToken0Orders[orderIndex]
		} else {
			orderIndex, ok := p.takeToken1OrdersMapping[filledOrderInfo.OrderHash]
			if !ok {
				continue
			}

			order = p.takeToken1Orders[orderIndex]
		}

		// update filled order
		order.Input.StartAmount = number.Zero
		order.Outputs[0].StartAmount = number.Zero
	}
}

func (p *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) interface{} {
	return PoolMetaInfo{
		ApprovalAddress: p.GetApprovalAddress(tokenIn, tokenOut),
		// ReactorAddress for backward compatibility
		ReactorAddress: p.reactorAddress,
	}
}

func (p *PoolSimulator) GetApprovalAddress(tokenIn, _ string) string {
	return p.reactorAddress
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

func (p *PoolSimulator) getOrdersBySwapSide(swapSide SwapSide) []*DutchOrder {
	if swapSide == SwapSideTakeToken0 {
		return p.takeToken0Orders
	}

	return p.takeToken1Orders
}
