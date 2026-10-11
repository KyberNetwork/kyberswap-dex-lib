package tickerdualpool

import (
	"math/big"

	"github.com/holiman/uint256"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

// QuoteRequiredInput reverses an exact-input swap. Ticker rounds two fees
// separately, so net output can FALL by one unit when both fees increment at
// once. Binary-searching net output would therefore assume something false.
// Instead search the monotone gross swap output, then check the exact two fees.
func QuoteRequiredInput(s State, zeroForOne bool, desired *uint256.Int) (Quote, error) {
	if desired == nil || desired.IsZero() || desired.BitLen() > 127 {
		return Quote{}, ErrInvalidState
	}
	if _, err := realizablePool(s); err != nil {
		return Quote{}, err
	}
	hook, tax := uint64(0), uint64(0)
	if s.Registered {
		hook = uint64(s.HookFeeBps)
		tax = uint64(s.CreatorTaxBps)
	}
	if hook+tax >= 10000 {
		return Quote{}, ErrInvalidState
	}
	// net(g) is strictly less than g*(1-fees)+2. This is the first
	// integer gross output that could possibly deliver the desired net amount.
	target := desired.ToBig()
	candidate := new(big.Int).Sub(target, big.NewInt(2))
	if candidate.Sign() < 0 {
		candidate.SetInt64(0)
	}
	candidate.Mul(candidate, big.NewInt(10000)).Div(candidate, new(big.Int).SetUint64(10000-hook-tax)).Add(candidate, big.NewInt(1))
	grossQuote := func(input *big.Int) (*big.Int, error) {
		amount, overflow := uint256.FromBig(input)
		if overflow {
			return nil, ErrInvalidState
		}
		q, err := QuoteExactInput(s, zeroForOne, amount)
		if err != nil {
			return nil, err
		}
		return q.Swap.AmountOut.ToBig(), nil
	}
	for attempts := 0; attempts <= 20000; attempts++ {
		g := uint256.MustFromBig(candidate)
		fee := new(uint256.Int).Mul(g, uint256.NewInt(hook))
		fee.Div(fee, uint256.NewInt(10000))
		taxAmount := new(uint256.Int).Mul(g, uint256.NewInt(tax))
		taxAmount.Div(taxAmount, uint256.NewInt(10000))
		net := new(uint256.Int).Sub(g, fee)
		net.Sub(net, taxAmount)
		if net.Cmp(desired) < 0 {
			candidate.Add(candidate, big.NewInt(1))
			continue
		}
		input, err := uniswapv4.MinAmountIn(candidate, nil, grossQuote)
		if err != nil {
			return Quote{}, err
		}
		q, err := QuoteExactInput(s, zeroForOne, uint256.MustFromBig(input))
		if err != nil {
			return Quote{}, err
		}
		if q.NetOutput.Cmp(desired) >= 0 {
			return q, nil
		}
		// The gross quote skipped our candidate and landed on a fee boundary.
		// All smaller inputs already fell below the first qualifying candidate.
		candidate.Add(q.Swap.AmountOut.ToBig(), big.NewInt(1))
	}
	return Quote{}, uniswapv4.ErrAmountOutUnreachable
}
