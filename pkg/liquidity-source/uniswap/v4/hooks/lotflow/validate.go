package lotflow

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/holiman/uint256"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
)

// validateExtra checks a decoded extra structurally before the hook uses it: every pointer the
// simulator dereferences is set, every big number is in its on-chain range, and parameters are within
// what the contracts can hold (NavJitHookV17.setParams: band <= 1000, spreads <= band,
// 0 < width <= 2000). Track validates its market before publishing and publishes the empty market
// instead of one that fails (trackMarket); the factory treats an extra that fails (a corrupted extra,
// or an old extra decoded after a schema change) as untracked. A nil Market is the valid fail-closed
// state.
func validateExtra(x *Extra) error {
	if x.Market == nil {
		return nil
	}
	if !x.Tracked {
		return errors.New("market on an untracked extra")
	}
	return validateMarket(x.Market)
}

var errInvalid = errors.New("lotflow: invalid hook extra")

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{errInvalid}, args...)...)
}

func nonNeg(v *big.Int) bool { return v != nil && v.Sign() >= 0 }

func uintBits(v *big.Int, bits int) bool { return nonNeg(v) && v.BitLen() <= bits }

func sqrtPriceOK(v *big.Int) bool {
	return v != nil && v.Cmp(MinSqrtPrice) >= 0 && v.Cmp(MaxSqrtPrice) < 0
}

func decimalsOK(d int) bool { return d >= 0 && d <= 18 }

func validateMarket(m *Market) error {
	v := m.Venue
	switch {
	case v == nil:
		return invalid("no venue")
	case !sqrtPriceOK(v.SqrtPriceX96):
		return invalid("venue sqrtPrice")
	case !decimalsOK(v.UsdgDecimals):
		return invalid("usdg decimals %d", v.UsdgDecimals)
	case v.Params.BandBps > 1000 || v.Params.BuySpreadBps > v.Params.BandBps ||
		v.Params.SellSpreadBps > v.Params.BandBps || v.Params.SizeBufferBps >= 10_000:
		return invalid("hook params %+v", v.Params)
	case v.Params.WidthTicks <= 0 || v.Params.WidthTicks > 2000:
		return invalid("width %d", v.Params.WidthTicks)
	case !uintBits(m.PmLotBalance, 256):
		return invalid("PoolManager LOT balance")
	case len(m.Constituents) == 0:
		return invalid("no constituents")
	}
	if s := v.Standing; s != nil && (!uintBits(s.Liquidity, 128) || s.Lower >= s.Upper) {
		return invalid("standing position")
	}
	if m.Mq != nil {
		if err := validateMq(m.Mq); err != nil {
			return err
		}
	}
	for i := range m.Constituents {
		if err := validateConstituent(&m.Constituents[i], m.Mq != nil); err != nil {
			return fmt.Errorf("constituent %d: %w", i, err)
		}
	}
	return nil
}

func validateConstituent(c *MarketConstituent, mqMarket bool) error {
	switch {
	case !uintBits(c.Unit, 256) || c.Unit.Sign() == 0:
		return invalid("unit")
	case !uintBits(c.Nav.Unit, 256):
		return invalid("nav unit")
	case !uintBits(c.Nav.SpotUsd18, 256):
		return invalid("spot")
	case !decimalsOK(c.Nav.StockDecimals) || !decimalsOK(c.StableDecimals):
		return invalid("decimals")
	case len(c.Pools) == 0:
		return invalid("no pools")
	}
	if f := c.Nav.Feed; f != nil && (f.Answer == nil || !decimalsOK(f.Decimals)) {
		return invalid("feed")
	}
	for id, p := range c.Pools {
		if err := validatePool(p); err != nil {
			return fmt.Errorf("pool %s: %w", id, err)
		}
	}
	if c.Mq != nil {
		if !mqMarket || len(c.Pools) != 1 || !decimalsOK(c.Mq.TokenDecimals) {
			return invalid("mq token")
		}
		return validateTwap(c.Mq.Twap)
	}
	if c.Registry.PriceIndex < 0 || c.Registry.PriceIndex >= len(c.Registry.Pools) {
		return invalid("price index %d of %d pools", c.Registry.PriceIndex, len(c.Registry.Pools))
	}
	for _, p := range c.Registry.Pools {
		if !uintBits(p.SqrtPriceX96, 160) || !uintBits(p.Liquidity, 128) {
			return invalid("registry pool %s", p.PoolID)
		}
	}
	for _, b := range c.Venues {
		if (b.SqrtOK && !uintBits(b.SqrtPriceX96, 256)) || (b.LiqOK && !uintBits(b.Liquidity, 256)) {
			return invalid("book venue %s", b.Pool)
		}
	}
	return nil
}

func validatePool(p *LegPool) error {
	switch {
	case p == nil:
		return invalid("nil pool")
	case !sqrtPriceOK(p.SqrtPriceX96):
		return invalid("sqrtPrice")
	case !uintBits(p.Liquidity, 128):
		return invalid("liquidity")
	case p.TickSpacing <= 0 || p.TickSpacing > 32767:
		return invalid("tick spacing %d", p.TickSpacing)
	}
	// slot0's tick is its sqrtPrice's tick, or one below it after a downward stop on a tick boundary
	t, err := TickAtSqrtPrice(p.SqrtPriceX96)
	if err != nil {
		return invalid("sqrtPrice")
	}
	if p.Tick != t {
		if at, err := SqrtPriceAtTick(t); err != nil || p.Tick != t-1 || at.Cmp(p.SqrtPriceX96) != 0 {
			return invalid("tick %d at the tick %d of its sqrtPrice", p.Tick, t)
		}
	}
	// initialized ticks: strictly ascending multiples of the spacing inside the tick range, and the
	// liquidity they imply in every interval of the window stays a uint128 (the engine walks them)
	split := len(p.Ticks)
	for i, tk := range p.Ticks {
		switch {
		case !uintBits(u256(tk.LiquidityGross), 128) || tk.LiquidityNet == nil:
			return invalid("tick %d", tk.Index)
		case i > 0 && tk.Index <= p.Ticks[i-1].Index:
			return invalid("ticks not strictly ascending at %d", tk.Index)
		case tk.Index%p.TickSpacing != 0 || tk.Index < uniswapv3.MinTick || tk.Index > uniswapv3.MaxTick:
			return invalid("tick index %d", tk.Index)
		}
		if split == len(p.Ticks) && tk.Index > p.Tick {
			split = i
		}
	}
	up, down := new(big.Int).Set(p.Liquidity), new(big.Int).Set(p.Liquidity)
	for i := split; i < len(p.Ticks); i++ {
		if !uintBits(up.Add(up, p.Ticks[i].LiquidityNet.ToBig()), 128) {
			return invalid("liquidity above tick %d", p.Ticks[i].Index)
		}
	}
	for i := split - 1; i >= 0; i-- {
		if !uintBits(down.Sub(down, p.Ticks[i].LiquidityNet.ToBig()), 128) {
			return invalid("liquidity below tick %d", p.Ticks[i].Index)
		}
	}
	return nil
}

func u256(v *uint256.Int) *big.Int {
	if v == nil {
		return nil
	}
	return v.ToBig()
}

func validateTwap(s *TwapState) error {
	if s == nil || s.Cardinality <= 0 || s.Index < 0 || s.Index >= s.Cardinality {
		return invalid("twap state")
	}
	for i, o := range s.Obs {
		if i < 0 || i >= s.Cardinality || o.TickCumulative == nil {
			return invalid("observation %d", i)
		}
	}
	return nil
}

func validateMq(q *MqMarket) error {
	switch {
	case validatePool(q.Conv) != nil:
		return invalid("conversion pool")
	case q.EthUsd == nil || q.EthUsd.Answer == nil || !decimalsOK(q.EthUsd.Decimals):
		return invalid("ETH/USD feed")
	case q.UsdgUsd == nil || q.UsdgUsd.Answer == nil || !decimalsOK(q.UsdgUsd.Decimals):
		return invalid("USDG/USD feed")
	case !uintBits(q.PmWeth, 256):
		return invalid("PoolManager WETH balance")
	case !decimalsOK(q.UsdgDecimals):
		return invalid("usdg decimals")
	}
	return validateTwap(q.ConvTwap)
}
