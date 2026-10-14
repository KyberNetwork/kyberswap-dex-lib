// Package tickerdualpool models the extra checks and output fees in TickerDualPoolHook.
package tickerdualpool

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/dualpool"
)

var (
	ErrVaultIlliquid = errors.New("Ticker vault cannot deliver the pool's full backing")
	ErrInvalidState  = errors.New("invalid Ticker pool state")
)

// registerPool caps the hook fee at 10% and both output fees together at 20%.
const maxTickerHookFeeBps = 1000
const maxTickerTradeFeeBps = 2000

func outputFeeRates(s State) (hook, tax uint64, err error) {
	if !s.Registered {
		return 0, 0, nil
	}
	hook, tax = uint64(s.HookFeeBps), uint64(s.CreatorTaxBps)
	if hook > maxTickerHookFeeBps || hook+tax > maxTickerTradeFeeBps {
		return 0, 0, ErrInvalidState
	}
	return hook, tax, nil
}

func outputFees(gross *uint256.Int, hook, tax uint64) (fee, creatorTax, net uint256.Int) {
	// Swap output and fee-inversion candidates fit 128 bits, so these products fit uint256.
	fee.Mul(gross, uint256.NewInt(hook)).Div(&fee, uint256.NewInt(totalWeightBps))
	creatorTax.Mul(gross, uint256.NewInt(tax)).Div(&creatorTax, uint256.NewInt(totalWeightBps))
	net.Sub(gross, &fee).Sub(&net, &creatorTax)
	return
}

// State contains a settled snapshot. Unbacked claims created during a simulated
// unlock must be supplied explicitly; a snapshot from an RPC cannot predict them.
type State struct {
	Pool                      dualpool.Extra
	Total0, Total1            *uint256.Int
	Unbacked0, Unbacked1      *uint256.Int
	TickSpacing               int
	Registered                bool
	HookFeeBps, CreatorTaxBps uint16
}

type Quote struct {
	RawAfter, ClaimsAfter, ManagerCashAfter [2]*uint256.Int
	ReservesAfter                           [2]*AaveReserve
	VaultsAfter                             [2]*AaveInventory
	Swap                                    SwapInfo
	NetOutput, HookFee, CreatorTax          *uint256.Int
}

// QuoteExactInput checks liquidity before applying the same bucket swap and two
// independently rounded output fees as the contract. It never mutates its inputs.
func QuoteExactInput(s State, zeroForOne bool, amountIn *uint256.Int) (Quote, error) {
	poolState, err := realizablePool(s)
	if err != nil {
		return Quote{}, err
	}
	swap, err := quoteTemporaryLiquidity(poolState, s.TickSpacing, zeroForOne, amountIn)
	if err != nil {
		return Quote{}, err
	}
	return quoteWithOutputFees(s, swap)
}

func quoteWithOutputFees(s State, swap SwapInfo) (Quote, error) {
	hookRate, taxRate, err := outputFeeRates(s)
	if err != nil {
		return Quote{}, err
	}
	fee, tax, net := outputFees(swap.AmountOut, hookRate, taxRate)
	if net.IsZero() {
		return Quote{}, dualpool.ErrZeroOutput
	}
	return Quote{Swap: swap, NetOutput: &net, HookFee: &fee, CreatorTax: &tax}, nil
}

// Check the full backing before either forward pricing or reverse search. A
// reserve shortage is independent of trade size and should retain its error.
func realizablePool(s State) (dualpool.Extra, error) {
	if s.Total0 == nil || s.Total1 == nil || s.Pool.Balance0 == nil || s.Pool.Balance1 == nil {
		return dualpool.Extra{}, ErrInvalidState
	}
	balances := [2]*uint256.Int{s.Pool.Balance0, s.Pool.Balance1}
	totals := [2]*uint256.Int{s.Total0, s.Total1}
	unbacked := [2]*uint256.Int{s.Unbacked0, s.Unbacked1}
	poolState := s.Pool
	for i := range balances {
		var available uint256.Int
		available.Set(balances[i])
		if unbacked[i] != nil {
			if available.Cmp(unbacked[i]) < 0 {
				return dualpool.Extra{}, ErrInvalidState
			}
			available.Sub(&available, unbacked[i])
		}
		if available.Cmp(totals[i]) < 0 {
			return dualpool.Extra{}, ErrVaultIlliquid
		}
		if i == 0 {
			poolState.Balance0 = available.Clone()
		} else {
			poolState.Balance1 = available.Clone()
		}
	}

	return poolState, nil
}
