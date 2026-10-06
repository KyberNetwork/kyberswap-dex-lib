package flywheelfun

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	v3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	v4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func asset(a common.Address) string {
	if a == (common.Address{}) {
		return WETH
	}
	return hexutil.Encode(a[:])
}
func keyID(k PoolKey) (string, error) {
	if k.Fee == nil || k.TickSpacing == nil || k.Fee.Sign() < 0 || k.Fee.BitLen() > 24 || !k.TickSpacing.IsInt64() || k.TickSpacing.Sign() <= 0 || k.TickSpacing.Int64() > 32767 || k.Currency0.Big().Cmp(k.Currency1.Big()) >= 0 {
		return "", ErrState
	}
	data, err := keyArguments.Pack(k.Currency0, k.Currency1, k.Fee, k.TickSpacing, k.Hooks)
	if err != nil {
		return "", err
	}
	return crypto.Keccak256Hash(data).Hex(), nil
}
func (h RouteHop) id() (string, error) {
	if h.Kind == 3 && h.Pool != (common.Address{}) {
		return hexutil.Encode(h.Pool[:]), nil
	}
	if h.Kind != 4 || h.Pool != (common.Address{}) {
		return "", ErrUnsupported
	}
	return keyID(h.Key)
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
func fingerprint(p pool.IPoolSimulator) string {
	c, e := core(p)
	if e != nil {
		return ""
	}
	return fmt.Sprintf("%s/%s/%s/%d/%s/%s", p.GetAddress(), c.V3Pool.SqrtRatioX96.Dec(), c.V3Pool.Liquidity.Dec(), c.V3Pool.TickCurrent, c.Info.Reserves[0], c.Info.Reserves[1])
}
func routePool(ep entity.Pool, h RouteHop) (pool.IPoolSimulator, error) {
	if h.Kind == 3 {
		return v3.NewPoolSimulator(ep, 4663)
	}
	return v4.NewPoolSimulator(ep, 4663)
}
func validateBase(p pool.IPoolSimulator, ep entity.Pool, h RouteHop, block uint64) error {
	c, err := core(p)
	if err != nil {
		return err
	}
	id, err := h.id()
	if err != nil {
		return err
	}
	if id != p.GetAddress() || c.Info.BlockNumber != block || ep.BlockNumber != block || len(ep.Tokens) != 2 || len(c.Info.Tokens) != 2 || ep.Tokens[0].Address != c.Info.Tokens[0] || ep.Tokens[1].Address != c.Info.Tokens[1] {
		return ErrState
	}
	if h.Kind == 3 {
		if _, ok := p.(*v3.PoolSimulator); !ok || ep.Type != v3.DexTypeUniswapV3 || c.V3Pool.Fee != v3.FeeAmount(ep.SwapFee) {
			return ErrUnsupported
		}
	} else {
		x, ok := p.(*v4.PoolSimulator)
		if !ok {
			return ErrUnsupported
		}
		meta := x.GetMetaInfo(c.Info.Tokens[0], c.Info.Tokens[1]).(v4.PoolMetaInfo)
		if h.Key.Hooks != (common.Address{}) || meta.HookAddress != (common.Address{}) || h.Key.Fee.Uint64() >= 1_000_000 || meta.Fee != uint32(h.Key.Fee.Uint64()) || meta.TickSpacing != int32(h.Key.TickSpacing.Int64()) || h.LPFee != uint32(h.Key.Fee.Uint64()) {
			return ErrUnsupported
		}
		if c.Info.Tokens[0] != asset(h.Key.Currency0) || c.Info.Tokens[1] != asset(h.Key.Currency1) {
			return ErrState
		}
	}
	return nil
}

// swapCL runs the existing concentrated-liquidity math with the actual directional
// fee. Hook-free V4 routes and the pinned native hook have no custom swap deltas.
// Full input consumption is mandatory in NativeQuoteRoute and NativeMarketAdapter.
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
func (s *PoolSimulator) routeSwap(amount *uint256.Int, buy bool, bases []pool.IPoolSimulator) (uint256.Int, []SwapStep, int64, error) {
	result := *amount
	current := WETH
	if !buy {
		current = s.Static.Quote
	}
	var steps []SwapStep
	var gas int64
	for n := range s.Route {
		i := n
		if !buy {
			i = len(s.Route) - 1 - n
		}
		h := s.Route[i]
		p := bases[i]
		c, err := core(p)
		if err != nil {
			return result, nil, 0, err
		}
		ix := c.GetTokenIndex(current)
		if ix < 0 {
			return result, nil, 0, ErrState
		}
		next := c.Info.Tokens[1-ix]
		fee := uint32(c.V3Pool.Fee)
		if h.Kind == 4 {
			protocol := h.Protocol[ix]
			fee = protocol + h.LPFee - uint32(uint64(protocol)*uint64(h.LPFee)/1_000_000)
		}
		var step SwapStep
		var g int64
		result, step, g, err = swapCL(p, &result, current, next, fee, i)
		if err != nil {
			return result, nil, 0, err
		}
		steps = append(steps, step)
		gas += g
		current = next
	}
	want := s.Static.Quote
	if !buy {
		want = WETH
	}
	if current != want {
		return result, nil, 0, ErrUnsupported
	}
	return result, steps, gas, nil
}

func copyRoutes(routes []RouteHop) []RouteHop {
	b, _ := json.Marshal(routes)
	var out []RouteHop
	_ = json.Unmarshal(b, &out)
	return out
}
func cloneBases(bases []pool.IPoolSimulator) []pool.IPoolSimulator {
	out := make([]pool.IPoolSimulator, len(bases))
	for i, p := range bases {
		out[i] = p.CloneState()
	}
	return out
}
func cloneReserves(r []*big.Int) []*big.Int {
	out := make([]*big.Int, len(r))
	for i, n := range r {
		out[i] = new(big.Int).Set(n)
	}
	return out
}
