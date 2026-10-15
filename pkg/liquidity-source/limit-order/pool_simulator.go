package limitorder

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
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
var _ = pool.RegisterUseSwapLimit(valueobject.ExchangeKyberSwapLimitOrder)
var _ = pool.RegisterUseSwapLimit(valueobject.ExchangeKyberSwapLimitOrderDS)

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
				Address:     strings.ToLower(entityPool.Address),
				SwapFee:     big.NewInt(0),
				Exchange:    entityPool.Exchange,
				Type:        entityPool.Type,
				Tokens:      tokens,
				Reserves:    reserves,
				BlockNumber: entityPool.BlockNumber,
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

// addUsedMaking adds making to maker's running total of used balance.
func addUsedMaking(usedByMaker map[string]uint256.Int, maker string, making *uint256.Int) {
	total := usedByMaker[maker]
	total.Add(&total, making)
	usedByMaker[maker] = total
}

// setAmount sets z to b and reports whether b is a usable amount: non-nil, non-negative and
// below 2^256. z is 0 otherwise.
func setAmount(z *uint256.Int, b *big.Int) bool {
	if b == nil || b.Sign() < 0 || z.SetFromBig(b) {
		z.Clear()
		return false
	}
	return true
}

// backedBigInt is a result big.Int with its own word storage, so it costs one allocation.
type backedBigInt struct {
	b big.Int
	w [4]big.Word
}

// amountToBig converts z, keeping zero as big.Int's canonical zero (nil abs) like big.NewInt(0),
// which a word-backed conversion does not, so results stay reflect.DeepEqual to big.Int ones.
func amountToBig(z *uint256.Int) *big.Int {
	if z.IsZero() {
		return new(big.Int)
	}
	r := new(backedBigInt)
	return big256.ToBigBacked(&r.b, &r.w, z)
}

// amountString formats a stored amount; uint256 does it in one allocation, big.Int in two.
func amountString(b *big.Int) string {
	var z uint256.Int
	if setAmount(&z, b) {
		return z.Dec()
	}
	return b.String()
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
	var amountIn uint256.Int
	if !setAmount(&amountIn, tokenAmountIn.Amount) {
		return nil, SwapInfo{}, nil, ErrCannotFulfillAmountIn
	}

	swapInfo := SwapInfo{
		FilledOrders: []*FilledOrderInfo{},
		SwapSide:     swapSide,
		AmountIn:     amountIn.Dec(),
	}
	isFulfillAmountIn := false

	// we need to update maker's remaining balance in 2 places:
	// - in UpdateBalance: mainly to deal with case where maker has orders with same makerAsset but different takerAsset
	// - when simulating filling each order here: we cannot do the same as in kyber-pmm (simulating first then check inventory limit at the end)
	//				because in LO we have multiple makers, and also because we still need to allow orders that have part of the balance available
	//		the problem is that in this func we cannot update the limit,
	//		so we'll use this map to track how much of each maker's balance is used, then subtract from the original balance, to have the remaining balance available
	usedByMaker := make(map[string]uint256.Int, len(p.allMakersBalanceAllowance))
	balances := makerBalances{limit: limit}

	var (
		totalAmountOut, totalFeeAmount, totalMakingAmount             uint256.Int
		ownMaking, remainingMaking, remainingTaking, amountInAfterFee uint256.Int
		terms                                                         orderTerms
		fill                                                          orderFill
	)
	for i, orderID := range orderIDs {
		order, ok := p.ordersMapping[orderID]
		if !ok {
			return nil, swapInfo, nil, fmt.Errorf("order %d is not existed in pool", orderID)
		}

		// Get remaining making amount, taking amount
		if !order.remainingAmount(&terms, &ownMaking, &remainingMaking, &remainingTaking, limit, usedByMaker) {
			continue
		}

		// ideally we should return totalFeeAmountWei in takerAsset here
		// but for now it's not used, and we might get mixed up with makerAsset fee, so will ignore for now
		if terms.amountAfterTakerFee(&amountInAfterFee, &amountIn).IsZero() {
			continue
		}

		totalMakingAmount.Add(&totalMakingAmount, &remainingMaking)

		done := fillOrder(&fill, &terms, &remainingTaking, &amountInAfterFee)
		if fill.making.IsZero() { // order too small
			continue
		}
		totalFeeAmount.Add(&totalFeeAmount, &fill.makerFee)
		totalAmountOut.Add(&totalAmountOut, &fill.making)
		totalAmountOut.Sub(&totalAmountOut, &fill.makerFee)

		filledTakingStr, filledMakingStr := fill.taking.Dec(), fill.making.Dec()
		// a used-up order not capped by its maker's balance has its own remaining making as fill
		ownMakingStr := filledMakingStr
		if !ownMaking.Eq(&fill.making) {
			ownMakingStr = ownMaking.Dec()
		}
		info := newFilledOrderInfo(order, filledTakingStr, filledMakingStr, fill.makerFee.Dec(), ownMakingStr)
		info.MakerBalance = balances.of(order)
		swapInfo.FilledOrders = append(swapInfo.FilledOrders, info)
		// like the executor, set the order's whole capped making aside, not just its fill
		addUsedMaking(usedByMaker, order.Maker, &remainingMaking)
		if !done {
			amountIn.Sub(&amountIn, &fill.spent)
			continue
		}
		isFulfillAmountIn = true

		// Currently, when Aggregator finds route and returns some orders and sends them to the smart contract to execute.
		// We will often meet edge cases that these orders can be fulfilled by a trading bot or taker on Aggregator.
		// From that, the estimated amount out and filled orders are not correct. So we need to add more orders when sending to SC to the executor.
		// In this case, we will some orders util total MakingAmount(remainMakingAmount)/estimated amountOut >= 1.3 (130%)
		threshold := fallbackThreshold(&totalAmountOut)
		for _, fallbackID := range orderIDs[i+1:] {
			if !totalMakingAmount.Lt(&threshold) {
				break
			}
			order, ok := p.ordersMapping[fallbackID]
			if !ok {
				continue
			}
			if !order.remainingAmount(&terms, &ownMaking, &remainingMaking, &remainingTaking, limit, usedByMaker) {
				continue
			}

			totalMakingAmount.Add(&totalMakingAmount, &remainingMaking)
			info := newFallbackOrderInfo(order, ownMaking.Dec())
			info.MakerBalance = balances.of(order)
			swapInfo.FilledOrders = append(swapInfo.FilledOrders, info)
		}
		break
	}
	if !isFulfillAmountIn {
		return nil, SwapInfo{}, nil, ErrCannotFulfillAmountIn
	}
	return amountToBig(&totalAmountOut), swapInfo, amountToBig(&totalFeeAmount), nil
}

// fallbackThreshold is ceil(amountOut * FallbackPercentageOfTotalMakingAmount): fallback orders
// are added until the remaining making amount reaches it. Computed with the same big.Float
// rounding as before, then compared exactly as an integer.
func fallbackThreshold(amountOut *uint256.Int) uint256.Int {
	f := new(big.Float).SetInt(amountOut.ToBig())
	f.Mul(f, FallbackPercentageOfTotalMakingAmount)
	t, accuracy := f.Int(nil)
	if accuracy == big.Below {
		t.Add(t, bignumber.One)
	}
	var threshold uint256.Int
	if t.Sign() > 0 && threshold.SetFromBig(t) {
		threshold.SetAllOne() // beyond 256 bits: never reached
	}
	return threshold
}

// orderTerms holds an order's price and fee terms for the fill math.
type orderTerms struct {
	making, taking uint256.Int // the order's total amounts
	feePct         uint256.Int // MakerTokenFeePercent, on the taker asset when takerAssetFee
	takerAssetFee  bool
}

// feePcts returns the maker-asset and the taker-asset fee pct; at most one is non-zero.
func (t *orderTerms) feePcts() (maker, taker uint64) {
	if t.takerAssetFee {
		return 0, t.feePct.Uint64()
	}
	return t.feePct.Uint64(), 0
}

// makerFee sets z to ceil(making * feePct / BPS) for a maker-asset fee, else 0.
func (t *orderTerms) makerFee(z, making *uint256.Int) *uint256.Int {
	if t.takerAssetFee || t.feePct.IsZero() {
		return z.Clear()
	}
	return big256.MulDivUp(z, making, &t.feePct, big256.UBasisPoint)
}

// amountAfterTakerFee sets z to the most taking that, plus its taker-asset fee
// ceil(taking * feePct / BPS), fits in amountIn: amountIn * BPS / (BPS + feePct).
func (t *orderTerms) amountAfterTakerFee(z, amountIn *uint256.Int) *uint256.Int {
	if !t.takerAssetFee || t.feePct.IsZero() {
		return z.Set(amountIn)
	}
	var denominator uint256.Int
	denominator.Add(big256.UBasisPoint, &t.feePct)
	return big256.MulDivDown(z, amountIn, big256.UBasisPoint, &denominator)
}

// amountWithTakerFee sets z to taking plus its taker-asset fee ceil(taking * feePct / BPS).
func (t *orderTerms) amountWithTakerFee(z, taking *uint256.Int) {
	if !t.takerAssetFee || t.feePct.IsZero() {
		z.Set(taking)
		return
	}
	var fee uint256.Int
	big256.MulDivUp(&fee, taking, &t.feePct, big256.UBasisPoint)
	z.Add(taking, &fee)
}

// orderFill is one order's share of an exact-in fill.
type orderFill struct {
	taking, making, makerFee uint256.Int
	spent                    uint256.Int // what a used-up order took from amountIn, taker-asset fee included
}

// fillOrder places amountInAfterFee into the order with terms t and remTaking left, as the
// executor does: it sends min(remTaking, amountInAfterFee) and the LO contract gives
// floor(taking * M / T) of the maker asset. done reports the order takes
// all of amountInAfterFee (making is 0 when that buys no maker wei); otherwise the order is used up
// and the next gets amountIn less spent. Shared by calcAmountOutWithSwapInfo and refillSignedOrders.
func fillOrder(f *orderFill, t *orderTerms, remTaking, amountInAfterFee *uint256.Int) (done bool) {
	if !remTaking.Lt(amountInAfterFee) {
		f.taking.Set(amountInAfterFee)
		done = true
	} else {
		f.taking.Set(remTaking)
		t.amountWithTakerFee(&f.spent, remTaking)
	}
	big256.MulDivDown(&f.making, &f.taking, &t.making, &t.taking)
	t.makerFee(&f.makerFee, &f.making)

	return done
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

		return orderAllowedSender == valueobject.AddrZero || lo.Contains(allowedSendersAddress, orderAllowedSender)
	})
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.MetaInfo{ApprovalAddress: p.contractAddress, BlockNumber: p.Info.BlockNumber}
}

func newFallbackOrderInfo(order *order, remainingMakingAmount string) *FilledOrderInfo {
	orderInfo := newFilledOrderInfo(order, "", "", "", remainingMakingAmount)
	orderInfo.markFallback()
	return orderInfo
}

// makerBalances formats each maker's balance in the swap limit once per quote, for
// FilledOrderInfo.MakerBalance; empty without a limit.
type makerBalances struct {
	limit   pool.SwapLimit
	byMaker map[string]string
}

func (m *makerBalances) of(o *order) string {
	if m.limit == nil {
		return ""
	}
	if s, ok := m.byMaker[o.Maker]; ok {
		return s
	}
	if m.byMaker == nil {
		m.byMaker = make(map[string]string, 4)
	}
	var s string
	if b := m.limit.GetLimit(NewMakerAndAsset(o.Maker, o.MakerAsset)); b != nil {
		s = amountString(b)
	}
	m.byMaker[o.Maker] = s
	return s
}

// markFallback clears f's fill and marks it a fallback, which the executor fills only with what
// the orders before it leave of amountIn.
func (f *FilledOrderInfo) markFallback() {
	f.FilledTakingAmount, f.FilledMakingAmount, f.FeeAmount, f.IsFallBack = "0", "0", "0", true
}

func newFilledOrderInfo(order *order, filledTakingAmount, filledMakingAmount, feeAmount,
	remainingMakingAmount string) *FilledOrderInfo {
	feeConfig := ""
	if order.FeeConfig != nil {
		feeConfig = amountString(order.FeeConfig)
	}
	return &FilledOrderInfo{
		OrderID:              order.ID,
		OrderHash:            order.OrderHash,
		FilledTakingAmount:   filledTakingAmount,
		FilledMakingAmount:   filledMakingAmount,
		TakingAmount:         amountString(order.TakingAmount),
		MakingAmount:         amountString(order.MakingAmount),
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

		RemainingMakingAmount: remainingMakingAmount,
		IsTakerAssetFee:       order.IsTakerAssetFee,
	}
}

// Inventory Limit

type makerAndAsset = string

func NewMakerAndAsset(maker, makerAsset string) makerAndAsset {
	return maker + ":" + makerAsset
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

func (s *PoolSimulator) CanSwapFrom(address string) []string {
	if s.GetTokenIndex(address) == -1 {
		return nil
	}
	result := make([]string, 0, len(s.Info.Tokens)-1)
	for _, token := range s.GetTokens() {
		if token == address {
			continue
		}
		orderIds := s.getOrderIDsBySwapSide(s.getSwapSide(address, token))
		if len(orderIds) != 0 {
			result = append(result, token)
		}
	}

	return result
}

func (s *PoolSimulator) CanSwapTo(address string) []string {
	if s.GetTokenIndex(address) == -1 {
		return nil
	}
	result := make([]string, 0, len(s.Info.Tokens)-1)
	for _, token := range s.GetTokens() {
		if token == address {
			continue
		}
		orderIds := s.getOrderIDsBySwapSide(s.getSwapSide(token, address))
		if len(orderIds) != 0 {
			result = append(result, token)
		}
	}

	return result
}
