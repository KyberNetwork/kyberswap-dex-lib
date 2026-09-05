package b20

import (
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// TestDirectProtocolMatch_ReferenceLaunch is dex-verify step 4: the local simulator's
// CalcAmountOut must agree with the deployed protocol's own quote source (Uniswap's
// canonical V4Quoter, which executes the real LaunchHook via PoolManager.unlock) for
// the reference pool used throughout this integration. Pool state (slot0, liquidity,
// the single seeded tick band, hook config) was read directly via cast against a
// Tenderly vnet fork of Base at block 50429052 on 2026-08-25 (see output/verify.md) --
// hand-built here rather than via PoolTracker.BootstrapPoolState, which requires a
// keyed thegraph.com subgraph client not available in this environment; production
// bootstrap goes through the tracker, this test only needs the resulting state shape.
//
// maxDriftWei documents an inherited, non-b20-specific finding: the "buy" direction
// (fee applied pre-swap via BeforeSwap, quote=input) shows a small absolute drift
// (4-17 wei observed across outputs spanning 1e22-1e25, i.e. NOT proportional to
// size) against the deployed quoter, traced to the shared v3-derived tick-crossing
// math's fixed-point sqrtPrice arithmetic -- not this hook's fee math, which is
// exact-floor mulDiv and matches on-chain bit-for-bit (confirmed by the "sell"
// direction, fee applied post-swap via AfterSwap, being bit-exact at every size
// tested, including 1e21 tokens sold). Bound kept as an absolute wei count, not a
// relative tolerance, per the "not proportional to size" finding.
const maxDriftWei = 20

func assertWithinDrift(t *testing.T, want string, got *big.Int) {
	t.Helper()
	wantBI, ok := new(big.Int).SetString(want, 10)
	require.True(t, ok)
	diff := new(big.Int).Sub(wantBI, got)
	diff.Abs(diff)
	require.LessOrEqualf(t, diff.Int64(), int64(maxDriftWei),
		"want=%s got=%s diff=%s exceeds the documented inherited-precision tolerance", wantBI, got, diff)
}

func TestDirectProtocolMatch_ReferenceLaunch(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	poolID := "0x68d39022eee9e18f82fe929f70dd8d2009e442e6d80c6c1ffc170c32b7d3b671"
	hook := "0x985c14baa2A18316ffDA0AefB3a632faDFCA2acc"
	token := "0xb200000000000000000000216203b40c83837b7d"
	native := "0x0000000000000000000000000000000000000000"

	staticExtra := `{"0x0":[true,false],"fee":0,"tS":200,"hooks":"` + hook +
		`","uR":"0x6fF5693b99212Da76ad316178A184AB56D299b43","pm2":"0x000000000022D473030F116dDEE9F6B43aC78BA3","mc3":"0xcA11bde05977b3631167028862bE2a173976CA11"}`

	// slot0/liquidity/tick band read live via `cast call` against StateView
	// (0xA3c0c9b65baD0b08107Aa264b0f3dB444b867A71) on the vnet; the seeded band
	// itself (tickLower=-887200, tickUpper=199200) comes from the launch tx's
	// ModifyLiquidity log (0x36b76594...cf636).
	extra := `{"liquidity":34511678704448028213888,"sqrtPriceX96":1557825037365388960186678851882063,"tickSpacing":200,"tick":197739,` +
		`"ticks":[{"index":-887200,"liquidityGross":34511678704448028213888,"liquidityNet":34511678704448028213888},` +
		`{"index":199200,"liquidityGross":34511678704448028213888,"liquidityNet":-34511678704448028213888}],` +
		`"hX":{"t0":false,"bf":100,"as":9900,"aw":16,"lt":1786986263}}`

	pool := entity.Pool{
		Address:  poolID,
		Exchange: string(valueobject.ExchangeUniswapV4B20),
		Type:     uniswapv4.DexType,
		SwapFee:  0,
		Tokens:   []*entity.PoolToken{{Address: native, Swappable: true}, {Address: token, Swappable: true}},
		// Reserves are only a swap-size sanity cap here (real per-pool balances live in
		// the shared PoolManager vault, not queryable per-pool via balanceOf) -- large
		// enough to not clip any of this grid's input amounts.
		Reserves:    entity.PoolReserves{"1000000000000000000000000", "1000000000000000000000000000000"},
		StaticExtra: staticExtra,
		Extra:       extra,
	}

	// The pool's launchTime (1786986263) is long past for all these fixtures (recorded
	// 2026-08-25), so totalFeeBps resolves to the flat baseFeeBps=100 regardless of
	// NowFn -- pinned explicitly anyway so this test's outcome never depends on another
	// test file's leftover package-level NowFn override.
	orig := NowFn
	NowFn = func() int64 { return time.Now().Unix() }
	defer func() { NowFn = orig }()

	sim, err := uniswapv4.NewPoolSimulator(pool, valueobject.ChainIDBase)
	require.NoError(t, err)

	buyCases := []struct {
		name, amountWei, wantOut string
	}{
		{"buy 1e14 wei ETH", "100000000000000", "38272681478685449659873"},
		{"buy 1e15 wei ETH", "1000000000000000", "382532639159565385084466"},
		{"buy 1e16 wei ETH", "10000000000000000", "3806016647895321434614353"},
		{"buy 1e17 wei ETH", "100000000000000000", "36231260193184936831706671"},
	}
	for _, c := range buyCases {
		t.Run(c.name, func(t *testing.T) {
			amt, ok := new(big.Int).SetString(c.amountWei, 10)
			require.True(t, ok)
			res, err := sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{Token: native, Amount: amt},
				TokenOut:      token,
			})
			require.NoError(t, err)
			assertWithinDrift(t, c.wantOut, res.TokenAmountOut.Amount)
		})
	}

	sellCases := []struct {
		name, amountToken, wantOut string
	}{
		{"sell 1 token", "1000000000000000000", "2560689982"},
		{"sell 1000 tokens", "1000000000000000000000", "2560686211853"},
	}
	for _, c := range sellCases {
		t.Run(c.name, func(t *testing.T) {
			amt, ok := new(big.Int).SetString(c.amountToken, 10)
			require.True(t, ok)
			res, err := sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{Token: token, Amount: amt},
				TokenOut:      native,
			})
			require.NoError(t, err)
			// Bit-exact at every size tested -- the AfterSwap (post-swap, fee-on-output)
			// path doesn't touch the pre-swap sqrtPrice math that produces the buy-side drift.
			require.Equal(t, c.wantOut, res.TokenAmountOut.Amount.String())
		})
	}
}
