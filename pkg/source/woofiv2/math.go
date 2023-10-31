package woofiv2

import (
	"math/big"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func GetAmountOut(
	fromToken, toToken string,
	fromAmount *big.Int,
	state *WooFiV2State,
) (*big.Int, error) {
	if fromToken == state.QuoteToken {
		return sellQuote(toToken, fromAmount, state)
	} else if toToken == state.QuoteToken {
		return sellBase(fromToken, fromAmount, state)
	} else {
		return swapBaseToBase(fromToken, toToken, fromAmount, state)
	}
}

func sellQuote(
	baseToken string,
	quoteAmount *big.Int,
	state *WooFiV2State,
) (*big.Int, error) {
	if baseToken == state.QuoteToken {
		return nil, ErrBaseTokenIsQuoteToken
	}

	baseTokenInfo, ok := state.TokenInfos[baseToken]
	if !ok {
		return nil, ErrTokenInfoNotFound
	}
	quoteTokenInfo, ok := state.TokenInfos[state.QuoteToken]
	if !ok {
		return nil, ErrTokenInfoNotFound
	}

	swapFee := new(big.Int).Div(
		new(big.Int).Mul(quoteAmount, baseTokenInfo.FeeRate),
		big.NewInt(1e5),
	)
	quoteAmountAfterFee := new(big.Int).Sub(quoteAmount, swapFee)
	state.UnclaimedFee = new(big.Int).Add(state.UnclaimedFee, swapFee)

	wooracleState := getState(baseToken, state)
	baseAmount, newPrice, err := calcBaseAmountSellQuote(baseToken, quoteAmountAfterFee, wooracleState, state)
	if err != nil {
		return nil, err
	}

	if err := postPrice(baseToken, newPrice, state); err != nil {
		return nil, err
	}

	baseTokenInfo.Reserve = new(big.Int).Sub(baseTokenInfo.Reserve, baseAmount)
	quoteTokenInfo.Reserve = new(big.Int).Add(quoteTokenInfo.Reserve, quoteAmount)

	if baseTokenInfo.Reserve.Cmp(bignumber.ZeroBI) < 0 || baseAmount.Cmp(bignumber.ZeroBI) < 0 {
		return nil, ErrBaseBalanceNotEnough
	}

	return baseAmount, nil
}

func sellBase(
	baseToken string,
	baseAmount *big.Int,
	state *WooFiV2State,
) (*big.Int, error) {
	if baseToken == state.QuoteToken {
		return nil, ErrBaseTokenIsQuoteToken
	}

	baseTokenInfo, ok := state.TokenInfos[baseToken]
	if !ok {
		return nil, ErrTokenInfoNotFound
	}
	quoteTokenInfo, ok := state.TokenInfos[state.QuoteToken]
	if !ok {
		return nil, ErrTokenInfoNotFound
	}

	wooracleState := getState(baseToken, state)
	quoteAmount, newPrice, err := calcQuoteAmountSellBase(baseToken, baseAmount, wooracleState, state)
	if err != nil {
		return nil, err
	}
	if err := postPrice(baseToken, newPrice, state); err != nil {
		return nil, err
	}

	swapFee := new(big.Int).Div(
		new(big.Int).Mul(quoteAmount, baseTokenInfo.FeeRate),
		big.NewInt(1e5),
	)
	quoteAmountAfterFee := new(big.Int).Sub(quoteAmount, swapFee)
	state.UnclaimedFee = new(big.Int).Add(state.UnclaimedFee, swapFee)

	baseTokenInfo.Reserve = new(big.Int).Add(baseTokenInfo.Reserve, baseAmount)
	quoteTokenInfo.Reserve = new(big.Int).Sub(quoteTokenInfo.Reserve, new(big.Int).Sub(quoteAmount, swapFee))

	if quoteTokenInfo.Reserve.Cmp(bignumber.ZeroBI) < 0 || quoteAmount.Cmp(bignumber.ZeroBI) < 0 {
		return nil, ErrQuoteBalanceNotEnough
	}

	return quoteAmountAfterFee, nil
}

func swapBaseToBase(
	baseToken1, baseToken2 string,
	base1Amount *big.Int,
	state *WooFiV2State,
) (*big.Int, error) {
	if baseToken1 == state.QuoteToken || baseToken2 == state.QuoteToken {
		return nil, ErrBaseTokenIsQuoteToken
	}

	base1TokenInfo, ok := state.TokenInfos[baseToken1]
	if !ok {
		return nil, ErrTokenInfoNotFound
	}
	base2TokenInfo, ok := state.TokenInfos[baseToken2]
	if !ok {
		return nil, ErrTokenInfoNotFound
	}
	quoteTokenInfo, ok := state.TokenInfos[state.QuoteToken]
	if !ok {
		return nil, ErrTokenInfoNotFound
	}

	wooracleState1 := getState(baseToken1, state)
	wooracleState2 := getState(baseToken2, state)

	spread := new(big.Int).Div(
		maxBigInt(base1TokenInfo.State.Spread, base2TokenInfo.State.Spread),
		bignumber.Two,
	)
	feeRate := maxBigInt(base1TokenInfo.FeeRate, base2TokenInfo.FeeRate)

	wooracleState1.Spread = spread
	wooracleState2.Spread = spread

	quoteAmount, newBase1Price, err := calcQuoteAmountSellBase(baseToken1, base1Amount, wooracleState1, state)
	if err != nil {
		return nil, err
	}
	if err := postPrice(baseToken1, newBase1Price, state); err != nil {
		return nil, err
	}

	swapFee := new(big.Int).Div(
		new(big.Int).Mul(quoteAmount, feeRate),
		big.NewInt(1e5),
	)
	quoteAmountAfterFee := new(big.Int).Sub(quoteAmount, swapFee)
	state.UnclaimedFee = new(big.Int).Add(state.UnclaimedFee, swapFee)

	quoteTokenInfo.Reserve = new(big.Int).Sub(quoteTokenInfo.Reserve, swapFee)
	base1TokenInfo.Reserve = new(big.Int).Add(base1TokenInfo.Reserve, base1Amount)

	base2Amount, newBase2Price, err := calcBaseAmountSellQuote(baseToken2, quoteAmountAfterFee, wooracleState2, state)
	if err != nil {
		return nil, err
	}
	if err := postPrice(baseToken2, newBase2Price, state); err != nil {
		return nil, err
	}

	base2TokenInfo.Reserve = new(big.Int).Sub(base2TokenInfo.Reserve, base2Amount)

	if base2TokenInfo.Reserve.Cmp(bignumber.ZeroBI) < 0 || base2Amount.Cmp(bignumber.ZeroBI) < 0 {
		return nil, ErrBase2BalanceNotEnough
	}

	return base2Amount, nil
}

func calcBaseAmountSellQuote(
	baseToken string,
	quoteAmount *big.Int,
	wooracleState *OracleState,
	state *WooFiV2State,
) (*big.Int, *big.Int, error) {
	if !wooracleState.WoFeasible {
		return nil, nil, ErrOracleNotFeasible
	}

	desc, err := decimalInfo(baseToken, state)
	if err != nil {
		return nil, nil, err
	}

	// baseAmount = quoteAmount / oracle.price * (1 - oracle.k * quoteAmount - oracle.spread)
	coef := new(big.Int).Sub(
		new(big.Int).Sub(
			bignumber.BONE,
			new(big.Int).Div(new(big.Int).Mul(quoteAmount, wooracleState.Coeff), desc.QuoteDec),
		),
		wooracleState.Spread,
	)
	baseAmount := new(big.Int).Div(
		new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Div(
					new(big.Int).Mul(
						new(big.Int).Mul(quoteAmount, desc.BaseDec),
						desc.PriceDec,
					),
					wooracleState.Price,
				),
				coef,
			),
			bignumber.BONE,
		),
		desc.QuoteDec,
	)

	// new_price = oracle.price * (1 + 2 * k * quoteAmount)
	newPrice := new(big.Int).Div(
		new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Add(
					new(big.Int).Mul(bignumber.BONE, desc.QuoteDec),
					new(big.Int).Mul(bignumber.Two, new(big.Int).Mul(wooracleState.Coeff, quoteAmount)),
				),
				wooracleState.Price,
			),
			desc.QuoteDec,
		),
		bignumber.BONE,
	)

	return baseAmount, newPrice, nil
}

func calcQuoteAmountSellBase(
	baseToken string,
	baseAmount *big.Int,
	wooracleState *OracleState,
	state *WooFiV2State,
) (*big.Int, *big.Int, error) {
	if !wooracleState.WoFeasible {
		return nil, nil, ErrOracleNotFeasible
	}

	decs, err := decimalInfo(baseToken, state)
	if err != nil {
		return nil, nil, err
	}
	// quoteAmount = baseAmount * oracle.price * (1 - oracle.k * baseAmount * oracle.price - oracle.spread)
	coef := new(big.Int).Sub(
		new(big.Int).Sub(
			bignumber.BONE,
			new(big.Int).Div(
				new(big.Int).Div(
					new(big.Int).Mul(wooracleState.Coeff, new(big.Int).Mul(baseAmount, wooracleState.Price)),
					decs.BaseDec,
				),
				decs.PriceDec,
			),
		),
		wooracleState.Spread,
	)
	quoteAmount := new(big.Int).Div(
		new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Div(
					new(big.Int).Mul(baseAmount, new(big.Int).Mul(decs.QuoteDec, wooracleState.Price)),
					decs.PriceDec,
				),
				coef,
			),
			bignumber.BONE,
		),
		decs.BaseDec,
	)

	// newPrice = oracle.price * (1 - 2 * k * oracle.price * baseAmount)
	newPrice := new(big.Int).Div(
		new(big.Int).Sub(
			bignumber.BONE,
			new(big.Int).Mul(
				new(big.Int).Div(
					new(big.Int).Div(
						new(big.Int).Mul(
							new(big.Int).Mul(bignumber.Two, wooracleState.Coeff),
							new(big.Int).Mul(wooracleState.Price, baseAmount),
						),
						decs.PriceDec,
					),
					decs.BaseDec,
				),
				wooracleState.Price,
			),
		),
		bignumber.BONE,
	)

	return quoteAmount, newPrice, nil
}

func decimalInfo(baseToken string, state *WooFiV2State) (DecimalInfo, error) {
	baseTokenInfo, ok := state.TokenInfos[baseToken]
	if !ok {
		return DecimalInfo{}, ErrTokenInfoNotFound
	}
	quoteTokenInfo, ok := state.TokenInfos[state.QuoteToken]
	if !ok {
		return DecimalInfo{}, ErrTokenInfoNotFound
	}

	priceDec := bignumber.TenPowInt(baseTokenInfo.State.Decimals)
	quoteDec := bignumber.TenPowInt(quoteTokenInfo.Decimals)
	baseDec := bignumber.TenPowInt(baseTokenInfo.Decimals)

	return DecimalInfo{
		PriceDec: priceDec,
		QuoteDec: quoteDec,
		BaseDec:  baseDec,
	}, nil
}

func maxBigInt(a, b *big.Int) *big.Int {
	if a.Cmp(b) >= 0 {
		return a
	}
	return b
}

func postPrice(baseToken string, newPrice *big.Int, state *WooFiV2State) error {
	baseTokenInfo, ok := state.TokenInfos[baseToken]
	if !ok {
		return ErrTokenInfoNotFound
	}

	baseTokenInfo.State.Price = new(big.Int).Set(newPrice)

	return nil
}

func getState(base string, state *WooFiV2State) *OracleState {
	basePrice, feasible := getPrice(base, state)
	return &OracleState{
		Price:        basePrice,
		Spread:       state.TokenInfos[base].State.Spread,
		Coeff:        state.TokenInfos[base].State.Coeff,
		WoFeasible:   feasible,
		Decimals:     state.TokenInfos[base].State.Decimals,
		CloPrice:     state.TokenInfos[base].State.CloPrice,
		CloPreferred: state.TokenInfos[base].State.CloPreferred,
	}
}

func getPrice(base string, state *WooFiV2State) (*big.Int, bool) {
	woPrice := new(big.Int).Set(state.TokenInfos[base].State.Price)
	woPriceTimestamp := state.Timestamp
	cloPrice := new(big.Int).Set(state.TokenInfos[base].State.CloPrice)

	woFeasible := false
	if woPrice.Cmp(bignumber.ZeroBI) != 0 && time.Now().Unix() <= woPriceTimestamp.Int64()+state.StaleDuration.Int64() {
		woFeasible = true
	}

	woPriceInBound := false
	if cloPrice.Cmp(bignumber.ZeroBI) == 0 ||
		new(big.Int).Div(new(big.Int).Mul(cloPrice, new(big.Int).Sub(bignumber.BONE, state.Bound)), bignumber.BONE).Cmp(woPrice) <= 0 &&
			new(big.Int).Div(new(big.Int).Mul(cloPrice, new(big.Int).Add(bignumber.BONE, state.Bound)), bignumber.BONE).Cmp(woPrice) >= 0 {
		woPriceInBound = true
	}

	if woFeasible {
		return woPrice, woPriceInBound
	} else {
		priceOut := cloPrice
		if !state.TokenInfos[base].State.CloPreferred {
			priceOut = big.NewInt(0)
		}
		feasible := false
		if priceOut.Cmp(bignumber.ZeroBI) != 0 {
			feasible = true
		}

		return priceOut, feasible
	}
}
