package wcm

import (
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
)

type PoolSimulator struct {
	pool.Pool
	StaticExtra StaticExtra
	Extra       Extra
	Gas         Gas

	buyTokenDecs, payTokenDecs uint8
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	info := pool.PoolInfo{
		Address:     entityPool.Address,
		Exchange:    entityPool.Exchange,
		Type:        entityPool.Type,
		Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
		Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
		BlockNumber: entityPool.BlockNumber,
	}

	return &PoolSimulator{
		Pool:         pool.Pool{Info: info},
		StaticExtra:  staticExtra,
		Extra:        extra,
		Gas:          DefaultGas,
		buyTokenDecs: entityPool.Tokens[0].Decimals,
		payTokenDecs: entityPool.Tokens[1].Decimals,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if !s.isValidTokenPair(param.TokenAmountIn.Token, param.TokenOut) {
		return nil, ErrInvalidTokenPair
	}
	if s.Extra.IsHalted {
		return nil, ErrPoolHalted
	}
	isBuy := s.isBuyOrder(param.TokenAmountIn.Token, param.TokenOut)

	var amountOut, grossBase, grossQuote *big.Int
	var totalGrossBasePD *big.Int
	var executedLevels int
	var err error

	if isBuy {
		walkInput := scaleAmountDecimals(param.TokenAmountIn.Amount, s.payTokenDecs, s.StaticExtra.BuyTokenPositionDecimals)

		totalGrossBasePD, executedLevels, err = s.executeAskOrders(walkInput)
		if err != nil {
			return nil, err
		}

		if s.Extra.MinOrderQuantity != nil && totalGrossBasePD.Cmp(s.Extra.MinOrderQuantity) < 0 {
			return nil, ErrQuantityTooLow
		}

		feeBasePD := calcSpotFee(totalGrossBasePD, s.Extra.TakerFeeMultiplier, s.Extra.FromMaxFee)
		totalNetBasePD := new(big.Int).Sub(totalGrossBasePD, feeBasePD)

		amountOut = scaleAmountDecimals(totalNetBasePD, s.StaticExtra.BuyTokenPositionDecimals, s.buyTokenDecs)

		grossBase = scaleAmountDecimals(totalGrossBasePD, s.StaticExtra.BuyTokenPositionDecimals, s.buyTokenDecs)
		grossQuote = param.TokenAmountIn.Amount
	} else {
		grossBase = scaleAmountDecimals(param.TokenAmountIn.Amount, s.buyTokenDecs, s.StaticExtra.BuyTokenPositionDecimals)

		if s.Extra.MinOrderQuantity != nil && grossBase.Cmp(s.Extra.MinOrderQuantity) < 0 {
			return nil, ErrQuantityTooLow
		}

		var totalGrossQuotePD *big.Int
		totalGrossQuotePD, executedLevels, err = s.executeBidOrders(grossBase)
		if err != nil {
			return nil, err
		}

		feeQuotePD := calcSpotFee(totalGrossQuotePD, s.Extra.TakerFeeMultiplier, s.Extra.ToMaxFee)
		totalNetQuotePD := new(big.Int).Sub(totalGrossQuotePD, feeQuotePD)

		amountOut = scaleAmountDecimals(totalNetQuotePD, s.StaticExtra.BuyTokenPositionDecimals, s.payTokenDecs)

		grossBase = param.TokenAmountIn.Amount
		grossQuote = scaleAmountDecimals(totalGrossQuotePD, s.StaticExtra.BuyTokenPositionDecimals, s.payTokenDecs)
	}

	if amountOut.Sign() <= 0 {
		return nil, ErrAmountOutTooSmall
	}

	gas := s.Gas.Base + int64(executedLevels)*s.Gas.Level

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  param.TokenOut,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  param.TokenOut,
			Amount: Zero,
		},
		Gas: gas,
		SwapInfo: SwapInfo{
			IsBuy:          isBuy,
			ExecutedLevels: executedLevels,
			GrossBase:      grossBase,
			GrossQuote:     grossQuote,
		},
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	if !s.isValidTokenPair(param.TokenIn, param.TokenAmountOut.Token) {
		return nil, ErrInvalidTokenPair
	}
	if s.Extra.IsHalted {
		return nil, ErrPoolHalted
	}

	if param.TokenAmountOut.Amount == nil || param.TokenAmountOut.Amount.Sign() <= 0 {
		return nil, ErrInvalidAmountOut
	}

	isBuy := s.isBuyOrder(param.TokenIn, param.TokenAmountOut.Token)

	var amountIn *big.Int
	var err error
	var executedLevels int
	var amountInVal, grossBaseVal, grossQuoteVal *big.Int

	if isBuy {
		netBase := scaleAmountDecimalsRounding(param.TokenAmountOut.Amount, s.buyTokenDecs, s.StaticExtra.BuyTokenPositionDecimals, true)

		grossBase := s.getGrossAmount(netBase, s.Extra.TakerFeeMultiplier, s.Extra.FromMaxFee)

		amountInVal, executedLevels, err = s.calculateQuoteTokenNeeded(grossBase)
		if err != nil {
			return nil, err
		}

		if s.Extra.MinOrderQuantity != nil && grossBase.Cmp(s.Extra.MinOrderQuantity) < 0 {
			return nil, ErrQuantityTooLow
		}
		amountIn = scaleAmountDecimalsRounding(amountInVal, s.StaticExtra.BuyTokenPositionDecimals, s.payTokenDecs, true)

		grossBaseVal = scaleAmountDecimals(grossBase, s.StaticExtra.BuyTokenPositionDecimals, s.buyTokenDecs)
		grossQuoteVal = amountIn
	} else {
		netQuote := scaleAmountDecimalsRounding(param.TokenAmountOut.Amount, s.payTokenDecs, s.StaticExtra.PayTokenPositionDecimals, true)

		grossQuote := s.getGrossAmount(netQuote, s.Extra.TakerFeeMultiplier, s.Extra.ToMaxFee)

		grossQuoteBasePD := scaleAmountDecimals(grossQuote, s.StaticExtra.PayTokenPositionDecimals, s.StaticExtra.BuyTokenPositionDecimals)

		amountInVal, executedLevels, err = s.calculateBaseTokenNeeded(grossQuoteBasePD)
		if err != nil {
			return nil, err
		}

		if s.Extra.MinOrderQuantity != nil && amountInVal.Cmp(s.Extra.MinOrderQuantity) < 0 {
			return nil, ErrQuantityTooLow
		}

		amountIn = scaleAmountDecimalsRounding(amountInVal, s.StaticExtra.BuyTokenPositionDecimals, s.buyTokenDecs, true)

		grossBaseVal = amountIn
		grossQuoteVal = scaleAmountDecimals(grossQuote, s.StaticExtra.PayTokenPositionDecimals, s.payTokenDecs)
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  param.TokenIn,
			Amount: amountIn,
		},
		Fee: &pool.TokenAmount{
			Token:  param.TokenAmountOut.Token,
			Amount: Zero,
		},
		Gas: s.Gas.Base + int64(executedLevels)*s.Gas.Level,
		SwapInfo: SwapInfo{
			IsBuy:          isBuy,
			ExecutedLevels: executedLevels,
			GrossBase:      grossBaseVal,
			GrossQuote:     grossQuoteVal,
		},
	}, nil
}

func (s *PoolSimulator) getGrossAmount(net, takerFeeMultiplier, maxFee *big.Int) *big.Int {
	if takerFeeMultiplier.Sign() == 0 {
		return new(big.Int).Set(net)
	}

	if maxFee != nil {
		grossWithMax := new(big.Int).Add(net, maxFee)
		fee := new(big.Int).Mul(grossWithMax, takerFeeMultiplier)
		fee.Div(fee, FeeDivisor)
		if fee.Cmp(maxFee) >= 0 {
			return grossWithMax
		}
	}

	denom := new(big.Int).Sub(FeeDivisor, takerFeeMultiplier)

	gross := new(big.Int).Mul(net, FeeDivisor)
	gross.Add(gross, new(big.Int).Sub(denom, One))
	gross.Div(gross, denom)

	return gross
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	if params.TokenAmountIn.Amount == nil || params.TokenAmountIn.Amount.Sign() <= 0 {
		return
	}
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}

	baseAmountPD := scaleAmountDecimals(si.GrossBase, s.buyTokenDecs, s.StaticExtra.BuyTokenPositionDecimals)

	if si.IsBuy {
		s.Extra.OrderBook.Asks = s.reduceOrders(s.Extra.OrderBook.Asks, baseAmountPD)
	} else {
		s.Extra.OrderBook.Bids = s.reduceOrders(s.Extra.OrderBook.Bids, baseAmountPD)
	}

	baseRes, quoteRes := s.calculateLiquidity()
	s.Info.Reserves = []*big.Int{baseRes, quoteRes}
}

func (s *PoolSimulator) reduceOrders(levels []OrderBookLevel, baseAmountPD *big.Int) []OrderBookLevel {
	if baseAmountPD == nil || baseAmountPD.Sign() <= 0 {
		return levels
	}
	remaining := new(big.Int).Set(baseAmountPD)
	for i := range levels {
		if remaining.Sign() <= 0 {
			break
		}
		level := &levels[i]
		if remaining.Cmp(level.Quantity) >= 0 {
			remaining.Sub(remaining, level.Quantity)
			level.Quantity.SetInt64(0)
		} else {
			level.Quantity.Sub(level.Quantity, remaining)
			remaining.SetInt64(0)
		}
	}
	return compactLevels(levels)
}

func (s *PoolSimulator) calculateLiquidity() (*big.Int, *big.Int) {
	baseReservePD := new(big.Int)
	for _, ask := range s.Extra.OrderBook.Asks {
		baseReservePD.Add(baseReservePD, ask.Quantity)
	}
	baseReserve := scaleAmountDecimals(baseReservePD, s.StaticExtra.BuyTokenPositionDecimals, s.buyTokenDecs)

	quoteReservePD := new(big.Int)
	for _, bid := range s.Extra.OrderBook.Bids {
		quoteAmountPD := new(big.Int).Mul(bid.Quantity, bid.Price)
		quoteAmountPD.Div(quoteAmountPD, PricePrecisionMultiplier)
		quoteReservePD.Add(quoteReservePD, quoteAmountPD)
	}
	quoteReserve := scaleAmountDecimals(quoteReservePD, s.StaticExtra.BuyTokenPositionDecimals, s.payTokenDecs)

	return baseReserve, quoteReserve
}

func compactLevels(levels []OrderBookLevel) []OrderBookLevel {
	n := 0
	for i := range levels {
		if levels[i].Quantity != nil && levels[i].Quantity.Sign() > 0 {
			if n != i {
				levels[n] = levels[i]
			}
			n++
		}
	}
	return levels[:n]
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) interface{} {
	return pool.ApprovalInfo{
		ApprovalAddress: s.StaticExtra.Router,
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s

	cloned.Info.Reserves = make([]*big.Int, len(s.Info.Reserves))
	for i, r := range s.Info.Reserves {
		cloned.Info.Reserves[i] = new(big.Int).Set(r)
	}

	cloned.Extra.OrderBook.Bids = make([]OrderBookLevel, len(s.Extra.OrderBook.Bids))
	for i, bid := range s.Extra.OrderBook.Bids {
		cloned.Extra.OrderBook.Bids[i] = bid
		cloned.Extra.OrderBook.Bids[i].Quantity = new(big.Int).Set(bid.Quantity)
	}

	cloned.Extra.OrderBook.Asks = make([]OrderBookLevel, len(s.Extra.OrderBook.Asks))
	for i, ask := range s.Extra.OrderBook.Asks {
		cloned.Extra.OrderBook.Asks[i] = ask
		cloned.Extra.OrderBook.Asks[i].Quantity = new(big.Int).Set(ask.Quantity)
	}

	return &cloned
}

func calcSpotFee(quantity, takerFeeMultiplier, maxFee *big.Int) *big.Int {
	if takerFeeMultiplier.Sign() == 0 {
		return new(big.Int)
	}

	fee := new(big.Int).Mul(quantity, takerFeeMultiplier)
	fee.Div(fee, FeeDivisor)

	if maxFee != nil && fee.Cmp(maxFee) > 0 {
		fee = new(big.Int).Set(maxFee)
	}
	return fee
}

func (s *PoolSimulator) isValidTokenPair(tokenIn, tokenOut string) bool {
	base, quote := s.baseToken(), s.quoteToken()
	if base == "" || quote == "" {
		return false
	}
	return (strings.EqualFold(tokenIn, quote) && strings.EqualFold(tokenOut, base)) ||
		(strings.EqualFold(tokenIn, base) && strings.EqualFold(tokenOut, quote))
}

func (s *PoolSimulator) isBuyOrder(tokenIn, tokenOut string) bool {
	return strings.EqualFold(tokenIn, s.quoteToken()) &&
		strings.EqualFold(tokenOut, s.baseToken())
}

func (s *PoolSimulator) baseToken() string {
	return s.Info.Tokens[0]
}

func (s *PoolSimulator) quoteToken() string {
	return s.Info.Tokens[1]
}

func (s *PoolSimulator) executeAskOrders(quoteAmount *big.Int) (*big.Int, int, error) {
	if len(s.Extra.OrderBook.Asks) == 0 {
		return nil, 0, ErrEmptyOrderBook
	}

	remainingQuote := new(big.Int).Set(quoteAmount)
	totalBaseReceivedPD := new(big.Int)
	executedLevels := 0

	for _, ask := range s.Extra.OrderBook.Asks {
		if remainingQuote.Sign() <= 0 {
			break
		}

		quoteNeeded := new(big.Int).Mul(ask.Quantity, ask.Price)
		quoteNeeded.Div(quoteNeeded, PricePrecisionMultiplier)

		if remainingQuote.Cmp(quoteNeeded) >= 0 {
			totalBaseReceivedPD.Add(totalBaseReceivedPD, ask.Quantity)
			remainingQuote.Sub(remainingQuote, quoteNeeded)
			executedLevels++
		} else {
			partialBasePD := new(big.Int).Mul(remainingQuote, PricePrecisionMultiplier)
			partialBasePD.Div(partialBasePD, ask.Price)
			totalBaseReceivedPD.Add(totalBaseReceivedPD, partialBasePD)
			remainingQuote.SetInt64(0)
			executedLevels++
		}
	}

	if remainingQuote.Sign() > 0 {
		return nil, 0, ErrInsufficientLiquidity
	}

	return totalBaseReceivedPD, executedLevels, nil
}

func (s *PoolSimulator) executeBidOrders(baseAmount *big.Int) (*big.Int, int, error) {
	if len(s.Extra.OrderBook.Bids) == 0 {
		return nil, 0, ErrEmptyOrderBook
	}

	remainingBasePD := new(big.Int).Set(baseAmount)
	totalQuoteReceivedPD := new(big.Int)
	executedLevels := 0

	for _, bid := range s.Extra.OrderBook.Bids {
		if remainingBasePD.Sign() <= 0 {
			break
		}

		if remainingBasePD.Cmp(bid.Quantity) >= 0 {
			quoteReceivedPD := new(big.Int).Mul(bid.Quantity, bid.Price)
			quoteReceivedPD.Div(quoteReceivedPD, PricePrecisionMultiplier)
			totalQuoteReceivedPD.Add(totalQuoteReceivedPD, quoteReceivedPD)
			remainingBasePD.Sub(remainingBasePD, bid.Quantity)
			executedLevels++
		} else {
			partialQuotePD := new(big.Int).Mul(remainingBasePD, bid.Price)
			partialQuotePD.Div(partialQuotePD, PricePrecisionMultiplier)
			totalQuoteReceivedPD.Add(totalQuoteReceivedPD, partialQuotePD)
			remainingBasePD.SetInt64(0)
			executedLevels++
		}
	}

	if remainingBasePD.Sign() > 0 {
		return nil, 0, ErrInsufficientLiquidity
	}

	return totalQuoteReceivedPD, executedLevels, nil
}

func (s *PoolSimulator) calculateQuoteTokenNeeded(baseAmountWanted *big.Int) (*big.Int, int, error) {
	if len(s.Extra.OrderBook.Asks) == 0 {
		return nil, 0, ErrEmptyOrderBook
	}

	remainingBase := new(big.Int).Set(baseAmountWanted)
	totalQuoteNeeded := new(big.Int)
	executedLevels := 0

	for _, ask := range s.Extra.OrderBook.Asks {
		if remainingBase.Sign() <= 0 {
			break
		}
		executedLevels++

		if remainingBase.Cmp(ask.Quantity) >= 0 {
			quoteNeeded := new(big.Int).Mul(ask.Quantity, ask.Price)
			quoteNeeded.Div(quoteNeeded, PricePrecisionMultiplier)
			totalQuoteNeeded.Add(totalQuoteNeeded, quoteNeeded)
			remainingBase.Sub(remainingBase, ask.Quantity)
		} else {
			partialQuote := new(big.Int).Mul(remainingBase, ask.Price)
			partialQuote.Add(partialQuote, new(big.Int).Sub(PricePrecisionMultiplier, One))
			partialQuote.Div(partialQuote, PricePrecisionMultiplier)
			totalQuoteNeeded.Add(totalQuoteNeeded, partialQuote)
			remainingBase.SetInt64(0)
		}
	}

	if remainingBase.Sign() > 0 {
		return nil, 0, ErrInsufficientLiquidity
	}

	return totalQuoteNeeded, executedLevels, nil
}

func (s *PoolSimulator) calculateBaseTokenNeeded(quoteAmountWanted *big.Int) (*big.Int, int, error) {
	if len(s.Extra.OrderBook.Bids) == 0 {
		return nil, 0, ErrEmptyOrderBook
	}

	remainingQuote := new(big.Int).Set(quoteAmountWanted)
	totalBaseNeeded := new(big.Int)
	executedLevels := 0

	for _, bid := range s.Extra.OrderBook.Bids {
		if remainingQuote.Sign() <= 0 {
			break
		}
		executedLevels++

		maxQuoteFromLevel := new(big.Int).Mul(bid.Quantity, bid.Price)
		maxQuoteFromLevel.Div(maxQuoteFromLevel, PricePrecisionMultiplier)

		if remainingQuote.Cmp(maxQuoteFromLevel) >= 0 {
			totalBaseNeeded.Add(totalBaseNeeded, bid.Quantity)
			remainingQuote.Sub(remainingQuote, maxQuoteFromLevel)
		} else {
			partialBase := new(big.Int).Mul(remainingQuote, PricePrecisionMultiplier)
			partialBase.Add(partialBase, new(big.Int).Sub(bid.Price, One))
			partialBase.Div(partialBase, bid.Price)
			totalBaseNeeded.Add(totalBaseNeeded, partialBase)
			remainingQuote.SetInt64(0)
		}
	}

	if remainingQuote.Sign() > 0 {
		return nil, 0, ErrInsufficientLiquidity
	}

	return totalBaseNeeded, executedLevels, nil
}
