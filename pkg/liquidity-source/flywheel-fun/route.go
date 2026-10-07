package flywheelfun

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"

	v3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	v4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func asset(a common.Address) string {
	if a == (common.Address{}) {
		return WETH
	}
	return hexutil.Encode(a[:])
}
func keyID(k poolKey) (string, error) {
	if k.Fee == nil || k.TickSpacing == nil || k.Fee.Sign() < 0 || k.Fee.BitLen() > 24 || !k.TickSpacing.IsInt64() || k.TickSpacing.Sign() <= 0 || k.TickSpacing.Int64() > 32767 || k.Currency0.Big().Cmp(k.Currency1.Big()) >= 0 {
		return "", ErrState
	}
	data, err := keyArguments.Pack(k.Currency0, k.Currency1, k.Fee, k.TickSpacing, k.Hooks)
	if err != nil {
		return "", err
	}
	return crypto.Keccak256Hash(data).Hex(), nil
}

// hopFor builds the settlement route hop for an indexed base pool. A V4 key is
// rebuilt from the simulator and must hash to the pool id.
func hopFor(base pool.IPoolSimulator) (routeHop, error) {
	switch b := base.(type) {
	case *v3.PoolSimulator:
		return routeHop{Kind: 3, Pool: common.HexToAddress(b.GetAddress()), Key: poolKey{Fee: new(big.Int), TickSpacing: new(big.Int)}}, nil
	case *v4.PoolSimulator:
		t := b.GetTokens()
		m := b.GetMetaInfo(t[0], t[1]).(v4.PoolMetaInfo)
		// The settlement passes no hook data; only hook-free pools are modeled.
		if m.HookAddress != (common.Address{}) {
			return routeHop{}, ErrUnsupported
		}
		key := poolKey{Currency0: m.TokenIn, Currency1: m.TokenOut, Fee: big.NewInt(int64(m.Fee)), TickSpacing: big.NewInt(int64(m.TickSpacing))}
		if id, err := keyID(key); err != nil || id != b.GetAddress() {
			return routeHop{}, ErrState
		}
		return routeHop{Kind: 4, Key: key}, nil
	}
	return routeHop{}, ErrUnsupported
}
func core(p pool.IPoolSimulator) (*v3.PoolSimulator, error) {
	switch x := p.(type) {
	case *v3.PoolSimulator:
		return x, nil
	case *v4.PoolSimulator:
		return x.PoolSimulator, nil
	default:
		return nil, ErrUnsupported
	}
}

// hopSwap quotes one hop with the pool's own simulator. It does not mutate p.
// Full input consumption is mandatory in NativeQuoteRoute and NativeMarketAdapter.
func hopSwap(p pool.IPoolSimulator, amount *uint256.Int, in, out string, index int) (uint256.Int, SwapStep, int64, error) {
	var zero uint256.Int
	if amount.IsZero() || amount.BitLen() > 255 {
		return zero, SwapStep{}, 0, ErrAmount
	}
	amountIn := pool.TokenAmount{Token: in, Amount: amount.ToBig()}
	r, err := p.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: amountIn, TokenOut: out})
	if err != nil {
		return zero, SwapStep{}, 0, err
	}
	if r.RemainingTokenAmountIn != nil && r.RemainingTokenAmountIn.Amount.Sign() != 0 {
		return zero, SwapStep{}, 0, ErrCurve
	}
	value, overflow := uint256.FromBig(r.TokenAmountOut.Amount)
	if overflow || value.IsZero() {
		return zero, SwapStep{}, 0, ErrAmount
	}
	step := SwapStep{Index: index, Params: pool.UpdateBalanceParams{TokenAmountIn: amountIn, TokenAmountOut: *r.TokenAmountOut, SwapInfo: r.SwapInfo}}
	if r.Fee != nil {
		step.Params.Fee = *r.Fee
	}
	return *value, step, r.Gas, nil
}

// swapCL quotes and updates the caller-owned clone with the actual directional fee.
// The pinned native hook has no custom swap deltas.
func swapCL(p pool.IPoolSimulator, amount *uint256.Int, in, out string, fee uint32, index int) (uint256.Int, SwapStep, int64, error) {
	var zero uint256.Int
	if amount.IsZero() || amount.BitLen() > 255 || fee >= 1_000_000 {
		return zero, SwapStep{}, 0, ErrAmount
	}
	c, err := core(p)
	if err != nil {
		return zero, SwapStep{}, 0, err
	}
	sim := c.CloneState().(*v3.PoolSimulator)
	sim.V3Pool.Fee = v3.FeeAmount(fee)
	r, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: in, Amount: amount.ToBig()}, TokenOut: out})
	if err != nil {
		return zero, SwapStep{}, 0, err
	}
	if r.RemainingTokenAmountIn != nil && r.RemainingTokenAmountIn.Amount.Sign() != 0 {
		return zero, SwapStep{}, 0, ErrCurve
	}
	value, overflow := uint256.FromBig(r.TokenAmountOut.Amount)
	if overflow || value.IsZero() {
		return zero, SwapStep{}, 0, ErrAmount
	}
	var info any = r.SwapInfo
	if _, ok := p.(*v4.PoolSimulator); ok {
		info = v4.SwapInfo{PoolSwapInfo: r.SwapInfo.(v3.SwapInfo)}
	}
	step := SwapStep{Index: index, Before: fingerprint(p), Params: pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: in, Amount: amount.ToBig()}, TokenAmountOut: *r.TokenAmountOut, Fee: *r.Fee, SwapInfo: info}}
	p.UpdateBalance(step.Params)
	return *value, step, r.Gas, nil
}

// quoteToWETH values amount of quote at the pool's spot price (token1 per token0).
// It is only a reserve proxy; quotes execute with price impact.
func quoteToWETH(amount, sqrtPriceX96 *uint256.Int, quoteIsToken0 bool) error {
	if sqrtPriceX96.IsZero() {
		return ErrState
	}
	x, d := sqrtPriceX96, big256.U2Pow96
	if !quoteIsToken0 {
		x, d = d, x
	}
	_, o1 := amount.MulDivOverflow(amount, x, d)
	_, o2 := amount.MulDivOverflow(amount, x, d)
	if o1 || o2 {
		return ErrState
	}
	return nil
}

func fingerprint(p pool.IPoolSimulator) string {
	c, e := core(p)
	if e != nil {
		return ""
	}
	return fmt.Sprintf("%s/%s/%s/%d/%s/%s", p.GetAddress(), c.V3Pool.SqrtRatioX96.Dec(), c.V3Pool.Liquidity.Dec(), c.V3Pool.TickCurrent, c.Info.Reserves[0], c.Info.Reserves[1])
}
func cloneBases(bases []pool.IPoolSimulator) []pool.IPoolSimulator {
	out := make([]pool.IPoolSimulator, len(bases))
	for i, p := range bases {
		if p != nil {
			out[i] = p.CloneState()
		}
	}
	return out
}
func (s *PoolSimulator) externalSwap(amount *uint256.Int, buy bool, bases []pool.IPoolSimulator) (uint256.Int, []SwapStep, int64, error) {
	if bases[0] == nil {
		if s.baseQuote() != WETH {
			return uint256.Int{}, nil, 0, ErrBasePool
		}
		return *amount, nil, 0, nil
	}
	in, out := WETH, s.baseQuote()
	if !buy {
		in, out = out, in
	}
	result, step, gas, err := hopSwap(bases[0], amount, in, out, 0)
	if err != nil {
		return result, nil, 0, err
	}
	step.Before = fingerprint(bases[0])
	bases[0].UpdateBalance(step.Params)
	return result, []SwapStep{step}, gas, nil
}
