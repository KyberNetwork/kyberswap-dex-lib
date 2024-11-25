package lo1inch

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type (
	PoolSimulator struct {
		pool.Pool

		// static extra fields
		token0 string
		token1 string

		// extra fields
		takeToken0Orders []*Order
		takeToken1Orders []*Order

		takeToken0OrdersMapping map[string]int
		takeToken1OrdersMapping map[string]int

		// store min(balance, allowance) for all unique pairs of maker:makerAsset in this pool
		// will be aggregated up by router-service to be a global value for all maker:makerAsset in LO
		minBalanceAllowanceByMakerAndAsset map[makerAndAsset]*big.Int
	}
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var numTokens = len(entityPool.Tokens)
	var tokens = make([]string, numTokens)
	var reserves = make([]*big.Int, numTokens)
	if numTokens != 2 {
		return nil, fmt.Errorf("pool's number of tokens should equal 2")
	}
	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = utils.NewBig10(entityPool.Reserves[i])
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

	numOrders := len(extra.TakeToken0Orders) + len(extra.TakeToken1Orders)
	minBalanceAllowanceByMakerAndAsset := make(map[makerAndAsset]*big.Int, numOrders)

	for i, takeToken0Order := range extra.TakeToken0Orders {
		takeToken0OrdersMapping[takeToken0Order.OrderHash] = i

		// get min(balance, allowance) for this maker:makerAsset pair
		minBalanceAllowanceByMakerAndAsset[newMakerAndAsset(takeToken0Order.Maker, takeToken0Order.MakerAsset)] = utils.Min(takeToken0Order.MakerBalance, takeToken0Order.MakerAllowance)
	}

	for i, takeToken1Order := range extra.TakeToken1Orders {
		takeToken1OrdersMapping[takeToken1Order.OrderHash] = i

		// get min(balance, allowance) for this maker:makerAsset pair
		minBalanceAllowanceByMakerAndAsset[newMakerAndAsset(takeToken1Order.Maker, takeToken1Order.MakerAsset)] = utils.Min(takeToken1Order.MakerBalance, takeToken1Order.MakerAllowance)
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
		token0:                             staticExtra.Token0,
		token1:                             staticExtra.Token1,
		takeToken0Orders:                   extra.TakeToken0Orders,
		takeToken1Orders:                   extra.TakeToken1Orders,
		takeToken0OrdersMapping:            takeToken0OrdersMapping,
		takeToken1OrdersMapping:            takeToken1OrdersMapping,
		minBalanceAllowanceByMakerAndAsset: minBalanceAllowanceByMakerAndAsset,
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

	totalAmountOut := integer.Zero()
	remainingAmountIn := tokenAmountIn.Amount

	swapInfo := SwapInfo{
		AmountIn:     tokenAmountIn.Amount.String(),
		SwapSide:     swapSide,
		FilledOrders: []*FilledOrderInfo{},
	}
	isAmountInFulfilled := false

	// we need to update maker's remaining balance in 2 places:
	// - in UpdateBalance: mainly to deal with case where maker has orders with same makerAsset but different takerAsset
	// - when simulating filling each order here: we cannot do the same as in kyber-pmm (simulating first then check inventory limit at the end)
	// because in LO we have multiple makers, and also because we still need to allow orders that have part of the balance available
	// the problem is that in this func we cannot update the limit,
	// so we'll use this map to track filled amount for each maker, then subtract from the original balance, to have the remaining balance available
	filledMakingAmountByMaker := make(map[string]*big.Int, len(p.minBalanceAllowanceByMakerAndAsset))

	totalMakingAmount := new(big.Int)

	for i, order := range orders {
		orderRemainingMakingAmount := order.RemainingMakerAmount

		// the actual available balance might be less than `order.RemainingMakerAmount`
		// for example: in this pool, we have used another order for the same maker and makerAsset:takerAsset pair in the previous loop
		if makerRemainingBalance := getMakerRemainingBalance(
			param.Limit,
			filledMakingAmountByMaker,
			order.Maker,
			order.MakerAsset,
		); makerRemainingBalance != nil && orderRemainingMakingAmount.Cmp(makerRemainingBalance) > 0 {
			orderRemainingMakingAmount = makerRemainingBalance
		}

		// Order was filled, just skip it
		if orderRemainingMakingAmount.Sign() <= 0 {
			continue
		}

		// calculate order's remaining taking amount
		// orderRemainingTakingAmount = order.TakingAmount * orderRemainingMakingAmount / order.MakingAmount
		orderRemainingTakingAmount := new(big.Int).Set(order.TakingAmount)
		orderRemainingTakingAmount.Mul(orderRemainingTakingAmount, orderRemainingMakingAmount)
		orderRemainingTakingAmount.Div(orderRemainingTakingAmount, order.MakingAmount)

		totalMakingAmount.Add(totalMakingAmount, orderRemainingMakingAmount)

		// Case 1: This order can fulfill the remaining amount in
		if orderRemainingTakingAmount.Cmp(remainingAmountIn) >= 0 {
			takerRateBF := new(big.Float).Quo(new(big.Float).SetInt(order.MakingAmount), new(big.Float).SetInt(order.TakingAmount))
			orderAmountOutBF := new(big.Float).Mul(
				new(big.Float).SetInt(remainingAmountIn),
				takerRateBF,
			)
			orderAmountOut, _ := orderAmountOutBF.Int(nil)

			// order too small
			if orderAmountOut.Sign() <= 0 {
				continue
			}

			totalAmountOut.Add(totalAmountOut, orderAmountOut)

			orderFilledMakingAmount := orderAmountOut
			orderFilledTakingAmount := remainingAmountIn
			filledOrderInfo := newFilledOrderInfo(
				order,
				orderFilledMakingAmount,
				orderFilledTakingAmount,
			)
			swapInfo.FilledOrders = append(swapInfo.FilledOrders, filledOrderInfo)

			isAmountInFulfilled = true

			// orderAmountOut is the filled making amount for this order because this order is partially filled
			addFilledMakingAmount(filledMakingAmountByMaker, order.Maker, orderAmountOut)

			// Currently, the aggregator finds route, returns some orders and sends them to the smart contract to execute.
			// We often meet edge cases that these orders can be fulfilled by a trading bot or another taker on the aggregator beforehand.
			// From that, the estimated amount out and filled orders are not correct. So we need to add more "backup" orders when sending to SC to the executor.
			// In this case, we will send some orders util total MakingAmount(remainMakingAmount)/estimated amountOut >= 1.3 (130%)
			totalAmountOutBF := new(big.Float).SetInt(totalAmountOut)
			for j := i + 1; j < len(orders); j++ {
				if new(big.Float).SetInt(totalMakingAmount).Cmp(new(big.Float).Mul(totalAmountOutBF, FallbackPercentageOfTotalMakingAmount)) >= 0 {
					break
				}

				order := orders[j]

				orderRemainingMakingAmount := new(big.Int).Set(order.RemainingMakerAmount)
				if makerRemainingBalance := getMakerRemainingBalance(
					param.Limit,
					filledMakingAmountByMaker,
					order.Maker,
					order.MakerAsset,
				); makerRemainingBalance != nil && orderRemainingMakingAmount.Cmp(makerRemainingBalance) > 0 {
					orderRemainingMakingAmount = makerRemainingBalance
				}

				if orderRemainingMakingAmount.Sign() <= 0 {
					continue
				}

				totalMakingAmount.Add(totalMakingAmount, orderRemainingMakingAmount)
				filledOrderInfo := newFilledOrderInfo(
					order,
					utils.ZeroBI,
					utils.ZeroBI,
				)
				filledOrderInfo.IsBackup = true
				swapInfo.FilledOrders = append(swapInfo.FilledOrders, filledOrderInfo)
			}

			break
		}

		// Case 2: This order can't fulfill the remaining amount in
		remainingAmountIn = new(big.Int).Sub(remainingAmountIn, orderRemainingTakingAmount)
		orderFilledMakingAmount := orderRemainingMakingAmount // because this order is fully filled
		orderFilledTakingAmount := orderRemainingTakingAmount // because this order is fully filled
		totalAmountOut = new(big.Int).Add(totalAmountOut, orderFilledMakingAmount)
		filledOrderInfo := newFilledOrderInfo(
			order,
			orderFilledMakingAmount,
			orderFilledTakingAmount,
		)
		swapInfo.FilledOrders = append(swapInfo.FilledOrders, filledOrderInfo)

		addFilledMakingAmount(filledMakingAmountByMaker, order.Maker, orderFilledMakingAmount)
	}

	if !isAmountInFulfilled {
		return nil, ErrCannotFulfillAmountIn
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: totalAmountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: integer.Zero(),
		}, // no fee for 1inch LO
		Gas:      p.estimateGas(len(swapInfo.FilledOrders)),
		SwapInfo: swapInfo,
	}, nil
}

func newFilledOrderInfo(
	order *Order,
	orderFilledMakingAmount *big.Int,
	orderFilledTakingAmount *big.Int,
) *FilledOrderInfo {
	return &FilledOrderInfo{
		Signature:            order.Signature,
		OrderHash:            order.OrderHash,
		CreateDateTime:       order.CreateDateTime,
		RemainingMakerAmount: new(big.Int).Sub(order.RemainingMakerAmount, orderFilledMakingAmount),
		MakerBalance:         new(big.Int).Sub(order.MakerBalance, orderFilledMakingAmount),
		MakerAllowance:       new(big.Int).Sub(order.MakerAllowance, orderFilledMakingAmount),
		MakerAsset:           order.MakerAsset,
		TakerAsset:           order.TakerAsset,
		Salt:                 order.Salt,
		Receiver:             order.Receiver,
		MakingAmount:         order.MakingAmount,
		TakingAmount:         order.TakingAmount,
		Maker:                order.Maker,
		MakerRate:            order.MakerRate,
		TakerRate:            order.TakerRate,

		// REMEMBER: these 2 values are the amounts of maker/taker asset that has been filled, but this is just the amount that has been filled after ONE CalcAmountOut call, not the total amount that has been filled in this order
		// (check their definition in the struct)
		FilledMakingAmount: orderFilledMakingAmount,
		FilledTakingAmount: orderFilledTakingAmount,
	}
}

func addFilledMakingAmount(
	filledMakingAmountByMaker map[string]*big.Int,
	maker string,
	filledMakingAmount *big.Int,
) {
	if totalFilled, ok := filledMakingAmountByMaker[maker]; ok {
		filledMakingAmountByMaker[maker] = new(big.Int).Add(totalFilled, filledMakingAmount)
	} else {
		filledMakingAmountByMaker[maker] = new(big.Int).Set(filledMakingAmount)
	}
}

func getMakerRemainingBalance(
	limit pool.SwapLimit,
	filledMakingAmountByMaker map[string]*big.Int,
	maker, makerAsset string,
) *big.Int {
	if limit == nil {
		// can happen if this change get deployed to router-service before pool-service, just ignore
		return nil
	}

	makerBalanceAllowance := limit.GetLimit(newMakerAndAsset(maker, makerAsset))
	if makerBalanceAllowance == nil {
		// should not happen, but anw just return 0 as if this maker has no balance left
		// Do not return bignumber.ZeroBI because if the `makerRemainingBalance` is updated somewhere else,
		// it will also alter the original value of bignumber.ZeroBI
		return integer.Zero()
	}

	if totalFilled := filledMakingAmountByMaker[maker]; totalFilled != nil {
		return new(big.Int).Sub(makerBalanceAllowance, totalFilled)
	} else {
		return makerBalanceAllowance
	}
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
		var order *Order

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

		order.RemainingMakerAmount = filledOrderInfo.RemainingMakerAmount
		order.MakerBalance = filledOrderInfo.MakerBalance
		order.MakerAllowance = filledOrderInfo.MakerAllowance

		if params.SwapLimit != nil {
			_, _, _ = params.SwapLimit.UpdateLimit(
				newMakerAndAsset(order.Maker, order.MakerAsset),
				newMakerAndAsset(order.Maker, order.TakerAsset),
				filledOrderInfo.FilledMakingAmount,
				filledOrderInfo.FilledTakingAmount,
			)
		}
	}
}

func (p *PoolSimulator) estimateGas(numberOfFilledOrders int) int64 {
	return p.estimateGasForExecutor(numberOfFilledOrders) + p.estimateGasForRouter(numberOfFilledOrders)
}

func (p *PoolSimulator) estimateGasForExecutor(numberOfFilledOrders int) int64 {
	return int64(BaseGas) + int64(numberOfFilledOrders)*int64(GasPerOrderExecutor)
}

func (p *PoolSimulator) estimateGasForRouter(numberOfFilledOrders int) int64 {
	return int64(numberOfFilledOrders) * int64(GasPerOrderRouter)

}

func (p *PoolSimulator) getOrdersBySwapSide(swapSide SwapSide) []*Order {
	if swapSide == SwapSideTakeToken0 {
		return p.takeToken0Orders
	}

	return p.takeToken1Orders
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

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}

// Inventory Limit

type makerAndAsset = string

func newMakerAndAsset(maker, makerAsset string) makerAndAsset {
	return fmt.Sprintf("%v:%v", maker, makerAsset)
}

func (p *PoolSimulator) CalculateLimit() map[string]*big.Int {
	count := len(p.minBalanceAllowanceByMakerAndAsset)
	if count == 0 {
		return nil
	}

	res := make(map[string]*big.Int, count)
	for k, v := range p.minBalanceAllowanceByMakerAndAsset {
		res[k] = new(big.Int).Set(v)
	}

	return res
}
