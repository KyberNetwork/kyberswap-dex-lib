package limitorder

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	constant "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type (
	PoolSimulator struct {
		pool.Pool
		ordersMapping map[int64]*order
		// extra fields
		sellOrderIDs []int64
		buyOrderIDs  []int64

		contractAddress string

		// store min(balance, allowance) for all unique pair of maker:makerAsset in this pool
		// will be aggregated up by router-service to be a global value for all maker:makerAsset in LO
		allMakersBalanceAllowance map[makerAndAsset]*big.Int
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
	numOrders := len(extra.BuyOrders) + len(extra.SellOrders)
	allMakersBalanceAllowance := make(map[makerAndAsset]*big.Int, numOrders)
	ordersMapping := make(map[int64]*order, numOrders)
	sellOrderIDs, buyOrderIDs := make([]int64, len(extra.SellOrders)), make([]int64, len(extra.BuyOrders))
	for i, buyOrder := range extra.BuyOrders {
		ordersMapping[buyOrder.ID] = buyOrder
		buyOrderIDs[i] = buyOrder.ID
		if buyOrder.MakerBalanceAllowance == nil {
			// old orders don't have this field
			continue
		}
		allMakersBalanceAllowance[NewMakerAndAsset(buyOrder.Maker, buyOrder.MakerAsset)] = buyOrder.MakerBalanceAllowance
	}
	for j, sellOrder := range extra.SellOrders {
		ordersMapping[sellOrder.ID] = sellOrder
		sellOrderIDs[j] = sellOrder.ID
		if sellOrder.MakerBalanceAllowance == nil {
			// old orders don't have this field
			continue
		}
		allMakersBalanceAllowance[NewMakerAndAsset(sellOrder.Maker, sellOrder.MakerAsset)] = sellOrder.MakerBalanceAllowance
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

		contractAddress: contractAddress,

		allMakersBalanceAllowance: allMakersBalanceAllowance,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	return p.calcAmountOut(param.TokenAmountIn, param.TokenOut, param.Limit)
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

		if order.AvailableMakingAmount != nil {
			order.AvailableMakingAmount = new(big.Int).Sub(order.AvailableMakingAmount, filledMakingAmount)
		}
		if params.SwapLimit != nil {
			_, _, _ = params.SwapLimit.UpdateLimit(
				NewMakerAndAsset(order.Maker, order.MakerAsset),
				NewMakerAndAsset(order.Maker, order.TakerAsset),
				filledMakingAmount,
				filledTakingAmount,
			)
		}
	}
}

func (p *PoolSimulator) calcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
	limit pool.SwapLimit,
) (*pool.CalcAmountOutResult, error) {
	swapSide := p.getSwapSide(tokenAmountIn.Token, tokenOut)
	amountOut, swapInfo, feeAmount, err := p.calcAmountWithSwapInfo(swapSide, tokenAmountIn, limit)
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

	makerBalanceAllowance := limit.GetLimit(NewMakerAndAsset(maker, makerAsset))
	if makerBalanceAllowance == nil {
		// should not happen, but anw just return 0 as if this maker has no balance left
		return big.NewInt(0)
	}

	if totalFilled := filledMakingAmountByMaker[maker]; totalFilled != nil {
		return new(big.Int).Sub(makerBalanceAllowance, totalFilled)
	} else {
		return makerBalanceAllowance
	}
}

func (p *PoolSimulator) calcAmountWithSwapInfo(swapSide SwapSide, tokenAmountIn pool.TokenAmount, limit pool.SwapLimit) (*big.Int, SwapInfo, *big.Int, error) {

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

	// we need to update maker's remaining balance in 2 places:
	// - in UpdateBalance: mainly to deal with case where maker has orders with same makerAsset but different takerAsset
	// - when simulating filling each order here: we cannot do the same as in kyber-pmm (simulating first then check inventory limit at the end)
	//				because in LO we have multiple makers, and also because we still need to allow orders that have part of the balance available
	//		the problem is that in this func we cannot update the limit,
	//		so we'll use this map to track filled amount for each maker, then subtract from the original balance, to have the remaining balance available
	filledMakingAmountByMaker := make(map[string]*big.Int, len(p.allMakersBalanceAllowance))

	totalMakingAmountWei := new(big.Int)
	for i, orderID := range orderIDs {
		order, ok := p.ordersMapping[orderID]
		if !ok {
			return nil, swapInfo, nil, fmt.Errorf("order %d is not existed in pool", orderID)
		}
		// rate should be the result of making amount/taking amount when dividing decimals per token.
		// However, we can also use rate with making amount/taking amount (wei) to calculate the amount out instead of converting to measure per token. Because we will return amount out(wei) (we have to multip amountOut(taken out) with decimals)
		rate := new(big.Float).Quo(new(big.Float).SetInt(order.MakingAmount), new(big.Float).SetInt(order.TakingAmount))
		var remainingMakingAmountWei, remainingTakingAmountWei *big.Int
		if order.AvailableMakingAmount == nil {
			remainingMakingAmountWei = new(big.Int).Sub(order.MakingAmount, order.FilledMakingAmount)
			remainingTakingAmountWei = new(big.Int).Sub(order.TakingAmount, order.FilledTakingAmount)
		} else {
			remainingMakingAmountWei = order.AvailableMakingAmount
			// the actual available balance might be less than `AvailableMakingAmount`
			// for example if we has used another order for this same maker and makerAsset (but with different takerAsset) before
			if makerRemainingBalance := getMakerRemainingBalance(limit, filledMakingAmountByMaker, order.Maker, order.MakerAsset); makerRemainingBalance != nil && remainingMakingAmountWei.Cmp(makerRemainingBalance) > 0 {
				remainingMakingAmountWei = makerRemainingBalance
			}
			remainingTakingAmountWei = new(big.Int).Div(new(big.Int).Mul(remainingMakingAmountWei, order.TakingAmount), order.MakingAmount)
		}
		totalMakingAmountWei = new(big.Int).Add(totalMakingAmountWei, remainingMakingAmountWei)
		// Order was filled out.
		if remainingMakingAmountWei.Cmp(constant.ZeroBI) <= 0 {
			continue
		}

		totalAmountInAfterFee, _ := p.calcTakerAssetFeeAmountExactIn(order, totalAmountIn)
		// ideally we should return totalFeeAmountWei in takerAsset here
		// but for now it's not used, and we might get mixed up with makerAsset fee, so will ignore for now

		if remainingTakingAmountWei.Cmp(totalAmountInAfterFee) >= 0 {
			amountOutWei := new(big.Float).Mul(new(big.Float).SetInt(totalAmountInAfterFee), rate)
			filledTakingAmountWei := totalAmountInAfterFee
			filledMakingAmountWei, _ := amountOutWei.Int(nil)

			// order too small
			if filledMakingAmountWei.Cmp(constant.ZeroBI) <= 0 {
				continue
			}

			feeAmountWeiByOrder := p.calcMakerAsetFeeAmount(order, filledMakingAmountWei)
			totalFeeAmountWei = new(big.Int).Add(totalFeeAmountWei, feeAmountWeiByOrder)
			actualAmountOut := new(big.Int).Sub(filledMakingAmountWei, feeAmountWeiByOrder)
			totalAmountOutWei = new(big.Int).Add(totalAmountOutWei, actualAmountOut)
			filledOrderInfo := newFilledOrderInfo(order, filledTakingAmountWei.String(), filledMakingAmountWei.String(), feeAmountWeiByOrder.String())
			swapInfo.FilledOrders = append(swapInfo.FilledOrders, filledOrderInfo)
			isFulfillAmountIn = true
			addFilledMakingAmount(filledMakingAmountByMaker, order.Maker, filledMakingAmountWei)

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
				var remainingMakingAmountWei *big.Int
				if order.AvailableMakingAmount == nil {
					remainingMakingAmountWei = new(big.Int).Sub(order.MakingAmount, order.FilledMakingAmount)
				} else {
					remainingMakingAmountWei = order.AvailableMakingAmount
					if makerRemainingBalance := getMakerRemainingBalance(limit, filledMakingAmountByMaker, order.Maker, order.MakerAsset); makerRemainingBalance != nil && remainingMakingAmountWei.Cmp(makerRemainingBalance) > 0 {
						remainingMakingAmountWei = makerRemainingBalance
					}
				}
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
		_, takerAssetFee := p.calcTakerAssetFeeAmountExactOut(order, remainingTakingAmountWei)
		totalAmountIn = new(big.Int).Sub(new(big.Int).Sub(totalAmountIn, takerAssetFee), remainingTakingAmountWei)
		feeAmountWeiByOrder := p.calcMakerAsetFeeAmount(order, remainingMakingAmountWei)
		actualAmountOut := new(big.Int).Sub(remainingMakingAmountWei, feeAmountWeiByOrder)
		totalAmountOutWei = new(big.Int).Add(totalAmountOutWei, actualAmountOut)
		totalFeeAmountWei = new(big.Int).Add(totalFeeAmountWei, feeAmountWeiByOrder)
		filledOrderInfo := newFilledOrderInfo(order, remainingTakingAmountWei.String(), remainingMakingAmountWei.String(), feeAmountWeiByOrder.String())
		swapInfo.FilledOrders = append(swapInfo.FilledOrders, filledOrderInfo)
		addFilledMakingAmount(filledMakingAmountByMaker, order.Maker, remainingMakingAmountWei)
	}
	if !isFulfillAmountIn {
		return nil, SwapInfo{}, nil, ErrCannotFulfillAmountIn
	}
	return totalAmountOutWei, swapInfo, totalFeeAmountWei, nil
}

// feeAmount = (params.makingAmount * params.order.makerTokenFeePercent + BPS - 1) / BPS
func (p *PoolSimulator) calcMakerAsetFeeAmount(order *order, filledMakingAmount *big.Int) *big.Int {
	if order.IsTakerAssetFee {
		return constant.ZeroBI
	}
	if order.MakerTokenFeePercent == 0 {
		return constant.ZeroBI
	}
	amount := new(big.Int).Mul(filledMakingAmount, big.NewInt(int64(order.MakerTokenFeePercent)))
	return new(big.Int).Div(new(big.Int).Sub(new(big.Int).Add(amount, valueobject.BasisPoint), constant.One), valueobject.BasisPoint)
}

// given total takingAmount, calculate fee and takingAmountAfterFee
func (p *PoolSimulator) calcTakerAssetFeeAmountExactIn(order *order, takingAmount *big.Int) (takingAmountAfterFee *big.Int, fee *big.Int) {
	if !order.IsTakerAssetFee {
		return takingAmount, constant.ZeroBI
	}

	feePct := order.MakerTokenFeePercent // reuse same field
	if feePct == 0 {
		return takingAmount, constant.ZeroBI
	}

	// fee = ceiling(takingAmountAfterFee * feePct / BasisPoint)
	// takingAmountAfterFee + fee = takingAmount
	// => takingAmountAfterFee + ceiling(takingAmountAfterFee * feePct / BasisPoint) = takingAmount

	takingAmountAfterFee = new(big.Int).Div(
		new(big.Int).Mul(takingAmount, valueobject.BasisPoint),
		new(big.Int).Add(
			big.NewInt(int64(feePct)),
			valueobject.BasisPoint,
		),
	)
	fee = new(big.Int).Sub(takingAmount, takingAmountAfterFee)
	return
}

// given filled takingAmountAfterFee, calculate fee and total takingAmount
func (p *PoolSimulator) calcTakerAssetFeeAmountExactOut(order *order, takingAmountAfterFee *big.Int) (takingAmount *big.Int, fee *big.Int) {
	if !order.IsTakerAssetFee {
		return takingAmountAfterFee, constant.ZeroBI
	}

	feePct := order.MakerTokenFeePercent // reuse same field
	if feePct == 0 {
		return takingAmountAfterFee, constant.ZeroBI
	}

	amount := new(big.Int).Mul(takingAmountAfterFee, big.NewInt(int64(feePct)))
	fee = new(big.Int).Div(
		new(big.Int).Add(amount, valueobject.BasisPointM1),
		valueobject.BasisPoint,
	)

	takingAmount = new(big.Int).Add(takingAmountAfterFee, fee)
	return
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

// Inventory Limit

type makerAndAsset = string

func NewMakerAndAsset(maker, makerAsset string) makerAndAsset {
	return fmt.Sprintf("%v:%v", maker, makerAsset)
}

func (p *PoolSimulator) CalculateLimit() map[string]*big.Int {
	count := len(p.allMakersBalanceAllowance)
	if count == 0 {
		return nil
	}
	res := make(map[string]*big.Int, count)
	for k, v := range p.allMakersBalanceAllowance {
		res[k] = new(big.Int).Set(v)
	}
	return res
}

// Inventory is an alias for swaplimit.Inventory
// Deprecated: directly use swaplimit.Inventory.
type Inventory = swaplimit.Inventory

// NewInventory has key: "<maker>:<makerAsset>", value: maker's min(balance, allowance) for makerAsset
// Deprecated: directly use swaplimit.NewInventory.
func NewInventory(balance map[string]*big.Int) pool.SwapLimit {
	return swaplimit.NewInventory(DexTypeLimitOrder, balance)
}
