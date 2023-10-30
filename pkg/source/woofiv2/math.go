package woofiv2

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"math/big"
	"strings"
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

	if balance(state.QuoteToken, quoteAmount, state).Cmp(quoteAmount) < 0 {
		return nil, ErrQuoteBalanceNotEnough
	}

	swapFee := new(big.Int).Div(
		new(big.Int).Mul(quoteAmount, baseTokenInfo.FeeRate),
		big.NewInt(1e5),
	)
	quoteAmountAfterFee := new(big.Int).Sub(quoteAmount, swapFee)
	state.UnclaimedFee = new(big.Int).Add(state.UnclaimedFee, swapFee)

	baseAmount, newPrice, err := calcBaseAmountSellQuote(baseToken, quoteAmountAfterFee, state)
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

	if balance(baseToken, baseAmount, state).Cmp(baseAmount) < 0 {
		return nil, ErrBaseBalanceNotEnough
	}

	quoteAmount, newPrice, err := calcQuoteAmountSellBase(baseToken, baseAmount, state)
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

	if balance(baseToken1, base1Amount, state).Cmp(base1Amount) < 0 {
		return nil, ErrBaseBalanceNotEnough
	}

	spread := new(big.Int).Div(
		maxBigInt(base1TokenInfo.State.Spread, base2TokenInfo.State.Spread),
		bignumber.Two,
	)
	feeRate := maxBigInt(base1TokenInfo.FeeRate, base2TokenInfo.FeeRate)

	base1TokenInfo.State.Spread = spread
	base2TokenInfo.State.Spread = spread

	quoteAmount, newBase1Price, err := calcQuoteAmountSellBase(baseToken1, base1Amount, state)
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

	base2Amount, newBase2Price, err := calcBaseAmountSellQuote(baseToken2, quoteAmountAfterFee, state)
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
	state *WooFiV2State,
) (*big.Int, *big.Int, error) {
	baseTokenInfo, ok := state.TokenInfos[baseToken]
	if !ok {
		return nil, nil, ErrTokenInfoNotFound
	}

	if !baseTokenInfo.State.WoFeasible {
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
			new(big.Int).Div(new(big.Int).Mul(quoteAmount, baseTokenInfo.State.Coeff), desc.QuoteDec),
		),
		baseTokenInfo.State.Spread,
	)
	baseAmount := new(big.Int).Div(
		new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Div(
					new(big.Int).Mul(
						new(big.Int).Mul(quoteAmount, desc.BaseDec),
						desc.PriceDec,
					),
					baseTokenInfo.State.Price,
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
					new(big.Int).Mul(bignumber.Two, new(big.Int).Mul(baseTokenInfo.State.Coeff, quoteAmount)),
				),
				baseTokenInfo.State.Price,
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
	state *WooFiV2State,
) (*big.Int, *big.Int, error) {
	baseTokenInfo, ok := state.TokenInfos[baseToken]
	if !ok {
		return nil, nil, ErrTokenInfoNotFound
	}

	if !baseTokenInfo.State.WoFeasible {
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
					new(big.Int).Mul(baseTokenInfo.State.Coeff, new(big.Int).Mul(baseAmount, baseTokenInfo.State.Price)),
					decs.BaseDec,
				),
				decs.PriceDec,
			),
		),
		baseTokenInfo.State.Spread,
	)
	quoteAmount := new(big.Int).Div(
		new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Div(
					new(big.Int).Mul(baseAmount, new(big.Int).Mul(decs.QuoteDec, baseTokenInfo.State.Price)),
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
							new(big.Int).Mul(bignumber.Two, baseTokenInfo.State.Coeff),
							new(big.Int).Mul(baseTokenInfo.State.Price, baseAmount),
						),
						decs.PriceDec,
					),
					decs.BaseDec,
				),
				baseTokenInfo.State.Price,
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

func balance(token string, amount *big.Int, state *WooFiV2State) *big.Int {
	if strings.EqualFold(token, state.QuoteToken) {
		return new(big.Int).Sub(amount, state.UnclaimedFee)
	}
	return amount
}
