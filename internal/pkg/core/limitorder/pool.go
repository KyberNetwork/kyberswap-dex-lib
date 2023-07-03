package limitorder

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type (
	Pool struct {
		pool.Pool
		tokens        []*entity.PoolToken
		ordersMapping map[int64]*valueobject.Order
		// extra fields
		sellOrderIDs []int64
		buyOrderIDs  []int64
	}
)

func NewPool(entityPool entity.Pool) (*Pool, error) {
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

	var extra Extra
	err := json.Unmarshal([]byte(entityPool.Extra), &extra)
	if err != nil {
		return nil, err
	}
	ordersMapping := make(map[int64]*valueobject.Order, (len(extra.BuyOrders) + len(extra.SellOrders)))
	sellOrderIDs, buyOrderIDs := make([]int64, len(extra.SellOrders)), make([]int64, len(extra.BuyOrders))
	for i, buyOrder := range extra.BuyOrders {
		ordersMapping[buyOrder.ID] = buyOrder
		buyOrderIDs[i] = buyOrder.ID
	}
	for j, sellOrder := range extra.SellOrders {
		ordersMapping[sellOrder.ID] = sellOrder
		sellOrderIDs[j] = sellOrder.ID
	}
	return &Pool{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				SwapFee:    constant.Zero,
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reserves,
				Checked:    false,
			},
		},
		sellOrderIDs:  sellOrderIDs,
		buyOrderIDs:   buyOrderIDs,
		ordersMapping: ordersMapping,
		tokens:        entity.ClonePoolTokens(entityPool.Tokens),
	}, nil
}

func (p *Pool) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	return p.calcAmountOut(tokenAmountIn, tokenOut)
}

func (p *Pool) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.Warn("failed to UpdateBalance for LO pool, wrong swapInfo type")
		return
	}

	if len(swapInfo.SwapSide) == 0 {
		return
	}
	for _, filledOrderInfo := range swapInfo.FilledOrders {
		order := p.ordersMapping[filledOrderInfo.OrderID]

		filledTakingAmount, _ := new(big.Int).SetString(filledOrderInfo.FilledTakingAmount, 10)
		filledMakingAmount, _ := new(big.Int).SetString(filledOrderInfo.FilledMakingAmount, 10)

		order.FilledTakingAmount = new(big.Int).Add(order.FilledTakingAmount, filledTakingAmount)
		order.FilledMakingAmount = new(big.Int).Add(order.FilledMakingAmount, filledMakingAmount)
	}
}

func (p *Pool) calcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	swapSide := p.getSwapSide(tokenAmountIn.Token, tokenOut)
	amountOut, swapInfo, feeAmount, err := p.calcAmountWithSwapInfo(swapSide, tokenAmountIn)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: feeAmount,
		},
		Gas:      p.estimateGas(len(swapInfo.FilledOrders)),
		SwapInfo: swapInfo,
	}, nil
}

func (p *Pool) calcAmountWithSwapInfo(swapSide SwapSide, tokenAmountIn pool.TokenAmount) (*big.Int, SwapInfo, *big.Int, error) {

	orderIDs := p.getOrderIDsBySwapSide(swapSide)
	if len(orderIDs) == 0 {
		return big.NewInt(0), SwapInfo{}, nil, nil
	}

	totalAmountOutWei := constant.Zero
	totalAmountIn := tokenAmountIn.Amount

	swapInfo := SwapInfo{
		FilledOrders: []*FilledOrderInfo{},
		SwapSide:     swapSide,
		AmountIn:     tokenAmountIn.Amount.String(),
	}
	isFulfillAmountIn := false
	totalFeeAmountWei := new(big.Int)

	totalMakingAmountWei := new(big.Int)
	for i, orderID := range orderIDs {
		order, ok := p.ordersMapping[orderID]
		if !ok {
			return nil, swapInfo, nil, fmt.Errorf("order %d is not existed in pool", orderID)
		}
		// rate should be the result of making amount/taking amount when dividing decimals per token.
		// However, we can also use rate with making amount/taking amount (wei) to calculate the amount out instead of converting to measure per token. Because we will return amount out(wei) (we have to multip amountOut(taken out) with decimals)
		rate := new(big.Float).Quo(new(big.Float).SetInt(order.MakingAmount), new(big.Float).SetInt(order.TakingAmount))
		remainingMakingAmountWei := new(big.Int).Sub(order.MakingAmount, order.FilledMakingAmount)
		remainingTakingAmountWei := new(big.Int).Sub(order.TakingAmount, order.FilledTakingAmount)
		totalMakingAmountWei = new(big.Int).Add(totalMakingAmountWei, remainingMakingAmountWei)
		// Order was filled out.
		if remainingMakingAmountWei.Cmp(constant.Zero) <= 0 {
			continue
		}
		if remainingTakingAmountWei.Cmp(totalAmountIn) >= 0 {
			amountOutWei := new(big.Float).Mul(new(big.Float).SetInt(totalAmountIn), rate)
			filledTakingAmountWei := totalAmountIn
			filledMakingAmountWei, _ := amountOutWei.Int(nil)
			feeAmountWeiByOrder := p.calcFeeAmountPerOrder(order, filledMakingAmountWei)
			totalFeeAmountWei = new(big.Int).Add(totalFeeAmountWei, feeAmountWeiByOrder)
			actualAmountOut := new(big.Int).Sub(filledMakingAmountWei, feeAmountWeiByOrder)
			totalAmountOutWei = new(big.Int).Add(totalAmountOutWei, actualAmountOut)
			filledOrderInfo := newFilledOrderInfo(order, filledTakingAmountWei.String(), filledMakingAmountWei.String(), feeAmountWeiByOrder.String())
			swapInfo.FilledOrders = append(swapInfo.FilledOrders, filledOrderInfo)
			isFulfillAmountIn = true

			// Currently, when Aggregator finds route and returns some orders and sends them to the smart contract to execute.
			// We will often meet edge cases that these orders can be fulfilled by a trading bot or taker on Aggregator.
			// From that, the estimated amount out and filled orders are not correct. So we need to add more orders when sending to SC to the executor.
			// In this case, we will some orders util total MakingAmount(remainMakingAmount)/estimated amountOut >= 1.3 (130%)
			totalAmountOutWeiBigFloat := new(big.Float).SetInt64(totalAmountOutWei.Int64())
			for j := i + 1; j < len(orderIDs); j++ {
				if new(big.Float).SetInt(totalMakingAmountWei).Cmp(new(big.Float).Mul(totalAmountOutWeiBigFloat, FallbackPercentageOfTotalMakingAmount)) >= 0 {
					break
				}
				order, ok := p.ordersMapping[orderIDs[j]]
				if !ok {
					continue
				}
				remainingMakingAmountWei := new(big.Int).Sub(order.MakingAmount, order.FilledMakingAmount)
				if remainingMakingAmountWei.Cmp(constant.Zero) == 0 {
					continue
				}
				totalMakingAmountWei = new(big.Int).Add(totalMakingAmountWei, remainingMakingAmountWei)
				filledOrderInfo := newFilledOrderInfo(order, "0", "0", "0")
				filledOrderInfo.IsFallBack = true
				swapInfo.FilledOrders = append(swapInfo.FilledOrders, filledOrderInfo)
			}
			break
		}
		totalAmountIn = new(big.Int).Sub(totalAmountIn, remainingTakingAmountWei)
		feeAmountWeiByOrder := p.calcFeeAmountPerOrder(order, remainingMakingAmountWei)
		actualAmountOut := new(big.Int).Sub(remainingMakingAmountWei, feeAmountWeiByOrder)
		totalAmountOutWei = new(big.Int).Add(totalAmountOutWei, actualAmountOut)
		totalFeeAmountWei = new(big.Int).Add(totalFeeAmountWei, feeAmountWeiByOrder)
		filledOrderInfo := newFilledOrderInfo(order, remainingTakingAmountWei.String(), remainingMakingAmountWei.String(), feeAmountWeiByOrder.String())
		swapInfo.FilledOrders = append(swapInfo.FilledOrders, filledOrderInfo)
	}
	if !isFulfillAmountIn {
		return nil, SwapInfo{}, nil, ErrCannotFulfillAmountIn
	}
	return totalAmountOutWei, swapInfo, totalFeeAmountWei, nil
}

// feeAmount = (params.makingAmount * params.order.makerTokenFeePercent + BPS - 1) / BPS
func (p *Pool) calcFeeAmountPerOrder(order *valueobject.Order, filledMakingAmount *big.Int) *big.Int {
	if order.MakerTokenFeePercent == 0 {
		return constant.Zero
	}
	amount := new(big.Int).Mul(filledMakingAmount, big.NewInt(int64(order.MakerTokenFeePercent)))
	return new(big.Int).Div(new(big.Int).Sub(new(big.Int).Add(amount, valueobject.BasisPoint), constant.One), valueobject.BasisPoint)
}

func (p *Pool) estimateGas(numberOfFilledOrders int) int64 {
	return p.estimateGasForExecutor(numberOfFilledOrders) + p.estimateGasForRouter(numberOfFilledOrders)
}

func (p *Pool) estimateGasForExecutor(numberOfFilledOrders int) int64 {
	return int64(BaseGas) + int64(numberOfFilledOrders)*int64(GasPerOrderExecutor)
}

func (p *Pool) estimateGasForRouter(numberOfFilledOrders int) int64 {
	return int64(numberOfFilledOrders) * int64(GasPerOrderRouter)

}

func (p *Pool) extractAssetTokenDecimals(order *valueobject.Order) (uint8, uint8) {
	if strings.EqualFold(order.MakerAsset, p.tokens[0].Address) {
		return p.tokens[0].Decimals, p.tokens[1].Decimals
	}
	return p.tokens[1].Decimals, p.tokens[0].Decimals
}

func (p *Pool) getOrderIDsBySwapSide(swapSide SwapSide) []int64 {
	if swapSide == Buy {
		return p.buyOrderIDs
	}
	return p.sellOrderIDs
}

func (p *Pool) getSwapSide(tokenIn string, TokenOut string) SwapSide {
	if strings.ToLower(tokenIn) > strings.ToLower(TokenOut) {
		return Sell
	}
	return Buy
}

func (p *Pool) GetLpToken() string {
	return ""
}

func (p *Pool) CanSwapTo(address string) []string {
	swappableTokens := make([]string, 0, len(p.GetTokens())-1)

	isTokenExists := false
	for _, token := range p.GetTokens() {
		if strings.EqualFold(token, address) {
			isTokenExists = true
			break
		}
	}

	if !isTokenExists {
		return swappableTokens
	}

	for _, token := range p.GetTokens() {
		if token == address {
			continue
		}

		swappableTokens = append(swappableTokens, token)
	}

	return swappableTokens
}

func (p *Pool) GetMidPrice(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	return p.getBestRate(tokenIn, tokenOut)
}

func (p *Pool) CalcExactQuote(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	return p.getBestRate(tokenIn, tokenOut)
}

func (p *Pool) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}

func (p *Pool) getBestRate(tokenIn string, tokenOut string) *big.Int {
	swapSide := p.getSwapSide(tokenIn, tokenOut)
	orderIDs := p.getOrderIDsBySwapSide(swapSide)
	if len(orderIDs) == 0 {
		return constant.Zero
	}
	order, ok := p.ordersMapping[orderIDs[0]]
	if !ok {
		return constant.Zero
	}
	makerAssetDecimal, takerAssetDecimal := p.extractAssetTokenDecimals(order)
	takerAssetTenPowDecimals := constant.TenPowDecimals(takerAssetDecimal)
	makerAssetTenPowDecimals := constant.TenPowDecimals(makerAssetDecimal)
	takingAmount := new(big.Float).Quo(
		new(big.Float).SetInt(order.TakingAmount), takerAssetTenPowDecimals)
	makingAmount := new(big.Float).Quo(
		new(big.Float).SetInt(order.MakingAmount), makerAssetTenPowDecimals)
	rate := new(big.Float).Quo(makingAmount, takingAmount)
	rateInt64, _ := rate.Int64()
	return new(big.Int).SetInt64(rateInt64)
}
