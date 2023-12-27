package woofiv2

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
	"github.com/holiman/uint256"
)

var (
	ErrInvalidAmountIn = errors.New("invalid amountIn")

	ErrBaseTokenIsQuoteToken = errors.New("WooPPV2: baseToken==quoteToken")
	ErrOracleIsNotFeasible   = errors.New("WooPPV2: !ORACLE_FEASIBLE")

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
		newPrice      *uint256.Int
		newBase1Price *uint256.Int
		newBase2Price *uint256.Int
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
		return &poolpkg.CalcAmountOutResult{}, fmt.Errorf("TokenInIndex: %v or TokenOutIndex: %v is not correct", tokenInIndex, tokenOutIndex)
	}

	var (
		amountOut, swapFee *uint256.Int
		swapInfo           woofiV2SwapInfo
		err                error
	)

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}
	if tokenAmountIn.Token == s.quoteToken {
		var newPrice *uint256.Int
		amountOut, swapFee, newPrice, err = s._sellQuote(tokenOut, amountIn)
		if err != nil {
			return &poolpkg.CalcAmountOutResult{}, err
		}

		swapInfo = woofiV2SwapInfo{
			newPrice: newPrice,
		}
	} else if tokenOut == s.quoteToken {
		var newPrice *uint256.Int
		amountOut, swapFee, newPrice, err = s._sellBase(tokenAmountIn.Token, amountIn)
		if err != nil {
			return &poolpkg.CalcAmountOutResult{}, err
		}

		swapInfo = woofiV2SwapInfo{
			newPrice: newPrice,
		}
	} else {
		var newBase1Price, newBase2Price *uint256.Int
		amountOut, swapFee, newBase1Price, newBase2Price, err = s._swapBaseToBase(tokenAmountIn.Token, tokenOut, amountIn)
		if err != nil {
			return &poolpkg.CalcAmountOutResult{}, err
		}

		swapInfo = woofiV2SwapInfo{
			newBase1Price: newBase1Price,
			newBase2Price: newBase2Price,
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
	_, ok := params.SwapInfo.(woofiV2SwapInfo)
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

// _sellBase
// https://github.com/woonetwork/WooPoolV2/blob/e4fc06d357e5f14421c798bf57a251f865b26578/contracts/WooPPV2.sol#L361
func (s *PoolSimulator) _sellBase(
	baseToken string,
	baseAmount *uint256.Int,
) (*uint256.Int, *uint256.Int, *uint256.Int, error) {
	if baseToken == s.quoteToken {
		return nil, nil, nil, ErrBaseTokenIsQuoteToken
	}

	state := s._wooracleV2State(baseToken)

	quoteAmount, newPrice, err := s._calcQuoteAmountSellBase(baseToken, baseAmount, state)
	if err != nil {
		return nil, nil, nil, err
	}

	swapFee := new(uint256.Int).Div(
		new(uint256.Int).Mul(
			quoteAmount,
			uint256.NewInt(uint64(s.tokenInfos[baseToken].FeeRate)),
		),
		Number_1e5,
	)

	quoteAmount = new(uint256.Int).Sub(quoteAmount, swapFee)

	// tokenInfos[quoteToken].reserve = uint192(tokenInfos[quoteToken].reserve - quoteAmount - swapFee);
	if s.tokenInfos[s.quoteToken].Reserve.Lt(new(uint256.Int).Add(quoteAmount, swapFee)) {
		return nil, nil, nil, ErrArithmeticOverflowUnderflow
	}

	return quoteAmount, swapFee, newPrice, nil
}

// _sellQuote
// https://github.com/woonetwork/WooPoolV2/blob/e4fc06d357e5f14421c798bf57a251f865b26578/contracts/WooPPV2.sol#L408
func (s *PoolSimulator) _sellQuote(
	baseToken string,
	quoteAmount *uint256.Int,
) (*uint256.Int, *uint256.Int, *uint256.Int, error) {
	if baseToken == s.quoteToken {
		return nil, nil, nil, ErrBaseTokenIsQuoteToken
	}

	swapFee := new(uint256.Int).Div(
		new(uint256.Int).Mul(
			quoteAmount,
			uint256.NewInt(uint64(s.tokenInfos[baseToken].FeeRate)),
		),
		Number_1e5,
	)

	quoteAmount = new(uint256.Int).Sub(quoteAmount, swapFee)

	state := s._wooracleV2State(baseToken)

	baseAmount, newPrice, err := s._calcBaseAmountSellQuote(baseToken, quoteAmount, state)
	if err != nil {
		return nil, nil, nil, err
	}

	// tokenInfos[baseToken].reserve = uint192(tokenInfos[baseToken].reserve - baseAmount);
	if s.tokenInfos[baseToken].Reserve.Lt(baseAmount) {
		return nil, nil, nil, ErrArithmeticOverflowUnderflow
	}

	return baseAmount, swapFee, newPrice, nil
}

// _swapBaseToBase
// https://github.com/woonetwork/WooPoolV2/blob/e4fc06d357e5f14421c798bf57a251f865b26578/contracts/WooPPV2.sol#L457
func (s *PoolSimulator) _swapBaseToBase(
	baseToken1 string,
	baseToken2 string,
	base1Amount *uint256.Int,
) (*uint256.Int, *uint256.Int, *uint256.Int, *uint256.Int, error) {
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

	quoteAmount, newBase1Price, err := s._calcQuoteAmountSellBase(baseToken1, base1Amount, state1)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	swapFee := new(uint256.Int).Div(new(uint256.Int).Mul(quoteAmount, uint256.NewInt(uint64(feeRate))), Number_1e5)

	quoteAmount = new(uint256.Int).Sub(quoteAmount, swapFee)

	// tokenInfos[quoteToken].reserve = uint192(tokenInfos[quoteToken].reserve - swapFee);
	if s.tokenInfos[s.quoteToken].Reserve.Lt(swapFee) {
		return nil, nil, nil, nil, ErrArithmeticOverflowUnderflow
	}

	base2Amount, newBase2Price, err := s._calcBaseAmountSellQuote(baseToken2, quoteAmount, state2)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// tokenInfos[baseToken2].reserve = uint192(tokenInfos[baseToken2].reserve - base2Amount);
	if s.tokenInfos[baseToken2].Reserve.Lt(base2Amount) {
		return nil, nil, nil, nil, ErrArithmeticOverflowUnderflow
	}

	return base2Amount, swapFee, newBase1Price, newBase2Price, nil
}

// _calcBaseAmountSellQuote
// https://github.com/woonetwork/WooPoolV2/blob/e4fc06d357e5f14421c798bf57a251f865b26578/contracts/WooPPV2.sol#L559
func (s *PoolSimulator) _calcBaseAmountSellQuote(
	baseToken string,
	quoteAmount *uint256.Int,
	state State,
) (*uint256.Int, *uint256.Int, error) {
	if !state.WoFeasible {
		return nil, nil, ErrOracleIsNotFeasible
	}

	desc := s.decimalInfo(baseToken)

	coef := new(uint256.Int).Sub(
		new(uint256.Int).Sub(
			number.Number_1e18,
			new(uint256.Int).Div(
				new(uint256.Int).Mul(quoteAmount, uint256.NewInt(state.Coeff)),
				desc.quoteDec,
			),
		),
		uint256.NewInt(state.Spread),
	)

	baseAmount := new(uint256.Int).Div(
		new(uint256.Int).Div(
			new(uint256.Int).Mul(
				new(uint256.Int).Div(
					new(uint256.Int).Mul(
						new(uint256.Int).Mul(
							quoteAmount,
							desc.baseDec,
						),
						desc.priceDec,
					),
					state.Price,
				),
				coef,
			),
			number.Number_1e18,
		),
		desc.quoteDec,
	)

	newPrice := new(uint256.Int).Div(
		new(uint256.Int).Div(
			new(uint256.Int).Mul(
				new(uint256.Int).Add(
					new(uint256.Int).Mul(number.Number_1e18, desc.quoteDec),
					new(uint256.Int).Mul(number.Number_2, new(uint256.Int).Mul(uint256.NewInt(state.Coeff), quoteAmount)),
				),
				state.Price,
			),
			desc.quoteDec,
		),
		number.Number_1e18,
	)

	return baseAmount, newPrice, nil
}

// _calcQuoteAmountSellBase
// https://github.com/woonetwork/WooPoolV2/blob/e4fc06d357e5f14421c798bf57a251f865b26578/contracts/WooPPV2.sol#L559
func (s *PoolSimulator) _calcQuoteAmountSellBase(
	baseToken string,
	baseAmount *uint256.Int,
	state State,
) (*uint256.Int, *uint256.Int, error) {
	if !state.WoFeasible {
		return nil, nil, ErrOracleIsNotFeasible
	}

	decs := s.decimalInfo(baseToken)

	coef := new(uint256.Int).Sub(
		new(uint256.Int).Sub(
			number.Number_1e18,
			new(uint256.Int).Div(
				new(uint256.Int).Div(
					new(uint256.Int).Mul(uint256.NewInt(state.Coeff), new(uint256.Int).Mul(baseAmount, state.Price)),
					decs.baseDec,
				),
				decs.priceDec,
			),
		),
		uint256.NewInt(state.Spread),
	)

	quoteAmount := new(uint256.Int).Div(
		new(uint256.Int).Div(
			new(uint256.Int).Mul(
				new(uint256.Int).Div(
					new(uint256.Int).Mul(baseAmount, new(uint256.Int).Mul(decs.quoteDec, state.Price)),
					decs.priceDec,
				),
				coef,
			),
			number.Number_1e18,
		),
		decs.baseDec,
	)

	newPrice := new(uint256.Int).Div(
		new(uint256.Int).Mul(
			new(uint256.Int).Sub(
				number.Number_1e18,
				new(uint256.Int).Div(
					new(uint256.Int).Div(
						new(uint256.Int).Mul(
							new(uint256.Int).Mul(number.Number_2, uint256.NewInt(state.Coeff)),
							new(uint256.Int).Mul(state.Price, baseAmount),
						),
						decs.priceDec,
					),
					decs.baseDec,
				),
			),
			state.Price,
		),
		number.Number_1e18,
	)

	return quoteAmount, newPrice, nil
}

// decimalInfo
// https://github.com/woonetwork/WooPoolV2/blob/e4fc06d357e5f14421c798bf57a251f865b26578/contracts/WooPPV2.sol#L181
func (s *PoolSimulator) decimalInfo(baseToken string) DecimalInfo {
	return DecimalInfo{
		priceDec: number.TenPow(s.wooracle.Decimals[baseToken]), // 8
		quoteDec: number.TenPow(s.decimals[s.quoteToken]),       // 18 or 6
		baseDec:  number.TenPow(s.decimals[baseToken]),          // 18 or 8
	}
}

func (s *PoolSimulator) updateBalanceSellBase(params poolpkg.UpdateBalanceParams) {
	swapInfo := params.SwapInfo.(woofiV2SwapInfo)
	amountIn, _ := uint256.FromBig(params.TokenAmountIn.Amount)
	amountOut, _ := uint256.FromBig(params.TokenAmountOut.Amount)
	swapFee, _ := uint256.FromBig(params.Fee.Amount)

	newBaseReserves := new(uint256.Int).Add(
		s.tokenInfos[params.TokenAmountIn.Token].Reserve,
		amountIn,
	)
	newQuoteReserve := new(uint256.Int).Sub(
		new(uint256.Int).Sub(
			s.tokenInfos[params.TokenAmountOut.Token].Reserve,
			amountOut,
		),
		swapFee,
	)

	s.tokenInfos[params.TokenAmountIn.Token] = TokenInfo{
		Reserve: newBaseReserves,
		FeeRate: s.tokenInfos[params.TokenAmountIn.Token].FeeRate,
	}
	s.tokenInfos[params.TokenAmountOut.Token] = TokenInfo{
		Reserve: newQuoteReserve,
		FeeRate: s.tokenInfos[params.TokenAmountOut.Token].FeeRate,
	}
	s.wooracle.States[params.TokenAmountIn.Token] = State{
		Price:      swapInfo.newPrice,
		Spread:     s.wooracle.States[params.TokenAmountIn.Token].Spread,
		Coeff:      s.wooracle.States[params.TokenAmountIn.Token].Coeff,
		WoFeasible: s.wooracle.States[params.TokenAmountIn.Token].WoFeasible,
	}
}

// WooracleV2.state
// https://github.com/woonetwork/WooPoolV2/blob/fb94e2bf4882f51340c66357e8c566edc2a767a9/contracts/wooracle/WooracleV2.sol#L281-L285
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
// https://github.com/woonetwork/WooPoolV2/blob/fb94e2bf4882f51340c66357e8c566edc2a767a9/contracts/wooracle/WooracleV2.sol#L223-L240
func (s *PoolSimulator) _wooracleV2Price(base string) (*uint256.Int, bool) {
	woPrice := s.wooracle.States[base].Price

	cloPrice, _ := s._wooracleCloPriceInQuote(base, s.quoteToken)

	woFeasible := !woPrice.Eq(number.Zero) && time.Now().Unix() <= s.wooracle.Timestamp+s.wooracle.StaleDuration

	bound := uint256.NewInt(s.wooracle.Bound)
	priceLowerBound := new(uint256.Int).Div(
		new(uint256.Int).Mul(
			cloPrice,
			new(uint256.Int).Sub(number.Number_1e18, bound),
		),
		number.Number_1e18,
	)
	priceUpperBound := new(uint256.Int).Div(
		new(uint256.Int).Mul(
			cloPrice,
			new(uint256.Int).Add(number.Number_1e18, bound),
		),
		number.Number_1e18,
	)
	woPriceInbound := cloPrice.Eq(number.Zero) || (priceLowerBound.Cmp(woPrice) <= 0 && woPrice.Cmp(priceUpperBound) <= 0)

	if woFeasible {
		return woPrice, woPriceInbound
	}

	priceOut := cloPrice
	if !s.cloracle[base].CloPreferred {
		priceOut = number.Zero
	}

	return priceOut, !priceOut.Eq(number.Zero)
}

// WooracleV2._cloPriceInQuote
// https://github.com/woonetwork/WooPoolV2/blob/fb94e2bf4882f51340c66357e8c566edc2a767a9/contracts/wooracle/WooracleV2.sol#L304-L325
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

	refPrice := new(uint256.Int).Div(
		new(uint256.Int).Mul(baseRefPrice, ceoff),
		quoteRefPrice,
	)
	refTimestamp := quoteUpdatedAt
	if baseUpdatedAt.Lt(quoteUpdatedAt) {
		refTimestamp = baseUpdatedAt
	}

	return refPrice, int64(refTimestamp.Uint64())
}

func (s *PoolSimulator) updateBalanceSellQuote(params poolpkg.UpdateBalanceParams) {
	swapInfo := params.SwapInfo.(woofiV2SwapInfo)
	amountIn, _ := uint256.FromBig(params.TokenAmountIn.Amount)
	amountOut, _ := uint256.FromBig(params.TokenAmountOut.Amount)
	swapFee, _ := uint256.FromBig(params.Fee.Amount)

	newBaseReserves := new(uint256.Int).Sub(
		s.tokenInfos[params.TokenAmountOut.Token].Reserve,
		amountOut,
	)
	newQuoteReserve := new(uint256.Int).Add(
		s.tokenInfos[params.TokenAmountIn.Token].Reserve,
		new(uint256.Int).Sub(amountIn, swapFee),
	)

	s.tokenInfos[params.TokenAmountOut.Token] = TokenInfo{
		Reserve: newBaseReserves,
		FeeRate: s.tokenInfos[params.TokenAmountOut.Token].FeeRate,
	}
	s.tokenInfos[params.TokenAmountIn.Token] = TokenInfo{
		Reserve: newQuoteReserve,
		FeeRate: s.tokenInfos[params.TokenAmountIn.Token].FeeRate,
	}
	s.wooracle.States[params.TokenAmountIn.Token] = State{
		Price:      swapInfo.newPrice,
		Spread:     s.wooracle.States[params.TokenAmountIn.Token].Spread,
		Coeff:      s.wooracle.States[params.TokenAmountIn.Token].Coeff,
		WoFeasible: s.wooracle.States[params.TokenAmountIn.Token].WoFeasible,
	}
}

func (s *PoolSimulator) updateBalanceSwapBaseToBase(params poolpkg.UpdateBalanceParams) {
	swapInfo := params.SwapInfo.(woofiV2SwapInfo)
	amountIn, _ := uint256.FromBig(params.TokenAmountIn.Amount)
	amountOut, _ := uint256.FromBig(params.TokenAmountOut.Amount)
	swapFee, _ := uint256.FromBig(params.Fee.Amount)

	newBase1Reserves := new(uint256.Int).Add(
		s.tokenInfos[params.TokenAmountIn.Token].Reserve,
		amountIn,
	)
	newBase2Reserves := new(uint256.Int).Sub(
		s.tokenInfos[params.TokenAmountOut.Token].Reserve,
		amountOut,
	)
	newQuoteReserve := new(uint256.Int).Sub(
		s.tokenInfos[s.quoteToken].Reserve,
		swapFee,
	)

	s.tokenInfos[params.TokenAmountIn.Token] = TokenInfo{
		Reserve: newBase1Reserves,
		FeeRate: s.tokenInfos[params.TokenAmountIn.Token].FeeRate,
	}
	s.tokenInfos[params.TokenAmountOut.Token] = TokenInfo{
		Reserve: newBase2Reserves,
		FeeRate: s.tokenInfos[params.TokenAmountOut.Token].FeeRate,
	}
	s.tokenInfos[s.quoteToken] = TokenInfo{
		Reserve: newQuoteReserve,
		FeeRate: s.tokenInfos[s.quoteToken].FeeRate,
	}
	s.wooracle.States[params.TokenAmountIn.Token] = State{
		Price:      swapInfo.newBase1Price,
		Spread:     s.wooracle.States[params.TokenAmountIn.Token].Spread,
		Coeff:      s.wooracle.States[params.TokenAmountIn.Token].Coeff,
		WoFeasible: s.wooracle.States[params.TokenAmountIn.Token].WoFeasible,
	}
	s.wooracle.States[params.TokenAmountOut.Token] = State{
		Price:      swapInfo.newBase2Price,
		Spread:     s.wooracle.States[params.TokenAmountOut.Token].Spread,
		Coeff:      s.wooracle.States[params.TokenAmountOut.Token].Coeff,
		WoFeasible: s.wooracle.States[params.TokenAmountOut.Token].WoFeasible,
	}
}
