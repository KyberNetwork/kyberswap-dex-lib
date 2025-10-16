package limitorder

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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

var _ = pool.RegisterFactory0(DexTypeLimitOrder, NewPoolSimulator)
var _ = pool.RegisterUseSwapLimit(DexTypeLimitOrder)

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

	var contractAddress string
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		// this is optional for now, will be changed to required later
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
				Address:  strings.ToLower(entityPool.Address),
				SwapFee:  big.NewInt(0),
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   tokens,
				Reserves: reserves,
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

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.ordersMapping = lo.MapEntries(p.ordersMapping, func(k int64, v *order) (int64, *order) {
		c := *v
		c.FilledTakingAmount = new(big.Int).Set(v.FilledTakingAmount)
		c.FilledMakingAmount = new(big.Int).Set(v.FilledMakingAmount)
		if c.AvailableMakingAmount != nil {
			c.AvailableMakingAmount = new(big.Int).Set(v.AvailableMakingAmount)
		}
		return k, &c
	})
	return &cloned
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
	amountOut, swapInfo, feeAmount, err := p.calcAmountOutWithSwapInfo(swapSide, tokenAmountIn, limit)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenOut,
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

func (p *PoolSimulator) calcAmountOutWithSwapInfo(swapSide SwapSide, tokenAmountIn pool.TokenAmount, limit pool.SwapLimit) (*big.Int, SwapInfo, *big.Int, error) {
	orderIDs := p.getOrderIDsBySwapSide(swapSide)
	if limit != nil {
		// EX-2684: Filter out orders that are not in allowedSenders list.
		orderIDs = p.filterOrdersByAllowedSenders(orderIDs, limit.GetAllowedSenders())
	}
	if len(orderIDs) == 0 {
		return big.NewInt(0), SwapInfo{}, nil, nil
	}

	totalAmountOutWei := big.NewInt(0)
	totalAmountIn := new(big.Int).Set(tokenAmountIn.Amount)

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

		// Get remaining making amount, taking amount
		remainingMakingAmountWei, remainingTakingAmountWei := order.RemainingAmount(limit, filledMakingAmountByMaker)
		if remainingMakingAmountWei.Sign() <= 0 || remainingTakingAmountWei.Sign() <= 0 {
			continue
		}

		// ideally we should return totalFeeAmountWei in takerAsset here
		// but for now it's not used, and we might get mixed up with makerAsset fee, so will ignore for now
		totalAmountInAfterFee, _ := p.calcTakerAssetFeeAmountExactIn(order, totalAmountIn)
		if totalAmountInAfterFee.Sign() <= 0 {
			continue
		}

		totalMakingAmountWei = new(big.Int).Add(totalMakingAmountWei, remainingMakingAmountWei)

		if remainingTakingAmountWei.Cmp(totalAmountInAfterFee) >= 0 {
			filledTakingAmountWei := totalAmountInAfterFee
			filledMakingAmountWei := new(big.Int).Div(
				new(big.Int).Mul(filledTakingAmountWei, order.MakingAmount),
				order.TakingAmount,
			) // filledMakingAmountWei = filledTakingAmountWei * order.MakingAmount / order.TakingAmount

			// order too small
			if filledMakingAmountWei.Sign() <= 0 {
				continue
			}

			feeAmountWeiByOrder := p.calcMakerAssetFeeAmount(order, filledMakingAmountWei)
			totalFeeAmountWei.Add(totalFeeAmountWei, feeAmountWeiByOrder)
			actualAmountOut := new(big.Int).Sub(filledMakingAmountWei, feeAmountWeiByOrder)
			totalAmountOutWei.Add(totalAmountOutWei, actualAmountOut)
			filledOrderInfo := newFilledOrderInfo(order, filledTakingAmountWei.String(), filledMakingAmountWei.String(), feeAmountWeiByOrder.String())
			swapInfo.FilledOrders = append(swapInfo.FilledOrders, filledOrderInfo)
			isFulfillAmountIn = true
			addFilledMakingAmount(filledMakingAmountByMaker, order.Maker, filledMakingAmountWei)

			// Currently, when Aggregator finds route and returns some orders and sends them to the smart contract to execute.
			// We will often meet edge cases that these orders can be fulfilled by a trading bot or taker on Aggregator.
			// From that, the estimated amount out and filled orders are not correct. So we need to add more orders when sending to SC to the executor.
			// In this case, we will some orders util total MakingAmount(remainMakingAmount)/estimated amountOut >= 1.3 (130%)
			totalAmountOutWeiBigFloat := new(big.Float).SetInt(totalAmountOutWei)
			// threshold = totalAmountOutWei * FallbackPercentageOfTotalMakingAmount
			threshold := new(big.Float).Mul(totalAmountOutWeiBigFloat, FallbackPercentageOfTotalMakingAmount)
			for j := i + 1; j < len(orderIDs); j++ {
				if new(big.Float).SetInt(totalMakingAmountWei).Cmp(threshold) >= 0 {
					break
				}
				order, ok := p.ordersMapping[orderIDs[j]]
				if !ok {
					continue
				}
				remainingMakingAmountWei, remainingTakingAmountWei := order.RemainingAmount(limit, filledMakingAmountByMaker)
				if remainingMakingAmountWei.Sign() <= 0 || remainingTakingAmountWei.Sign() <= 0 {
					continue
				}

				totalMakingAmountWei = new(big.Int).Add(totalMakingAmountWei, remainingMakingAmountWei)
				filledOrderInfo := newFallbackOrderInfo(order)
				swapInfo.FilledOrders = append(swapInfo.FilledOrders, filledOrderInfo)
			}
			break
		}
		_, takerAssetFee := p.calcTakerAssetFeeAmountExactOut(order, remainingTakingAmountWei)
		totalAmountIn.Sub(totalAmountIn, takerAssetFee)
		totalAmountIn.Sub(totalAmountIn, remainingTakingAmountWei)
		feeAmountWeiByOrder := p.calcMakerAssetFeeAmount(order, remainingMakingAmountWei)
		actualAmountOut := new(big.Int).Sub(remainingMakingAmountWei, feeAmountWeiByOrder)
		totalAmountOutWei.Add(totalAmountOutWei, actualAmountOut)
		totalFeeAmountWei.Add(totalFeeAmountWei, feeAmountWeiByOrder)
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
func (p *PoolSimulator) calcMakerAssetFeeAmount(order *order, filledMakingAmount *big.Int) *big.Int {
	if order.IsTakerAssetFee {
		return big.NewInt(0)
	}
	if order.MakerTokenFeePercent == 0 {
		return big.NewInt(0)
	}
	amount := new(big.Int).Mul(filledMakingAmount, big.NewInt(int64(order.MakerTokenFeePercent)))
	return new(big.Int).Div(new(big.Int).Sub(new(big.Int).Add(amount, valueobject.BasisPoint), bignumber.One), valueobject.BasisPoint)
}

// given total takingAmount, calculate fee and takingAmountAfterFee
func (p *PoolSimulator) calcTakerAssetFeeAmountExactIn(order *order, takingAmount *big.Int) (takingAmountAfterFee *big.Int, fee *big.Int) {
	if !order.IsTakerAssetFee {
		return new(big.Int).Set(takingAmount), big.NewInt(0)
	}

	feePct := order.MakerTokenFeePercent // reuse same field
	if feePct == 0 {
		return new(big.Int).Set(takingAmount), big.NewInt(0)
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
		return new(big.Int).Set(takingAmountAfterFee), big.NewInt(0)
	}

	feePct := order.MakerTokenFeePercent // reuse same field
	if feePct == 0 {
		return new(big.Int).Set(takingAmountAfterFee), big.NewInt(0)
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

// filterOrdersByAllowedSenders returns orderIDs that have order.allowedSender
// either in the allowedSenders, or is empty value.
func (p *PoolSimulator) filterOrdersByAllowedSenders(orderIDs []int64, allowedSenders string) []int64 {
	allowedSendersSlice := lo.Filter(strings.Split(allowedSenders, ","), func(s string, _ int) bool {
		return s != ""
	})

	if len(allowedSendersSlice) == 0 {
		return orderIDs
	}

	allowedSendersAddress := lo.Map(allowedSendersSlice, func(s string, _ int) common.Address {
		return common.HexToAddress(s)
	})

	return lo.Filter(orderIDs, func(orderID int64, _ int) bool {
		order := p.ordersMapping[orderID]
		orderAllowedSender := common.HexToAddress(order.AllowedSenders)
		// order.AllowedSenders can be multiple, separate by ','.
		// We only check for single allowedSenders address for now.

		return orderAllowedSender == (common.Address{}) || lo.Contains(allowedSendersAddress, orderAllowedSender)
	})
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.ApprovalInfo{ApprovalAddress: p.contractAddress}
}

func newFallbackOrderInfo(order *order) *FilledOrderInfo {
	orderInfo := newFilledOrderInfo(order, "0", "0", "0")
	orderInfo.IsFallBack = true
	return orderInfo
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
