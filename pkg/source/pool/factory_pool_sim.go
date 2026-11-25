package pool

import (
	"math/big"

	"github.com/holiman/uint256"
	"github.com/pkg/errors"
)

// PoolSimFactoryFn builds a pool that implements IPoolSim.
type PoolSimFactoryFn func(FactoryParams) (IPoolSim, error)

// FactoryPoolSim returns the factory for a pool type as an IPoolSim. A registered pool that already implements IPoolSim
// is returned as-is. A legacy pool is wrapped by legacyAdapter.
func FactoryPoolSim(poolType string) PoolSimFactoryFn {
	factory := factoryMap[poolType]
	if factory == nil {
		return nil
	}
	return func(params FactoryParams) (IPoolSim, error) {
		sim, err := factory(params)
		if err != nil {
			return nil, err
		}
		return AsIPoolSim(sim), nil
	}
}

// AsIPoolSim returns sim as an IPoolSim: as-is when it already implements IPoolSim, else wrapped by legacyAdapter.
func AsIPoolSim(sim any) IPoolSim {
	if u, ok := sim.(IPoolSim); ok {
		return u
	}
	return legacyAdapter{IPoolSimulator: sim.(IPoolSimulator)}
}

// legacyAdapter exposes a legacy big.Int IPoolSimulator as IPoolSim. Each uint256 call converts to big.Int, calls
// the legacy method, and converts the result back.
type legacyAdapter struct {
	IPoolSimulator
}

func (a legacyAdapter) CalcAmtOut(params CalcAmtOutParams) (*CalcAmtOutResult, error) {
	// A pool with a native CalcAmtOut but a legacy CloneState (it fails IPoolSim) quotes natively.
	if native, ok := a.IPoolSimulator.(ICalcAmtOut); ok {
		return native.CalcAmtOut(params)
	}
	legacy, ok := a.IPoolSimulator.(ILegacyCalcAmountOut)
	if !ok {
		return nil, errors.Errorf("pool %s has no legacy CalcAmountOut", a.GetAddress())
	}
	res, err := legacy.CalcAmountOut(CalcAmountOutParams{
		TokenAmountIn: tokenAmountFromAmt(&params.TokenAmountIn),
		TokenOut:      params.TokenOut,
		Limit:         AsSwapLimitBig(params.Limit),
	})
	if err != nil || res == nil {
		return nil, err
	}
	return calcAmtOutResultFromBig(res)
}

func (a legacyAdapter) CloneState() IPoolSim {
	cloned := a.IPoolSimulator.CloneState()
	if cloned == nil {
		return nil
	}
	return legacyAdapter{IPoolSimulator: cloned}
}

func (a legacyAdapter) UpdateState(params UpdateStateParams) {
	if native, ok := a.IPoolSimulator.(IUpdateState); ok {
		native.UpdateState(params)
		return
	}
	a.IPoolSimulator.UpdateBalance(UpdateBalanceParams{
		TokenAmountIn:  tokenAmountFromAmt(&params.TokenAmountIn),
		TokenAmountOut: tokenAmountFromAmt(&params.TokenAmountOut),
		SwapInfo:       params.SwapInfo,
		SwapLimitBig:   AsSwapLimitBig(params.SwapLimit),
	})
}

func (a legacyAdapter) CalcLimit() map[string]*uint256.Int {
	return limitsToU256(a.IPoolSimulator.CalculateLimit())
}

func (a legacyAdapter) GetReserves() []uint256.Int {
	reserves := a.IPoolSimulator.GetReserves()
	out := make([]uint256.Int, len(reserves))
	for i, r := range reserves {
		out[i] = *bigToU256OrZero(r)
	}
	return out
}

func tokenAmountFromAmt(t *TokenAmt) TokenAmount {
	return TokenAmount{Token: t.Token, Amount: t.Amount.ToBig(), AmountUsd: t.AmountUsd}
}

// tokenAmtFromBig converts a legacy amount. A nil, negative or oversized amount is an error.
func tokenAmtFromBig(t *TokenAmount) (TokenAmt, error) {
	amt, err := bigToU256(t.Amount)
	return TokenAmt{Token: t.Token, Amount: amt, AmountUsd: t.AmountUsd}, err
}

// calcAmountOutResultFromAmt converts a native result to the legacy shape. The native result has no fee, so Fee is a
// zero amount (not nil) for legacy callers that read Fee.Amount.
func calcAmountOutResultFromAmt(r *CalcAmtOutResult) *CalcAmountOutResult {
	out := tokenAmountFromAmtPtr(r.TokenAmountOut)
	fee := &TokenAmount{Amount: new(big.Int)}
	if out != nil {
		fee.Token = out.Token
	}
	return &CalcAmountOutResult{
		TokenAmountOut:         out,
		Fee:                    fee,
		RemainingTokenAmountIn: tokenAmountFromAmtPtr(r.RemainingTokenAmountIn),
		Gas:                    r.Gas,
		SwapInfo:               r.SwapInfo,
	}
}

func tokenAmountFromAmtPtr(t *TokenAmt) *TokenAmount {
	if t == nil {
		return nil
	}
	out := tokenAmountFromAmt(t)
	return &out
}

// tokenAmountPtrToAmt converts an optional legacy amount. A nil pointer stays nil; a nil, negative or oversized
// Amount is an error.
func tokenAmountPtrToAmt(t *TokenAmount) (*TokenAmt, error) {
	if t == nil {
		return nil, nil
	}
	out := &TokenAmt{Token: t.Token, AmountUsd: t.AmountUsd}
	amt, err := bigToU256(t.Amount)
	out.Amount = amt
	return out, err
}

// calcAmtOutResultFromBig converts a legacy result. The amount out and the remaining amount must be valid
// uint256 values. The legacy Fee is dropped: a pool whose UpdateBalance needs it carries it in its SwapInfo.
func calcAmtOutResultFromBig(r *CalcAmountOutResult) (*CalcAmtOutResult, error) {
	out, err := tokenAmountPtrToAmt(r.TokenAmountOut)
	if err != nil {
		return nil, errors.Wrap(err, "token amount out")
	}
	remaining, err := tokenAmountPtrToAmt(r.RemainingTokenAmountIn)
	if err != nil {
		return nil, errors.Wrap(err, "remaining token amount in")
	}
	return &CalcAmtOutResult{TokenAmountOut: out, RemainingTokenAmountIn: remaining, Gas: r.Gas,
		SwapInfo: r.SwapInfo}, nil
}

func limitsToU256(limits map[string]*big.Int) map[string]*uint256.Int {
	out := make(map[string]*uint256.Int, len(limits))
	for k, v := range limits {
		out[k] = bigToU256OrZero(v)
	}
	return out
}

// bigToU256 converts a big.Int that is non-nil, non-negative and fits 256 bits.
func bigToU256(b *big.Int) (uint256.Int, error) {
	if b == nil || b.Sign() < 0 || b.BitLen() > 256 {
		return uint256.Int{}, ErrInvalidLegacyAmount
	}
	return *uint256.MustFromBig(b), nil
}

// bigToU256OrZero converts a non-negative big.Int that fits 256 bits. A nil, negative, or oversized value
// becomes zero. For reserves and limits only; quote amounts use bigToU256.
func bigToU256OrZero(b *big.Int) *uint256.Int {
	if b == nil || b.Sign() < 0 || b.BitLen() > 256 {
		return new(uint256.Int)
	}
	return uint256.MustFromBig(b)
}

// FeeSwapInfo is the SwapInfo of a legacy pool whose UpdateBalance needs the fee of its CalcAmountOut. The adapter
// passes SwapInfo through untouched, so the fee must travel in it.
type FeeSwapInfo struct {
	Fee *TokenAmount `json:"-"`
}

// SwapInfoFee returns the fee a legacy pool carried in FeeSwapInfo for its UpdateBalance, or a zero fee when
// swapInfo is not a FeeSwapInfo.
func SwapInfoFee(swapInfo any) TokenAmount {
	if info, ok := swapInfo.(FeeSwapInfo); ok && info.Fee != nil && info.Fee.Amount != nil {
		return *info.Fee
	}
	return TokenAmount{Amount: new(big.Int)}
}
