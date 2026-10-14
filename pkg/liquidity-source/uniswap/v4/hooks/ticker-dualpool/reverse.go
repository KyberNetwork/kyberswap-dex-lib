package tickerdualpool

import (
	"math/big"

	"github.com/holiman/uint256"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// Separate fee floors differ from the combined floor by at most one unit.
// GrossBeforeFee(desired-1) is a lower bound. With fees capped at 20%, the
// safe GrossBeforeFee(desired) bound is at most two gross units above it.
func grossBeforeOutputFees(desired *uint256.Int, hook, tax uint64) (*uint256.Int, error) {
	lowerNet := new(big.Int).Sub(desired.ToBig(), bignumber.One)
	g := uniswapv4.GrossBeforeFee(lowerNet, new(big.Int).SetUint64(hook+tax), big.NewInt(totalWeightBps))
	gross, overflow := uint256.FromBig(g)
	if overflow {
		return nil, ErrInvalidState
	}
	if gross.IsZero() {
		gross.SetOne()
	}
	for checks := 0; checks < 3; checks++ {
		_, _, net := outputFees(gross, hook, tax)
		if net.Cmp(desired) >= 0 {
			return gross, nil
		}
		gross.AddUint64(gross, 1)
	}
	return nil, ErrInvalidState
}

// QuoteRequiredInput inverts output fees before pricing temporary liquidity.
// Every probe reuses one allocation; teardown is calculated only for the result.
func QuoteRequiredInput(s State, zeroForOne bool, desired *uint256.Int) (Quote, error) {
	if desired == nil || desired.IsZero() || desired.BitLen() > 127 {
		return Quote{}, ErrInvalidState
	}
	state, err := realizablePool(s)
	if err != nil {
		return Quote{}, err
	}
	hook, tax, err := outputFeeRates(s)
	if err != nil {
		return Quote{}, err
	}
	gross, err := grossBeforeOutputFees(desired, hook, tax)
	if err != nil {
		return Quote{}, err
	}
	prepared, err := prepareTemporaryLiquidity(state, s.TickSpacing, zeroForOne)
	if err != nil {
		return Quote{}, err
	}
	if state.SqrtPriceX96.IsZero() {
		return Quote{}, ErrInvalidQuoteState
	}
	grossQuote := func(input *big.Int) (*big.Int, error) {
		var amount uint256.Int
		if amount.SetFromBig(input) {
			return nil, ErrInvalidState
		}
		result, err := prepared.swap(&amount)
		if err != nil {
			return nil, err
		}
		return result.amountOut.ToBig(), nil
	}
	// Reuse DualPool's spot-price guess rather than equating output and input
	// units when the currencies can differ by many orders of magnitude.
	var price, guess big.Int
	price.Mul(state.SqrtPriceX96.ToBig(), state.SqrtPriceX96.ToBig())
	q192 := new(big.Int).Lsh(big.NewInt(1), 192)
	if zeroForOne {
		bignumber.MulDivUp(&guess, gross.ToBig(), q192, &price)
	} else {
		bignumber.MulDivUp(&guess, gross.ToBig(), &price, q192)
	}
	for searches := 0; searches < 2; searches++ {
		if gross.BitLen() > 127 {
			return Quote{}, uniswapv4.ErrAmountOutUnreachable
		}
		input, err := uniswapv4.MinAmountIn(gross.ToBig(), &guess, grossQuote)
		if err != nil {
			return Quote{}, err
		}
		var amount uint256.Int
		if amount.SetFromBig(input) {
			return Quote{}, ErrInvalidState
		}
		result, err := prepared.swap(&amount)
		if err != nil {
			return Quote{}, err
		}
		_, _, net := outputFees(result.amountOut, hook, tax)
		if net.Cmp(desired) >= 0 {
			swap, err := prepared.finish(result)
			if err != nil {
				return Quote{}, err
			}
			return quoteWithOutputFees(s, swap)
		}
		// A price step skipped the qualifying gross and hit a one-unit fee dip.
		// Only the middle of the three-unit window can do this; the next gross
		// reaches the safe combined-fee bound. One additional search suffices.
		gross.AddUint64(result.amountOut, 1)
		guess.Set(input)
	}
	return Quote{}, uniswapv4.ErrAmountOutUnreachable
}
