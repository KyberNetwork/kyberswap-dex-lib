package limitorder

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	constant "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type (
	PoolSimulator struct {
		pool.Pool
		tokens        []*entity.PoolToken
		ordersMapping map[int64]*order
		// extra fields
		sellOrderIDs []int64
		buyOrderIDs  []int64

		contractAddress string
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

	var contractAddress string
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		// this is optional for now, will changed to required later
		contractAddress = ""
	} else {
		contractAddress = staticExtra.ContractAddress
	}

	var extra Extra
	err := json.Unmarshal([]byte(entityPool.Extra), &extra)
	if err != nil {
		return nil, err
	}
	ordersMapping := make(map[int64]*order, (len(extra.BuyOrders) + len(extra.SellOrders)))
	sellOrderIDs, buyOrderIDs := make([]int64, len(extra.SellOrders)), make([]int64, len(extra.BuyOrders))
	for i, buyOrder := range extra.BuyOrders {
		ordersMapping[buyOrder.ID] = buyOrder
		buyOrderIDs[i] = buyOrder.ID
	}
	for j, sellOrder := range extra.SellOrders {
		ordersMapping[sellOrder.ID] = sellOrder
		sellOrderIDs[j] = sellOrder.ID
	}
	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				SwapFee:    constant.ZeroBI,
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

		contractAddress: contractAddress,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	return p.calcAmountOut(tokenAmountIn, tokenOut)
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
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

func (p *PoolSimulator) calcAmountOut(
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

func (p *PoolSimulator) calcAmountWithSwapInfo(swapSide SwapSide, tokenAmountIn pool.TokenAmount) (*big.Int, SwapInfo, *big.Int, error) {

	orderIDs := p.getOrderIDsBySwapSide(swapSide)
	if len(orderIDs) == 0 {
		return big.NewInt(0), SwapInfo{}, nil, nil
	}

	totalAmountOutWei := constant.ZeroBI
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
		if remainingMakingAmountWei.Cmp(constant.ZeroBI) <= 0 {
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
				if remainingMakingAmountWei.Cmp(constant.ZeroBI) == 0 {
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
func (p *PoolSimulator) calcFeeAmountPerOrder(order *order, filledMakingAmount *big.Int) *big.Int {
	if order.MakerTokenFeePercent == 0 {
		return constant.ZeroBI
	}
	amount := new(big.Int).Mul(filledMakingAmount, big.NewInt(int64(order.MakerTokenFeePercent)))
	return new(big.Int).Div(new(big.Int).Sub(new(big.Int).Add(amount, valueobject.BasisPoint), constant.One), valueobject.BasisPoint)
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

func (p *PoolSimulator) getOrderIDsBySwapSide(swapSide SwapSide) []int64 {
	if swapSide == Buy {
		return p.buyOrderIDs
	}
	return p.sellOrderIDs
}

func (p *PoolSimulator) getSwapSide(tokenIn string, TokenOut string) SwapSide {
	if strings.ToLower(tokenIn) > strings.ToLower(TokenOut) {
		return Sell
	}
	return Buy
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return p.contractAddress
}

func newFilledOrderInfo(order *order, filledTakingAmount, filledMakingAmount string, feeAmount string) *FilledOrderInfo {
	feeConfig := ""
	if order.FeeConfig != nil {
		feeConfig = order.FeeConfig.String()
	}
	return &FilledOrderInfo{
		OrderID:              order.ID,
		FilledTakingAmount:   filledTakingAmount,
		FilledMakingAmount:   filledMakingAmount,
		TakingAmount:         order.TakingAmount.String(),
		MakingAmount:         order.MakingAmount.String(),
		Salt:                 order.Salt,
		MakerAsset:           order.MakerAsset,
		TakerAsset:           order.TakerAsset,
		Maker:                order.Maker,
		Receiver:             order.Receiver,
		AllowedSenders:       order.AllowedSenders,
		GetMakerAmount:       order.GetMakerAmount,
		GetTakerAmount:       order.GetTakerAmount,
		FeeConfig:            feeConfig,
		FeeRecipient:         order.FeeRecipient,
		MakerAssetData:       order.MakerAssetData,
		MakerTokenFeePercent: order.MakerTokenFeePercent,
		TakerAssetData:       order.TakerAssetData,
		Predicate:            order.Predicate,
		Permit:               order.Permit,
		Interaction:          order.Interaction,
		Signature:            order.Signature,
		FeeAmount:            feeAmount,
	}
}
