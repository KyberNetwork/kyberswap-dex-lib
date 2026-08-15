package integral

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// Alandale is an Algebra Integral deployment on Robinhood Chain (4663). Its
// plugin is BasePlugin V1: feeConfig() exists, feeZeroToOne() / feeOneToZero() /
// s_feeFactors() do not.
//
// This fixture is a real USDG/LUTE pool state captured from that chain. It is
// frozen deliberately — a test that queried Alandale live would start failing in
// this repo the moment that deployment changed, went quiet, or went away, which
// is not a dependency this repo should carry.
//
// What it pins:
//
//  1. the two absent selectors leave DynamicFeeConfig.ZeroToOne/OneToZero at 0,
//     which is the sentinel beforeSwapV1 tests before falling through to the
//     adaptive sigmoid path. Zero is the correct value here, not a failed read.
//  2. Alpha1/Alpha2 are non-zero, so it does not take the flat BaseFee branch.
//  3. no sliding-fee block, since this is a V1 plugin.
//  4. the resulting quote, exact to the wei.
//
// When captured, the simulator agreed with an on-chain QuoterV2 call on the same
// pool to the wei.
const alandaleFixture = `{"address":"0xad8d8c09ab39bbf7e65f4b6896860d956dc8580e","exchange":"alandale","type":"algebra-integral","timestamp":1786808397,"reserves":["12144110932","4791606548818582251351530"],"tokens":[{"address":"0x5fc5360d0400a0fd4f2af552add042d716f1d168","symbol":"USDG","decimals":6,"swappable":true},{"address":"0xd1e861cc5eee7ea88649206b74504d78ccd7aeea","symbol":"LUTE","decimals":18,"swappable":true}],"extra":"{\"liq\":\"288232250710633324\",\"gS\":{\"price\":\"1543378334770342101239018989226542354\",\"tick\":335714,\"lF\":111,\"pC\":193,\"cF\":1000,\"un\":true},\"ticks\":[{\"Index\":-887220,\"LiquidityGross\":\"27767555552071542\",\"LiquidityNet\":\"27767555552071542\"},{\"Index\":299340,\"LiquidityGross\":\"260464695158561782\",\"LiquidityNet\":\"260464695158561782\"},{\"Index\":339480,\"LiquidityGross\":\"23596931931634372\",\"LiquidityNet\":\"-23596931931634372\"},{\"Index\":377580,\"LiquidityGross\":\"260464695158561782\",\"LiquidityNet\":\"-260464695158561782\"},{\"Index\":887220,\"LiquidityGross\":\"4170623620437170\",\"LiquidityNet\":\"-4170623620437170\"}],\"tS\":60,\"tP\":{\"0\":{\"init\":true,\"ts\":1786116633,\"vo\":\"0\",\"tick\":338473,\"avgT\":338473},\"80\":{\"init\":true,\"ts\":1786703106,\"cum\":198105211634,\"vo\":\"155744678405\",\"tick\":335742,\"avgT\":335783,\"wsI\":71},\"81\":{\"init\":true,\"ts\":1786722717,\"cum\":204689310719,\"vo\":\"155759431262\",\"tick\":335735,\"avgT\":335734,\"wsI\":77},\"82\":{\"init\":true,\"ts\":1786726392,\"cum\":205922982494,\"vo\":\"155765312407\",\"tick\":335693,\"avgT\":335732,\"wsI\":77},\"83\":{\"init\":true,\"ts\":1786805220,\"cum\":232385147954,\"vo\":\"155802282055\",\"tick\":335695,\"avgT\":335696,\"wsI\":80},\"84\":{\"vo\":\"0\"},\"85\":{\"vo\":\"0\"}},\"vo\":{\"init\":true,\"tpIdx\":83,\"lastTs\":1786805220},\"dF\":{\"a1\":2900,\"a2\":12000,\"b1\":360,\"b2\":60000,\"g1\":59,\"g2\":8500,\"bF\":100}}","staticExtra":"{\"pluginV2\":false}"}`

func TestPoolSimulator_Alandale_BasePluginV1(t *testing.T) {
	// The package pins blockTimestamp for its fixtures; this one was captured at a
	// different moment, so it needs its own. Not parallel: the hook is package-level.
	origBlockTimestamp := blockTimestamp
	blockTimestamp = func() uint32 { return 1786808397 }
	defer func() { blockTimestamp = origBlockTimestamp }()

	var ep entity.Pool
	require.NoError(t, json.Unmarshal([]byte(alandaleFixture), &ep))

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(ep.Extra), &extra))

	// (1) missing selectors -> 0 sentinel, not a corrupted read
	require.NotNil(t, extra.DynamicFee)
	require.Zero(t, extra.DynamicFee.ZeroToOne)
	require.Zero(t, extra.DynamicFee.OneToZero)
	// (2) adaptive path, not the flat baseFee branch
	require.NotZero(t, extra.DynamicFee.Alpha1)
	require.NotZero(t, extra.DynamicFee.Alpha2)
	// (3) V1 plugin: no sliding fee, but the volatility oracle is required and live
	require.Nil(t, extra.SlidingFee)
	require.NotNil(t, extra.VolatilityOracle)
	require.True(t, extra.VolatilityOracle.IsInitialized)

	ps, err := NewPoolSimulator(ep)
	require.NoError(t, err)

	// (4) 1000 USDG -> LUTE, exact
	res, err := ps.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: ep.Tokens[0].Address, Amount: bignumber.NewBig10("1000000000")},
		TokenOut:      ep.Tokens[1].Address,
	})
	require.NoError(t, err)
	require.Equal(t, "355416851774847704658559", res.TokenAmountOut.Amount.String())
	require.Equal(t, "111000", res.Fee.Amount.String())
}
