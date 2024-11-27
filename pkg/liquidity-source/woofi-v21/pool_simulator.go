package woofiv21

import (
	"errors"
	"fmt"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var (
	ErrInvalidAmountIn = errors.New("invalid amountIn")

	ErrBaseTokenIsQuoteToken       = errors.New("WooPPV2: baseToken==quoteToken")
	ErrOracleIsNotFeasible         = errors.New("WooPPV2: !ORACLE_FEASIBLE")
	ErrOraclePriceNotPositive      = errors.New("WooPPV2: !ORACLE_PRICE")
	ErrGammaExceedsLimit           = errors.New("WooPPV2: !gamma")
	ErrNotionalSwapExceedsLimit    = errors.New("WooPPV2: !maxNotionalValue")
	ErrArithmeticOverflowUnderflow = errors.New("arithmetic overflow / underflow")
)

var (
	Number_1e5 = number.TenPow(5)
)

type (
	PoolSimulator struct {
		poolpkg.Pool
		quoteToken string
		tokenInfos map[string]TokenInfo
		decimals   map[string]uint8
		wooracle   Wooracle
		cloracle   map[string]Cloracle

		gas Gas
	}

	// DecimalInfo
	// https://github.com/woonetwork/WooPoolV2/blob/e4fc06d357e5f14421c798bf57a251f865b26578/contracts/WooPPV2.sol#L58
	DecimalInfo struct {
		priceDec *uint256.Int // 10**(price_decimal)
		quoteDec *uint256.Int // 10**(quote_decimal)
		baseDec  *uint256.Int // 10**(base_decimal)
	}

	woofiV2SwapInfo struct {
		newPrice           *uint256.Int
		newMaxNotionalSwap *uint256.Int
		newMaxGamma        *uint256.Int
		base2              *woofiV2SwapInfo
	}

	Gas struct {
		Swap int64
	}
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var tokens = make([]string, len(entityPool.Tokens))
	var decimals = make(map[string]uint8)

	for i, token := range entityPool.Tokens {
		tokens[i] = token.Address
		decimals[token.Address] = token.Decimals
	}

	return &PoolSimulator{
		Pool: poolpkg.Pool{
			Info: poolpkg.PoolInfo{
				Address:  entityPool.Address,
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   tokens,
				Checked:  false,
			},
		},
		quoteToken: extra.QuoteToken,
		tokenInfos: extra.TokenInfos,
		decimals:   decimals,
		wooracle:   extra.Wooracle,
		cloracle:   extra.Cloracle,

		gas: DefaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	tokenAmountIn := params.TokenAmountIn
	tokenOut := params.TokenOut
	tokenInIndex := s.GetTokenIndex(tokenAmountIn.Token)
	tokenOutIndex := s.GetTokenIndex(tokenOut)

	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return &poolpkg.CalcAmountOutResult{}, fmt.Errorf("TokenInIndex: %v or TokenOutIndex: %v is not correct",
			tokenInIndex, tokenOutIndex)
	}

	var (
		amountOut, swapFee *uint256.Int
		swapInfo           *woofiV2SwapInfo
		err                error
	)

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}
	if tokenAmountIn.Token == s.quoteToken {
		amountOut, swapFee, swapInfo, err = s._sellQuote(tokenOut, amountIn)
		if err != nil {
			return &poolpkg.CalcAmountOutResult{}, err
		}
	} else if tokenOut == s.quoteToken {
		amountOut, swapFee, swapInfo, err = s._sellBase(tokenAmountIn.Token, amountIn)
		if err != nil {
			return &poolpkg.CalcAmountOutResult{}, err
		}
	} else {
		amountOut, swapFee, swapInfo, err = s._swapBaseToBase(tokenAmountIn.Token, tokenOut, amountIn)
		if err != nil {
			return &poolpkg.CalcAmountOutResult{}, err
		}
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &poolpkg.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: swapFee.ToBig(),
		},
		Gas:      s.gas.Swap,
		SwapInfo: swapInfo,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	_, ok := params.SwapInfo.(*woofiV2SwapInfo)
	if !ok {
		logger.Error("failed to UpdateBalancer for WooFiV2 pool, wrong swapInfo type")
		return
	}

	if params.TokenAmountIn.Token == s.quoteToken {
		s.updateBalanceSellQuote(params)
	} else if params.TokenAmountOut.Token == s.quoteToken {
		s.updateBalanceSellBase(params)
	} else {
		s.updateBalanceSwapBaseToBase(params)
	}
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} { return nil }

// _sellQuote
// https://arbiscan.io/address/0xCf4EA1688bc23DD93D933edA535F8B72FC8934Ec#code#F1#L479
func (s *PoolSimulator) _sellQuote(
	baseToken string,
	quoteAmount *uint256.Int,
) (*uint256.Int, *uint256.Int, *woofiV2SwapInfo, error) {
	if baseToken == s.quoteToken {
		return nil, nil, nil, ErrBaseTokenIsQuoteToken
	}

	swapFee := new(uint256.Int)
	swapFee = swapFee.Div(
		swapFee.Mul(
			quoteAmount,
			swapFee.SetUint64(uint64(s.tokenInfos[baseToken].FeeRate)),
		),
		Number_1e5,
	)

	quoteAmount = quoteAmount.Sub(quoteAmount, swapFee)

	state := s._wooracleV2State(baseToken)

	baseAmount, swapInfo, err := s._calcBaseAmountSellQuote(baseToken, quoteAmount, state)
	if err != nil {
		return nil, nil, nil, err
	}

	// tokenInfos[baseToken].reserve = uint192(tokenInfos[baseToken].reserve - baseAmount);
	if s.tokenInfos[baseToken].Reserve.Lt(baseAmount) {
		return nil, nil, nil, ErrArithmeticOverflowUnderflow
	}

	return baseAmount, swapFee, swapInfo, nil
}

// _sellBase
// https://arbiscan.io/address/0xCf4EA1688bc23DD93D933edA535F8B72FC8934Ec#code#F1#L432
func (s *PoolSimulator) _sellBase(
	baseToken string,
	baseAmount *uint256.Int,
) (*uint256.Int, *uint256.Int, *woofiV2SwapInfo, error) {
	if baseToken == s.quoteToken {
		return nil, nil, nil, ErrBaseTokenIsQuoteToken
	}

	state := s._wooracleV2State(baseToken)

	quoteAmount, swapInfo, err := s._calcQuoteAmountSellBase(baseToken, baseAmount, state)
	if err != nil {
		return nil, nil, nil, err
	}

	swapFee := new(uint256.Int)
	swapFee = swapFee.Div(
		swapFee.Mul(
			quoteAmount,
			swapFee.SetUint64(uint64(s.tokenInfos[baseToken].FeeRate)),
		),
		Number_1e5,
	)

	quoteAmount = quoteAmount.Sub(quoteAmount, swapFee)

	// tokenInfos[quoteToken].reserve = uint192(tokenInfos[quoteToken].reserve - quoteAmount - swapFee);
	if s.tokenInfos[s.quoteToken].Reserve.Lt(new(uint256.Int).Add(quoteAmount, swapFee)) {
		return nil, nil, nil, ErrArithmeticOverflowUnderflow
	}

	return quoteAmount, swapFee, swapInfo, nil
}

// _swapBaseToBase
// https://arbiscan.io/address/0xCf4EA1688bc23DD93D933edA535F8B72FC8934Ec#code#F1#525
func (s *PoolSimulator) _swapBaseToBase(
	baseToken1 string,
	baseToken2 string,
	base1Amount *uint256.Int,
) (*uint256.Int, *uint256.Int, *woofiV2SwapInfo, error) {
	state1 := s._wooracleV2State(baseToken1)
	state2 := s._wooracleV2State(baseToken2)

	var spread uint64
	if state1.Spread > state2.Spread {
		spread = state1.Spread / 2
	} else {
		spread = state2.Spread / 2
	}

	var feeRate uint16
	if s.tokenInfos[baseToken1].FeeRate > s.tokenInfos[baseToken2].FeeRate {
		feeRate = s.tokenInfos[baseToken1].FeeRate
	} else {
		feeRate = s.tokenInfos[baseToken2].FeeRate
	}

	state1.Spread, state2.Spread = spread, spread

	quoteAmount, swapInfo, err := s._calcQuoteAmountSellBase(baseToken1, base1Amount, state1)
	if err != nil {
		return nil, nil, nil, err
	}

	swapFee := new(uint256.Int)
	swapFee = swapFee.Div(
		swapFee.Mul(
			quoteAmount,
			swapFee.SetUint64(uint64(feeRate)),
		),
		Number_1e5,
	)

	quoteAmount = quoteAmount.Sub(quoteAmount, swapFee)

	// tokenInfos[quoteToken].reserve = uint192(tokenInfos[quoteToken].reserve - swapFee);
	if s.tokenInfos[s.quoteToken].Reserve.Lt(swapFee) {
		return nil, nil, nil, ErrArithmeticOverflowUnderflow
	}

	base2Amount, base2SwapInfo, err := s._calcBaseAmountSellQuote(baseToken2, quoteAmount, state2)
	if err != nil {
		return nil, nil, nil, err
	}

	// tokenInfos[baseToken2].reserve = uint192(tokenInfos[baseToken2].reserve - base2Amount);
	if s.tokenInfos[baseToken2].Reserve.Lt(base2Amount) {
		return nil, nil, nil, ErrArithmeticOverflowUnderflow
	}

	swapInfo.base2 = base2SwapInfo
	return base2Amount, swapFee, swapInfo, nil
}

// _calcBaseAmountSellQuote
// https://arbiscan.io/address/0xCf4EA1688bc23DD93D933edA535F8B72FC8934Ec#code#F1#L635
func (s *PoolSimulator) _calcBaseAmountSellQuote(
	baseToken string,
	quoteAmount *uint256.Int,
	state State,
) (*uint256.Int, *woofiV2SwapInfo, error) {
	if !state.WoFeasible {
		return nil, nil, ErrOracleIsNotFeasible
	}
	if state.Price.Sign() <= 0 {
		return nil, nil, ErrOraclePriceNotPositive
	}

	decs := s.decimalInfo(baseToken)
	tokenInfoBase := s.tokenInfos[baseToken]

	maxNotionalSwap := tokenInfoBase.MaxNotionalSwap
	if maxNotionalSwap == nil || quoteAmount.Cmp(maxNotionalSwap) > 0 {
		return nil, nil, ErrNotionalSwapExceedsLimit
	}

	// gamma = k * quote_amount; and decimal 18
	var gamma uint256.Int
	gamma.Div(
		gamma.Mul(quoteAmount, gamma.SetUint64(state.Coeff)),
		decs.quoteDec,
	)

	maxGamma := tokenInfoBase.MaxGamma
	if maxGamma == nil || gamma.Cmp(maxGamma) > 0 {
		return nil, nil, ErrGammaExceedsLimit
	}

	// baseAmount = quoteAmount / oracle.price * (1 - oracle.k * quoteAmount - oracle.spread)
	var num, deno uint256.Int
	baseAmount := num.Div(
		num.Div(
			num.Mul(
				num.Mul(
					quoteAmount,
					num.Mul(decs.baseDec, decs.priceDec),
				),
				deno.Sub(
					deno.Sub(number.Number_1e18, &gamma),
					uint256.NewInt(state.Spread),
				),
			),
			state.Price,
		),
		deno.Mul(number.Number_1e18, decs.quoteDec),
	)

	// new_price = oracle.price / (1 - k * quoteAmount)
	newPrice := new(uint256.Int)
	newPrice = newPrice.Div(
		newPrice.Mul(number.Number_1e18, state.Price),
		deno.Sub(number.Number_1e18, &gamma),
	)

	return baseAmount, &woofiV2SwapInfo{
		newPrice:           newPrice,
		newMaxNotionalSwap: new(uint256.Int).Sub(maxNotionalSwap, quoteAmount),
		newMaxGamma:        new(uint256.Int).Sub(maxGamma, &gamma),
	}, nil
}

// _calcQuoteAmountSellBase
// https://arbiscan.io/address/0xCf4EA1688bc23DD93D933edA535F8B72FC8934Ec#code#F1#L604
func (s *PoolSimulator) _calcQuoteAmountSellBase(
	baseToken string,
	baseAmount *uint256.Int,
	state State,
) (*uint256.Int, *woofiV2SwapInfo, error) {
	if !state.WoFeasible {
		return nil, nil, ErrOracleIsNotFeasible
	}
	if state.Price.Cmp(uint256.NewInt(0)) <= 0 {
		return nil, nil, ErrOraclePriceNotPositive
	}

	decs := s.decimalInfo(baseToken)
	tokenInfoBase := s.tokenInfos[baseToken]

	var notionalSwap, deno uint256.Int
	notionalSwap.Div(
		notionalSwap.Mul(
			notionalSwap.Mul(baseAmount, state.Price),
			decs.quoteDec,
		),
		deno.Mul(decs.baseDec, decs.priceDec),
	)

	maxNotionalSwap := tokenInfoBase.MaxNotionalSwap
	if maxNotionalSwap == nil || notionalSwap.Cmp(maxNotionalSwap) > 0 {
		return nil, nil, ErrNotionalSwapExceedsLimit
	}

	// gamma = k * price * base_amount; and decimal 18
	var gamma uint256.Int
	gamma.Div(
		gamma.Mul(
			gamma.Mul(gamma.SetUint64(state.Coeff), state.Price),
			baseAmount,
		),
		&deno,
	)

	maxGamma := tokenInfoBase.MaxGamma
	if maxGamma == nil || gamma.Cmp(maxGamma) > 0 {
		return nil, nil, ErrGammaExceedsLimit
	}

	// quoteAmount = baseAmount * oracle.price * (1 - oracle.k * baseAmount * oracle.price - oracle.spread)
	quoteAmount := new(uint256.Int)
	quoteAmount = quoteAmount.Div(
		quoteAmount.Div(
			quoteAmount.Mul(
				quoteAmount.Div(
					quoteAmount.Mul(
						quoteAmount.Mul(baseAmount, state.Price),
						decs.quoteDec,
					),
					decs.priceDec,
				),
				deno.Sub(
					deno.Sub(number.Number_1e18, &gamma),
					uint256.NewInt(state.Spread),
				),
			),
			number.Number_1e18,
		),
		decs.baseDec,
	)

	// newPrice = oracle.price * (1 - k * oracle.price * baseAmount)
	newPrice := new(uint256.Int)
	newPrice = newPrice.Div(
		newPrice.Mul(
			newPrice.Sub(number.Number_1e18, &gamma),
			state.Price,
		),
		number.Number_1e18,
	)

	return quoteAmount, &woofiV2SwapInfo{
		newPrice:           newPrice,
		newMaxNotionalSwap: new(uint256.Int).Sub(maxNotionalSwap, &notionalSwap),
		newMaxGamma:        new(uint256.Int).Sub(maxGamma, &gamma),
	}, nil
}

// decimalInfo
// https://arbiscan.io/address/0xCf4EA1688bc23DD93D933edA535F8B72FC8934Ec#code#F1#L181
func (s *PoolSimulator) decimalInfo(baseToken string) DecimalInfo {
	return DecimalInfo{
		priceDec: number.TenPow(s.wooracle.Decimals[baseToken]), // 8
		quoteDec: number.TenPow(s.decimals[s.quoteToken]),       // 18 or 6
		baseDec:  number.TenPow(s.decimals[baseToken]),          // 18 or 8
	}
}

// WooracleV2.state
// https://arbiscan.io/address/0xCf4EA1688bc23DD93D933edA535F8B72FC8934Ec#code#F1#L325
func (s *PoolSimulator) _wooracleV2State(base string) State {
	info := s.wooracle.States[base]
	basePrice, feasible := s._wooracleV2Price(base)
	return State{
		Price:      basePrice,
		Spread:     info.Spread,
		Coeff:      info.Coeff,
		WoFeasible: feasible,
	}
}

// WooracleV2.price
// https://arbiscan.io/address/0xCf4EA1688bc23DD93D933edA535F8B72FC8934Ec#code#F1#L272
func (s *PoolSimulator) _wooracleV2Price(base string) (*uint256.Int, bool) {
	woPrice := s.wooracle.States[base].Price

	cloPrice, _ := s._wooracleCloPriceInQuote(base, s.quoteToken)

	woFeasible := woPrice.Sign() != 0 && time.Now().Unix() <= s.wooracle.Timestamp+s.wooracle.StaleDuration

	bound := uint256.NewInt(s.wooracle.Bound)
	priceLowerBound := new(uint256.Int)
	priceLowerBound = priceLowerBound.Div(
		priceLowerBound.Mul(
			cloPrice,
			priceLowerBound.Sub(number.Number_1e18, bound),
		),
		number.Number_1e18,
	)
	priceUpperBound := new(uint256.Int)
	priceUpperBound = priceUpperBound.Div(
		priceUpperBound.Mul(
			cloPrice,
			priceUpperBound.Add(number.Number_1e18, bound),
		),
		number.Number_1e18,
	)
	woPriceInbound := cloPrice.Sign() == 0 || (priceLowerBound.Cmp(woPrice) <= 0 && woPrice.Cmp(priceUpperBound) <= 0)

	if woFeasible {
		return woPrice, woPriceInbound
	}

	priceOut := cloPrice
	if !s.cloracle[base].CloPreferred {
		priceOut = number.Zero
	}

	return priceOut, priceOut.Sign() != 0
}

// WooracleV2._cloPriceInQuote
// https://arbiscan.io/address/0xCf4EA1688bc23DD93D933edA535F8B72FC8934Ec#code#F1#L391
func (s *PoolSimulator) _wooracleCloPriceInQuote(fromToken string, toToken string) (*uint256.Int, int64) {
	if v, ok := s.cloracle[fromToken]; !ok || v.OracleAddress.Cmp(zeroAddress) == 0 {
		return number.Zero, 0
	}

	quoteDecimal := uint64(s.wooracle.Decimals[toToken])

	baseRefPrice := s.cloracle[fromToken].Answer
	baseUpdatedAt := s.cloracle[fromToken].UpdatedAt

	quoteRefPrice := s.cloracle[toToken].Answer
	quoteUpdatedAt := s.cloracle[toToken].UpdatedAt

	ceoff := new(uint256.Int).Exp(number.Number_10, uint256.NewInt(quoteDecimal))

	refPrice := new(uint256.Int)
	refPrice = refPrice.Div(
		refPrice.Mul(baseRefPrice, ceoff),
		quoteRefPrice,
	)
	refTimestamp := quoteUpdatedAt
	if baseUpdatedAt.Lt(quoteUpdatedAt) {
		refTimestamp = baseUpdatedAt
	}

	return refPrice, int64(refTimestamp.Uint64())
}

func (s *PoolSimulator) updateBalanceSellQuote(params poolpkg.UpdateBalanceParams) {
	swapInfo := params.SwapInfo.(*woofiV2SwapInfo)
	amountIn, _ := uint256.FromBig(params.TokenAmountIn.Amount)
	amountOut, _ := uint256.FromBig(params.TokenAmountOut.Amount)
	swapFee, _ := uint256.FromBig(params.Fee.Amount)
	tokenInfoIn, tokenInfoOut := s.tokenInfos[params.TokenAmountIn.Token], s.tokenInfos[params.TokenAmountOut.Token]

	newQuoteReserve := amountIn.Add(
		tokenInfoIn.Reserve,
		amountIn.Sub(amountIn, swapFee),
	)
	newBaseReserves := amountOut.Sub(
		tokenInfoOut.Reserve,
		amountOut,
	)

	s.tokenInfos[params.TokenAmountIn.Token] = TokenInfo{
		Reserve:         newQuoteReserve,
		FeeRate:         tokenInfoIn.FeeRate,
		MaxGamma:        tokenInfoIn.MaxGamma,
		MaxNotionalSwap: tokenInfoIn.MaxNotionalSwap,
	}
	s.tokenInfos[params.TokenAmountOut.Token] = TokenInfo{
		Reserve:         newBaseReserves,
		FeeRate:         tokenInfoOut.FeeRate,
		MaxGamma:        swapInfo.newMaxGamma,
		MaxNotionalSwap: swapInfo.newMaxNotionalSwap,
	}
	stateIn := s.wooracle.States[params.TokenAmountIn.Token]
	s.wooracle.States[params.TokenAmountIn.Token] = State{
		Price:      swapInfo.newPrice,
		Spread:     stateIn.Spread,
		Coeff:      stateIn.Coeff,
		WoFeasible: stateIn.WoFeasible,
	}
}

func (s *PoolSimulator) updateBalanceSellBase(params poolpkg.UpdateBalanceParams) {
	swapInfo := params.SwapInfo.(*woofiV2SwapInfo)
	amountIn, _ := uint256.FromBig(params.TokenAmountIn.Amount)
	amountOut, _ := uint256.FromBig(params.TokenAmountOut.Amount)
	swapFee, _ := uint256.FromBig(params.Fee.Amount)
	tokenInfoIn, tokenInfoOut := s.tokenInfos[params.TokenAmountIn.Token], s.tokenInfos[params.TokenAmountOut.Token]

	newBaseReserves := amountIn.Add(
		tokenInfoIn.Reserve,
		amountIn,
	)
	newQuoteReserve := swapFee.Sub(
		amountOut.Sub(
			tokenInfoOut.Reserve,
			amountOut,
		),
		swapFee,
	)

	s.tokenInfos[params.TokenAmountIn.Token] = TokenInfo{
		Reserve:         newBaseReserves,
		FeeRate:         tokenInfoIn.FeeRate,
		MaxGamma:        swapInfo.newMaxGamma,
		MaxNotionalSwap: swapInfo.newMaxNotionalSwap,
	}
	s.tokenInfos[params.TokenAmountOut.Token] = TokenInfo{
		Reserve:         newQuoteReserve,
		FeeRate:         tokenInfoOut.FeeRate,
		MaxGamma:        tokenInfoOut.MaxGamma,
		MaxNotionalSwap: tokenInfoOut.MaxNotionalSwap,
	}
	stateIn := s.wooracle.States[params.TokenAmountIn.Token]
	s.wooracle.States[params.TokenAmountIn.Token] = State{
		Price:      swapInfo.newPrice,
		Spread:     stateIn.Spread,
		Coeff:      stateIn.Coeff,
		WoFeasible: stateIn.WoFeasible,
	}
}

func (s *PoolSimulator) updateBalanceSwapBaseToBase(params poolpkg.UpdateBalanceParams) {
	swapInfo := params.SwapInfo.(*woofiV2SwapInfo)
	amountIn, _ := uint256.FromBig(params.TokenAmountIn.Amount)
	amountOut, _ := uint256.FromBig(params.TokenAmountOut.Amount)
	swapFee, _ := uint256.FromBig(params.Fee.Amount)
	tokenInfoIn, tokenInfoOut := s.tokenInfos[params.TokenAmountIn.Token], s.tokenInfos[params.TokenAmountOut.Token]
	tokenInfoQuote := s.tokenInfos[s.quoteToken]

	newBase1Reserves := amountIn.Add(tokenInfoIn.Reserve, amountIn)
	newBase2Reserves := amountOut.Sub(tokenInfoOut.Reserve, amountOut)
	newQuoteReserve := swapFee.Sub(tokenInfoQuote.Reserve, swapFee)

	s.tokenInfos[params.TokenAmountIn.Token] = TokenInfo{
		Reserve:         newBase1Reserves,
		FeeRate:         tokenInfoIn.FeeRate,
		MaxGamma:        swapInfo.newMaxGamma,
		MaxNotionalSwap: swapInfo.newMaxNotionalSwap,
	}
	s.tokenInfos[params.TokenAmountOut.Token] = TokenInfo{
		Reserve:         newBase2Reserves,
		FeeRate:         tokenInfoOut.FeeRate,
		MaxGamma:        swapInfo.base2.newMaxGamma,
		MaxNotionalSwap: swapInfo.base2.newMaxNotionalSwap,
	}
	s.tokenInfos[s.quoteToken] = TokenInfo{
		Reserve:         newQuoteReserve,
		FeeRate:         tokenInfoQuote.FeeRate,
		MaxGamma:        tokenInfoQuote.MaxGamma,
		MaxNotionalSwap: tokenInfoQuote.MaxNotionalSwap,
	}
	stateIn, stateOut := s.wooracle.States[params.TokenAmountIn.Token], s.wooracle.States[params.TokenAmountOut.Token]
	s.wooracle.States[params.TokenAmountIn.Token] = State{
		Price:      swapInfo.newPrice,
		Spread:     stateIn.Spread,
		Coeff:      stateIn.Coeff,
		WoFeasible: stateIn.WoFeasible,
	}
	s.wooracle.States[params.TokenAmountOut.Token] = State{
		Price:      swapInfo.base2.newPrice,
		Spread:     stateOut.Spread,
		Coeff:      stateOut.Coeff,
		WoFeasible: stateOut.WoFeasible,
	}
}
