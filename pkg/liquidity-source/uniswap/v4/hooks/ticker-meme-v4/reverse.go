package tickermemev4

import (
	"math/big"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var _ uniswapv4.HookExactInputReverser = (*Hook)(nil)

func (h *Hook) ReverseExactInput(param pool.CalcAmountInParams, quote, grossQuote uniswapv4.ExactInputQuote) (*pool.CalcAmountInResult, error) {
	if err := h.validate(); err != nil {
		return nil, err
	}
	desired := param.TokenAmountOut.Amount
	if desired == nil || desired.Sign() <= 0 || desired.BitLen() > 127 {
		return nil, uniswapv4.ErrInvalidAmountOut
	}
	keep := int64(basisPoints)
	if h.Registered {
		keep -= int64(h.HookFeeBps) + int64(h.CreatorTaxBps)
	}
	// Net = gross*keep/10000 plus less than two units from the fee floors.
	// Start strictly above (desired-2)*10000/keep; no smaller gross can qualify.
	candidate := new(big.Int).Sub(desired, big.NewInt(2))
	if candidate.Sign() < 0 {
		candidate.SetInt64(0)
	}
	candidate.Mul(candidate, big.NewInt(basisPoints)).Div(candidate, big.NewInt(keep)).Add(candidate, big.NewInt(1))
	upper := new(big.Int).Mul(desired, big.NewInt(basisPoints))
	upper.Add(upper, big.NewInt(keep-1)).Div(upper, big.NewInt(keep))
	params := func(input *big.Int) pool.CalcAmountOutParams {
		return pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: param.TokenIn, Amount: input}, TokenOut: param.TokenAmountOut.Token, Limit: param.Limit}
	}
	calcGross := func(input *big.Int) (*big.Int, error) {
		result, err := grossQuote(params(input))
		if err != nil {
			return nil, err
		}
		return result.TokenAmountOut.Amount, nil
	}
	// At most four gross values lie between these bounds because keep >= 8000.
	// Net output can fall by one when both fees increment together, so binary
	// search only gross output, then replay the exact-input quote to verify net.
	for candidate.Cmp(upper) <= 0 {
		fee, err := h.fee(candidate)
		if err != nil {
			return nil, err
		}
		if new(big.Int).Sub(candidate, fee).Cmp(desired) < 0 {
			candidate.Add(candidate, big.NewInt(1))
			continue
		}
		input, err := uniswapv4.MinAmountIn(candidate, nil, calcGross)
		if err != nil {
			return nil, err
		}
		result, err := quote(params(input))
		if err != nil {
			return nil, err
		}
		if result.TokenAmountOut.Amount.Cmp(desired) >= 0 {
			return &pool.CalcAmountInResult{
				TokenAmountIn:           &pool.TokenAmount{Token: param.TokenIn, Amount: input},
				RemainingTokenAmountOut: &pool.TokenAmount{Token: param.TokenAmountOut.Token, Amount: new(big.Int)},
				Fee:                     result.Fee, Gas: result.Gas, SwapInfo: result.SwapInfo,
			}, nil
		}
		// A price step skipped the candidate and landed on a fee boundary.
		actual, err := calcGross(input)
		if err != nil {
			return nil, err
		}
		candidate.Add(actual, big.NewInt(1))
	}
	return nil, uniswapv4.ErrAmountOutUnreachable
}
