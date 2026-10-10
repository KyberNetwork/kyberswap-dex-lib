package lotflow

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

// F-N1: a NAV per LOT large enough that the venue's sqrtPrice floors raw18 to 0 made PoolUsd18 divide
// by zero (a Go panic) where FullMath.mulDiv reverts on chain. Every entry point must refuse instead.
// AUDIT2 repro: every constituent's Unit scaled by 1e32 (x1e30 still quotes).
func TestNumericExtremesRefuseNotPanic(t *testing.T) {
	for _, path := range []string{"testdata/fixtures/navjit2-buy-1kusd.json", "testdata/mq/mq-buy-1kusd.json"} {
		for _, exp := range []int64{30, 32, 36, 40} {
			h, _ := integrationHook(t, path)
			mul := new(big.Int).Exp(big.NewInt(10), big.NewInt(exp), nil)
			for i := range h.Market.Constituents {
				c := &h.Market.Constituents[i]
				c.Unit = new(big.Int).Mul(c.Unit, mul)
				c.Nav.Unit = new(big.Int).Set(c.Unit)
			}
			raw, err := json.Marshal(h.Extra)
			require.NoError(t, err)
			h = hookFromJSON(raw)
			require.True(t, h.Tracked, "a scaled unit is structurally valid")
			for _, zfo := range []bool{true, false} {
				for _, amt := range []int64{1_000_000, 1_000_000_000_000_000_000} {
					require.Equal(t, "ok", tryCall(10*time.Second, func() {
						_, _ = h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: zfo,
							AmountSpecified: big.NewInt(amt)})
					}), "%s x1e%d BeforeSwap zfo=%v amt=%d", path, exp, zfo, amt)
				}
				require.Equal(t, "ok", tryCall(5*time.Second, func() { h.SqrtPriceLimit(zfo) }), "%s x1e%d limit", path, exp)
			}
			require.Equal(t, "ok", tryCall(10*time.Second, func() { _, _ = h.GetReserves(context.Background(), nil) }),
				"%s x1e%d GetReserves", path, exp)
			if exp == 32 { // the audit's crash size: the sell now meets the chain's revert, not the recover
				_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: !h.Market.Venue.UsdgIs0,
					AmountSpecified: big.NewInt(1_000_000)})
				require.ErrorIs(t, err, ErrMulDivZero, path)
				require.NotErrorIs(t, err, ErrNumeric, path)
			}
		}
	}
}

// PoolUsd18 at a sqrtPrice whose raw18 floors to zero is the chain's FullMath revert, not a panic.
func TestPoolUsd18ZeroDenominator(t *testing.T) {
	v := &Venue{UsdgIs0: true, UsdgDecimals: 6}
	_, err := v.PoolUsd18(new(big.Int).Set(MinSqrtPrice))
	require.ErrorIs(t, err, ErrMulDivZero)
	px, err := v.PoolUsd18(new(big.Int).Lsh(big.NewInt(1), 96))
	require.NoError(t, err)
	require.Equal(t, "1000000000000000000000000000000", px.String()) // 1e30/1e18 * 1e18: price 1 raw = 1e12 USD18
}

// The recover boundary: a panic inside the quote math becomes ErrNumeric, never a crash.
func TestRecoverNumericBoundary(t *testing.T) {
	f := func() (_ *big.Int, err error) {
		defer recoverNumeric(&err)
		return new(big.Int).Quo(big.NewInt(1), new(big.Int)), nil
	}
	_, err := f()
	require.ErrorIs(t, err, ErrNumeric)
	_, err = checkedNavOf(&Market{Constituents: []MarketConstituent{{}}}) // nil Nav fields: panics inside
	require.ErrorIs(t, err, ErrNumeric)
}
