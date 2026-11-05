package twocryptong

import (
	"math"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// The three Ethereum YBTwocryptoPolicy contracts (WBTC 0x0436c064, cbBTC 0x038f549d, tBTC 0xe99ee179) at blocks
// 26307080, 26306780 and 26306480: state(), the immutables and get_price_scale() as the chain returned them
// (mevblocker eth_call). The cbBTC policy sits inside its deadband and returns price_scale.
func TestPolicyState_PriceScaleMatchesChain(t *testing.T) {
	u := uint256.MustFromDecimal
	policy := func(lastUpdate int64, lastPrices, fast, slow, priceScale string) *PolicyState {
		return &PolicyState{
			LastUpdateTs: lastUpdate, LastPrices: u(lastPrices), FastEma: u(fast), SlowEma: u(slow),
			PriceScale: u(priceScale), FastHalfLife: u("8100"), SlowHalfLife: u("57600"),
			Kappa: u("1300000000000000000"), Deadband: u("800000000000000"), MinCap: u("1200000000000000"),
			MaxCap: u("5400000000000000"),
		}
	}
	for _, c := range []struct {
		name  string
		at    int64
		state *PolicyState
		want  string
	}{
		{"WBTC 26307080", 1793381459, policy(1793379071, "77646391836179594649312", "77634216023848666491693",
			"79013462006129642748426", "72156952696539490158287"), "73595040763781522197137"},
		{"cbBTC 26307080", 1793381459, policy(1793379683, "77621960447248580232137", "77679624759137754326123",
			"79029975712695655203686", "77286628838017654881551"), "77286628838017654881551"},
		{"tBTC 26307080", 1793381459, policy(1793377823, "77615373230567813312479", "77690241909311081921498",
			"79132535032160375615465", "71980553859879763736828"), "73924028814096517357718"},
		{"WBTC 26306780", 1793377847, policy(1793374823, "77528790387276542290251", "77665690690261708925626",
			"79089753458498862753192", "72156952696539490158287"), "73862743058285683705627"},
		{"cbBTC 26306780", 1793377847, policy(1793377223, "77621960477063707436537", "77693136024148270097422",
			"79072280502496188202860", "77286628838017654881551"), "77286628838017654881551"},
		{"tBTC 26306780", 1793377847, policy(1793377823, "77615373230567813312479", "77690241909311081921498",
			"79132535032160375615465", "71980553859879763736828"), "72422514460579425486168"},
		{"WBTC 26306480", 1793374235, policy(1793371031, "77528790387675124597323", "77718169049328438108854",
			"79162633820098255609174", "72156952696539490158287"), "73938507858617050170292"},
		{"cbBTC 26306480", 1793374235, policy(1793373023, "77621960477063707436537", "77723918273658171044000",
			"79147466572998736904980", "77286628838017654881551"), "77286628838017654881551"},
		{"tBTC 26306480", 1793374235, policy(1793373623, "77615373230567813312479", "77722621377315926585135",
			"79211186248566421623652", "71980553859879763736828"), "72669407760318813075788"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.state.PriceScaleAt(c.at)
			require.NoError(t, err)
			assert.Equal(t, c.want, got.Dec())
		})
	}
}

// Chain quotes below come from an anvil fork of Ethereum at block 26307080: get_dy(probe), then a real
// exchange(tiny) one second after the fork block, then get_dy(probe) again. They pin what the pool quotes
// after a trade, which depends on price_oracle as the contract computes it at the trade's time.

// crvUSD/cbBTC as the pool tracker stored it 3.3 hours before the trade; nothing traded in between, so only
// price_oracle() has moved (toward last_prices). MaTime 866 is packed_rebalancing_params at the fork block;
// the ma_time() view says 601. PriceOracleTimestamp is the stored state's timestamp.
const idlePoolEntity = `{"address":"0x83f24023d15d835a213df24fd309c47dab5beb32","exchange":"curve-twocrypto-ng","type":"curve-twocrypto-ng","timestamp":1793369701,"reserves":["10492728068291092625890044","14187434464"],"tokens":[{"address":"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e","symbol":"crvUSD","decimals":18,"swappable":true},{"address":"0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf","symbol":"cbBTC","decimals":8,"swappable":true}],"extra":"{\"InitialA\":\"90000\",\"InitialGamma\":\"1000000000000000\",\"FutureA\":\"90000\",\"FutureGamma\":\"1000000000000000\",\"FutureAGammaTime\":1758724751,\"D\":\"21518122469275311037783025\",\"PriceScale\":[\"77717050847445574608404\"],\"PriceOracle\":[\"77467697907118841752543\"],\"LastPrices\":[\"77332378669606361103093\"],\"PriceOracleTimestamp\":1793369701,\"MaTime\":\"866\",\"FeeGamma\":\"3000000000000000\",\"MidFee\":\"100000000\",\"OutFee\":\"100000000\",\"LpSupply\":\"35281142363187041197032\",\"XcpProfit\":\"1187543877622588755\",\"VirtualPrice\":\"1093889829990635709\",\"AllowedExtraProfit\":\"100000000\",\"AdjustmentStep\":\"100000000000\",\"UseCustomMath\":true}","staticExtra":"{\"IsNativeCoins\":[false,false]}","blockNumber":26306102}`

func quoteAfterTrade(t *testing.T, p entity.Pool, in, out int, probe, tiny string) (pre, post *big.Int) {
	t.Helper()
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)
	tokenIn, tokenOut := sim.Info.Tokens[in], sim.Info.Tokens[out]
	quote := func(s *PoolSimulator, amount *big.Int) *pool.CalcAmountOutResult {
		res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amount},
			TokenOut:      tokenOut,
		})
		require.NoError(t, err)
		return res
	}
	probeAmount, _ := new(big.Int).SetString(probe, 10)
	tinyAmount, _ := new(big.Int).SetString(tiny, 10)

	pre = quote(sim, probeAmount).TokenAmountOut.Amount
	clone := sim.CloneState().(*PoolSimulator)
	trade := quote(clone, tinyAmount)
	clone.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: tokenIn, Amount: tinyAmount},
		TokenAmountOut: *trade.TokenAmountOut,
		SwapInfo:       trade.SwapInfo,
	})
	post = quote(clone, probeAmount).TokenAmountOut.Amount

	assert.Equal(t, pre.String(), quote(sim, probeAmount).TokenAmountOut.Amount.String(),
		"the trade on the clone must not move the original")
	return pre, post
}

// bpsOff is |got - want| / want in basis points.
func bpsOff(t *testing.T, want, got string) float64 {
	t.Helper()
	w, ok := new(big.Float).SetString(want)
	require.True(t, ok)
	g, ok := new(big.Float).SetString(got)
	require.True(t, ok)
	off, _ := new(big.Float).Quo(new(big.Float).Sub(g, w), w).Float64()
	return math.Abs(off) * 1e4
}

// The chain carries price_oracle from the pool's last trade to the swap in one step; the simulator carries
// the value read later on. The two agree exactly in one direction and within 1e-5 bps in the other, against
// a 3.1 bps miss before.
func TestPriceOracleCarriedForward_IdlePool(t *testing.T) {
	var p entity.Pool
	require.NoError(t, json.Unmarshal([]byte(idlePoolEntity), &p))
	defer func(f func() int64) { nowUnix = f }(nowUnix)
	for _, tc := range []struct {
		name        string
		in, out     int
		swapAt      int64
		probe, tiny string
		pre, post   string
		maxBps      float64
	}{
		{"crvUSD to cbBTC", 0, 1, 1793381462, "104927280682910926258", "104927280682910926", "134327", "134446", 0},
		{"cbBTC to crvUSD", 1, 0, 1793381463, "141874", "141", "108617282156010210394", "108520532036121485931", 1e-5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nowUnix = func() int64 { return tc.swapAt }
			pre, post := quoteAfterTrade(t, p, tc.in, tc.out, tc.probe, tc.tiny)
			assert.Equal(t, tc.pre, pre.String(), "get_dy before the trade")
			assert.LessOrEqual(t, bpsOff(t, tc.post, post.String()), tc.maxBps, "get_dy after the trade: %s", post)
		})
	}
}

// A state written before this change has neither field and keeps quoting from the stored oracle.
func TestPriceOracleNotCarriedForward_OldState(t *testing.T) {
	var p entity.Pool
	require.NoError(t, json.Unmarshal([]byte(idlePoolEntity), &p))
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	extra.MaTime, extra.PriceOracleTimestamp = nil, 0
	b, err := json.Marshal(extra)
	require.NoError(t, err)
	p.Extra = string(b)
	defer func(f func() int64) { nowUnix = f }(nowUnix)
	nowUnix = func() int64 { return 1793381462 }

	_, post := quoteAfterTrade(t, p, 0, 1, "104927280682910926258", "104927280682910926")
	assert.Equal(t, "134404", post.String())
}

func TestPriceOracleAt(t *testing.T) {
	var p entity.Pool
	require.NoError(t, json.Unmarshal([]byte(idlePoolEntity), &p))
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)
	readAt := sim.Extra.PriceOracleTimestamp
	stored, last := sim.Extra.PriceOracle, sim.Extra.LastPrices

	at := func(s *PoolSimulator, timestamp int64) ([]uint256.Int, int64) {
		out := make([]uint256.Int, NumTokens-1)
		got, err := s.priceOracleAt(timestamp, out)
		require.NoError(t, err)
		return out, got
	}

	t.Run("at or before the read, the stored value", func(t *testing.T) {
		for _, timestamp := range []int64{readAt - 100, readAt} {
			out, got := at(sim, timestamp)
			assert.Equal(t, readAt, got)
			assert.Equal(t, stored, out)
		}
	})
	t.Run("after the read, between the stored value and last_prices", func(t *testing.T) {
		out, got := at(sim, readAt+300)
		assert.Equal(t, readAt+300, got)
		assert.True(t, out[0].Lt(&stored[0]) && out[0].Gt(&last[0]), "the oracle moves down toward last_prices")
	})
	t.Run("long after the read, last_prices", func(t *testing.T) {
		out, _ := at(sim, readAt+100*866)
		assert.Equal(t, last, out)
	})
	t.Run("capped at twice price_scale", func(t *testing.T) {
		far := sim.CloneState().(*PoolSimulator)
		far.Extra.LastPrices = []uint256.Int{*new(uint256.Int).Mul(&sim.Extra.PriceScale[0], uint256.NewInt(3))}
		out, _ := at(far, readAt+100*866)
		assert.Equal(t, new(uint256.Int).Mul(&sim.Extra.PriceScale[0], uint256.NewInt(2)).Dec(), out[0].Dec())
	})
	t.Run("does not write to the stored value", func(t *testing.T) {
		before := append([]uint256.Int(nil), sim.Extra.PriceOracle...)
		at(sim, readAt+300)
		assert.Equal(t, before, sim.Extra.PriceOracle)
	})
}

func TestMaTimeFromRebalancingParams(t *testing.T) {
	allowedExtraProfit, adjustmentStep := big.NewInt(100000000), big.NewInt(100000000000)
	packed := new(big.Int).Lsh(allowedExtraProfit, 128)
	packed.Or(packed, new(big.Int).Lsh(adjustmentStep, 64))
	packed.Or(packed, big.NewInt(866))
	assert.Equal(t, uint64(866), MaTimeFromRebalancingParams(packed).Uint64())
	assert.True(t, MaTimeFromRebalancingParams(new(big.Int)).IsZero())
}

// crvUSD/WETH 0x6e5492f8, version v2.1.0d (MATH v0.1.0), as the pool tracker stored it at block 26306565. Nothing
// traded before the anvil fork at block 26307080 (D, xcp_profit, price_scale and last_prices match the fork's
// views). Version, LastTimestamp and the donation fields are the pool's views at block 26307080 (mevblocker
// eth_call); totalSupply there equals the stored LpSupply. PriceOracle is set per case to the chain's price_oracle at the trade, read at the trade's time, so the
// oracle is not in question here; MaTime 866 is packed_rebalancing_params at the fork block.
const v21dPoolEntity = `{"address":"0x6e5492f8ea2370844ee098a56dd88e1717e4a9c2","exchange":"curve-twocrypto-ng","type":"curve-twocrypto-ng","timestamp":1793375257,"reserves":["78261053618607372972457","35085801511259898377"],"tokens":[{"address":"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e","symbol":"crvUSD","decimals":18,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"InitialA\":\"25000\",\"InitialGamma\":\"1000000000000000\",\"FutureA\":\"25000\",\"FutureGamma\":\"1000000000000000\",\"FutureAGammaTime\":1767800279,\"D\":\"156850417944366468716557\",\"PriceScale\":[\"2239922110122448820209\"],\"PriceOracle\":[\"2241862324510400716898\"],\"LastPrices\":[\"2236897704637006360940\"],\"FeeGamma\":\"1395000000000000\",\"MidFee\":\"60000000\",\"OutFee\":\"220000000\",\"LpSupply\":\"1564432119144803173996\",\"XcpProfit\":\"1101083691955027841\",\"VirtualPrice\":\"1059211049124097889\",\"AllowedExtraProfit\":\"1000000\",\"AdjustmentStep\":\"5000000000000000\",\"UseCustomMath\":true,\"Version\":\"v2.1.0d\",\"LastTimestamp\":1793375255,\"DonationShares\":\"395782627665194639\",\"DonationDuration\":\"604800\",\"LastDonationReleaseTs\":1793373191,\"DonationProtectionExpiryTs\":1793376611,\"DonationProtectionPeriod\":\"3600\"}","staticExtra":"{\"IsNativeCoins\":[false,false]}","blockNumber":26306565}`

// v21dPoolAt is the pool with price_oracle read at the trade's time, which the simulator then uses as it is.
func v21dPoolAt(t *testing.T, priceOracle string, swapAt int64) *PoolSimulator {
	t.Helper()
	var p entity.Pool
	require.NoError(t, json.Unmarshal([]byte(v21dPoolEntity), &p))
	var extra map[string]any
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	extra["PriceOracle"] = []string{priceOracle}
	extra["PriceOracleTimestamp"] = swapAt
	extra["MaTime"] = "866"
	b, err := json.Marshal(extra)
	require.NoError(t, err)
	p.Extra = string(b)
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)
	return sim
}

// A trade on 0x6e5492f8 rebalances: norm = |price_oracle/price_scale - 1| = 1.3485e15 and v2.1.0d steps by
// min(adjustment_step, norm/5) = 2.697e14, where this package takes the max (5e15 >= norm, so no step at all).
// v2.1.0d also grows xcp_profit by the virtual-price gain, not by the ratio. Chain values are the pool's views
// after a real exchange on the fork. The crvUSD-in trade's fork moved crvUSD
// out of the pool first, which may touch donation accounting, so only its price_scale is held here; the
// WETH-in trade's fork moved nothing out of the pool.
func TestPoolV21d_RebalanceMatchesChain(t *testing.T) {
	defer func(f func() int64) { nowUnix = f }(nowUnix)
	for _, tc := range []struct {
		name                      string
		in, out                   int
		swapAt                    int64
		tiny                      string
		priceOracle               string
		priceScale                string
		xcpProfit, virtualPrice   string
		d, lastPrices             string
		postTradeStateIsUntainted bool
	}{
		{
			name: "WETH to crvUSD", in: 1, out: 0, swapAt: 1793381462, tiny: "350858015112",
			priceOracle:  "2236901533835545908094",
			priceScale:   "2239317994865068238233",
			xcpProfit:    "1101083691987117553",
			virtualPrice: "1059211049156187601",
			d:            "156829234340012197210528",
			lastPrices:   "2237243753742046899680",

			postTradeStateIsUntainted: true,
		},
		{
			name: "crvUSD to WETH", in: 0, out: 1, swapAt: 1793381463, tiny: "782610536186073",
			priceOracle: "2236901529416390554023",
			priceScale:  "2239317993981237166971",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nowUnix = func() int64 { return tc.swapAt }
			sim := v21dPoolAt(t, tc.priceOracle, tc.swapAt)
			tokenIn, tokenOut := sim.Info.Tokens[tc.in], sim.Info.Tokens[tc.out]
			amount, _ := new(big.Int).SetString(tc.tiny, 10)
			res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amount},
				TokenOut:      tokenOut,
			})
			require.NoError(t, err)
			after := sim.CloneState().(*PoolSimulator)
			after.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  pool.TokenAmount{Token: tokenIn, Amount: amount},
				TokenAmountOut: *res.TokenAmountOut,
				SwapInfo:       res.SwapInfo,
			})
			e := after.Extra
			assert.Equal(t, tc.priceOracle, e.PriceOracle[0].Dec(), "price_oracle")
			assert.Equal(t, tc.priceScale, e.PriceScale[0].Dec(), "price_scale")
			if !tc.postTradeStateIsUntainted {
				return
			}
			assert.Equal(t, tc.xcpProfit, e.XcpProfit.Dec(), "xcp_profit")
			assert.Equal(t, tc.virtualPrice, e.VirtualPrice.Dec(), "virtual_price")
			assert.Equal(t, tc.d, e.D.Dec(), "D")
			assert.Equal(t, tc.lastPrices, e.LastPrices[0].Dec(), "last_prices")
		})
	}
}

// v2.1.0d rebalances at most once per block: a second trade in the trade's block keeps price_scale, and the
// state after the first trade carries the burnt donation shares and the block's timestamp.
func TestPoolV21d_RebalancesOncePerBlock(t *testing.T) {
	defer func(f func() int64) { nowUnix = f }(nowUnix)
	const swapAt = 1793381462
	nowUnix = func() int64 { return swapAt }
	sim := v21dPoolAt(t, "2236901533835545908094", swapAt)
	weth, crvUSD := sim.Info.Tokens[1], sim.Info.Tokens[0]
	trade := func(s *PoolSimulator, amount string) *pool.CalcAmountOutResult {
		a, _ := new(big.Int).SetString(amount, 10)
		res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: weth, Amount: a}, TokenOut: crvUSD})
		require.NoError(t, err)
		s.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn: pool.TokenAmount{Token: weth, Amount: a}, TokenAmountOut: *res.TokenAmountOut,
			SwapInfo: res.SwapInfo})
		return res
	}
	after := sim.CloneState().(*PoolSimulator)
	trade(after, "350858015112")
	assert.Equal(t, "2239317994865068238233", after.Extra.PriceScale[0].Dec())
	assert.Equal(t, int64(swapAt), after.Extra.LastTimestamp)
	assert.True(t, after.Extra.LpSupply.Cmp(sim.Extra.LpSupply) <= 0, "burns only")
	assert.Equal(t, new(uint256.Int).Sub(sim.Extra.LpSupply, after.Extra.LpSupply).Dec(),
		new(uint256.Int).Sub(sim.Extra.DonationShares, after.Extra.DonationShares).Dec(),
		"total supply drops by the burnt donation shares")

	second := trade(after, "350858015112").SwapInfo.(SwapInfo)
	assert.Equal(t, "2239317994865068238233", second.PriceScale[0].Dec(), "no second rebalance in the block")
	assert.Equal(t, "2239922110122448820209", sim.Extra.PriceScale[0].Dec(), "the original is untouched")
}

// The earlier v2.1.0d build (0x83f24023, same version string) steps by max(adjustment_step, norm / 5): with its
// donation state at block 26307080 it quotes after a trade as the chain does, as the v2.0.0 rule did before.
func TestPoolV21d_EarlierBuildStepsByMax(t *testing.T) {
	raw := strings.Replace(idlePoolEntity, `\"UseCustomMath\":true}`, `\"UseCustomMath\":true,`+
		`\"Version\":\"v2.1.0d\",\"LastTimestamp\":1793369699,\"DonationShares\":\"9409465428899535405\",`+
		`\"DonationDuration\":\"604800\",\"LastDonationReleaseTs\":1793369699,`+
		`\"DonationProtectionExpiryTs\":1793370834,\"DonationProtectionPeriod\":\"3600\"}`, 1)
	require.NotEqual(t, idlePoolEntity, raw)
	var p entity.Pool
	require.NoError(t, json.Unmarshal([]byte(raw), &p))
	defer func(f func() int64) { nowUnix = f }(nowUnix)

	nowUnix = func() int64 { return 1793381462 }
	pre, post := quoteAfterTrade(t, p, 0, 1, "104927280682910926258", "104927280682910926")
	assert.Equal(t, "134327", pre.String(), "v2.1.0d's fee before the trade")
	assert.Equal(t, "134446", post.String())
	nowUnix = func() int64 { return 1793381463 }
	pre, post = quoteAfterTrade(t, p, 1, 0, "141874", "141")
	assert.Equal(t, "108617282156010210394", pre.String(), "v2.1.0d's fee before the trade")
	assert.LessOrEqual(t, bpsOff(t, "108520532036121485931", post.String()), 1e-5)
}

// v3PostTradeCase is a trade on a Yield Basis v3.0.0 BTC pool on an anvil fork of Ethereum at block 26326740
// . Pool is the tracker's state read on the fork at the block
// before the trade, with PriceOracleTimestamp set to that block's time; Now is the trade's block time, and the
// trade is its block's first. After holds the pool's and the policy's views after the trade, and Quotes the
// pool's get_dy then ([i, j, dx, dy]). That fork set the pools' fee parameters to reserved_profit_fraction 5e9
// and admin_fee 5e9 (3e9 and 0 on chain) through the factory admin, so the admin share of the fee is booked; the
// last case is from a fork with the chain's parameters. Source names the recorded trade.
type v3PostTradeCase struct {
	Name       string
	Source     string
	Pool       entity.Pool
	Now        int64
	Dir        int
	Dx, Dy     string
	Rebalanced bool
	After      map[string]string
	Quotes     [][4]string
}

func loadV3PostTradeCases(t *testing.T) []v3PostTradeCase {
	t.Helper()
	b, err := os.ReadFile("testdata/yb_v3_posttrade.json")
	require.NoError(t, err)
	var cases []v3PostTradeCase
	require.NoError(t, json.Unmarshal(b, &cases))
	require.Len(t, cases, 6)
	return cases
}

func bigOf(t *testing.T, s string) *big.Int {
	t.Helper()
	x, ok := new(big.Int).SetString(s, 10)
	require.True(t, ok, s)
	return x
}

func (c *v3PostTradeCase) trade(t *testing.T, sim *PoolSimulator) *pool.CalcAmountOutResult {
	t.Helper()
	nowUnix = func() int64 { return c.Now }
	tokens := c.Pool.Tokens
	in := pool.TokenAmount{Token: tokens[c.Dir].Address, Amount: bigOf(t, c.Dx)}
	r, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: in, TokenOut: tokens[1-c.Dir].Address})
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: in, TokenAmountOut: *r.TokenAmountOut, SwapInfo: r.SwapInfo})
	return r
}

// The state after a v3.0.0 trade (admin fee booked, tweak_price with the policy's target, donation burn, and the
// policy's update_pool_state) matches the chain to the wei, and so do the quotes on it.
func TestPoolV3_PostTradeMatchesChain(t *testing.T) {
	defer func(f func() int64) { nowUnix = f }(nowUnix)
	for _, c := range loadV3PostTradeCases(t) {
		t.Run(c.Name, func(t *testing.T) {
			sim, err := NewPoolSimulator(c.Pool)
			require.NoError(t, err)
			psBefore := sim.Extra.PriceScale[0].Dec()
			r := c.trade(t, sim)
			assert.Equal(t, c.Dy, r.TokenAmountOut.Amount.String(), "amount out")
			assert.Equal(t, c.Rebalanced, sim.Extra.PriceScale[0].Dec() != psBefore, "rebalanced")

			e := &sim.Extra
			require.NotNil(t, e.PolicyState)
			for name, got := range map[string]string{
				"balances0":                sim.Reserves[0].Dec(),
				"balances1":                sim.Reserves[1].Dec(),
				"D":                        e.D.Dec(),
				"price_scale":              e.PriceScale[0].Dec(),
				"last_prices":              e.LastPrices[0].Dec(),
				"price_oracle":             e.PriceOracle[0].Dec(),
				"virtual_price":            e.VirtualPrice.Dec(),
				"xcp_profit":               e.XcpProfit.Dec(),
				"lp_xcp_profit":            e.LpXcpProfit.Dec(),
				"totalSupply":              e.LpSupply.Dec(),
				"donation_shares":          e.DonationShares.Dec(),
				"last_donation_release_ts": strconv.FormatInt(e.LastDonationReleaseTs, 10),
				"last_timestamp":           strconv.FormatInt(e.LastTimestamp, 10),
				"policy.last_update_ts":    strconv.FormatInt(e.PolicyState.LastUpdateTs, 10),
				"policy.last_prices":       e.PolicyState.LastPrices.Dec(),
				"policy.fast_ema":          e.PolicyState.FastEma.Dec(),
				"policy.slow_ema":          e.PolicyState.SlowEma.Dec(),
				"policy.price_scale":       e.PolicyState.PriceScale.Dec(),
			} {
				assert.Equal(t, c.After[name], got, name)
			}
			assert.Equal(t, c.After["balances"+strconv.Itoa(1-c.Dir)], sim.Info.Reserves[1-c.Dir].String(),
				"Info.Reserves")

			for _, q := range c.Quotes {
				i, _ := strconv.Atoi(q[0])
				j, _ := strconv.Atoi(q[1])
				got, err := sim.CloneState().(*PoolSimulator).CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: c.Pool.Tokens[i].Address, Amount: bigOf(t, q[2])},
					TokenOut:      c.Pool.Tokens[j].Address,
				})
				require.NoError(t, err, "quote %v", q)
				assert.Equal(t, q[3], got.TokenAmountOut.Amount.String(), "quote %v", q)
			}
		})
	}
}

// A clone's trade leaves the original's state and quote unchanged.
func TestPoolV3_PostTradeCloneIsolated(t *testing.T) {
	defer func(f func() int64) { nowUnix = f }(nowUnix)
	c := loadV3PostTradeCases(t)[1]
	sim, err := NewPoolSimulator(c.Pool)
	require.NoError(t, err)
	before, err := json.Marshal(sim.Extra)
	require.NoError(t, err)
	c.trade(t, sim.CloneState().(*PoolSimulator))
	after, err := json.Marshal(sim.Extra)
	require.NoError(t, err)
	assert.JSONEq(t, string(before), string(after))
	r := c.trade(t, sim)
	assert.Equal(t, c.Dy, r.TokenAmountOut.Amount.String())
}

// The state after a trade stays refused when it is not known: without the policy's state (an older tracker),
// and when price_oracle is carried from a view read away from last_timestamp and the policy's target lies
// outside the oracle band, where the carried oracle's rounding would set the target. The trade's own quote is
// still given.
func TestPoolV3_PostTradeUnknownStateRefused(t *testing.T) {
	defer func(f func() int64) { nowUnix = f }(nowUnix)
	c := loadV3PostTradeCases(t)[1] // rebalances
	for name, edit := range map[string]func(e *Extra){
		"no policy state":  func(e *Extra) { e.PolicyState = nil },
		"no lp_xcp_profit": func(e *Extra) { e.LpXcpProfit = nil },
		"carried oracle, target outside the band": func(e *Extra) {
			// price_oracle at 83% of the policy's target puts the target just above the band (oracle + 20%).
			target, err := e.PolicyState.PriceScaleAt(c.Now)
			require.NoError(t, err)
			oracle := new(uint256.Int).Mul(target, uint256.NewInt(83))
			e.PriceOracle = []uint256.Int{*oracle.Div(oracle, uint256.NewInt(100))}
			e.PriceOracleTimestamp = c.Now // read at the trade's time: used as is, but not the pool's cache
		},
	} {
		t.Run(name, func(t *testing.T) {
			sim, err := NewPoolSimulator(c.Pool)
			require.NoError(t, err)
			edit(&sim.Extra)
			r := c.trade(t, sim)
			assert.Equal(t, c.Dy, r.TokenAmountOut.Amount.String())
			_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: c.Pool.Tokens[0].Address, Amount: big.NewInt(1e18)},
				TokenOut:      c.Pool.Tokens[1].Address,
			})
			assert.ErrorIs(t, err, ErrStateAfterTrade)
		})
	}
}
