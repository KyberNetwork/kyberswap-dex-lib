package twocryptong

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	ybCrvUSD = "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e"
	ybWETH   = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
)

// Yield Basis WETH pool 0x656341Ef90b622c6634e0573772FfB7f3669b9f3 (Twocrypto v3.0.0, MATH
// StableswapMath v0.1.1, YBOraclizedPolicy attached) at ethereum block 26356760. The policy's
// stored report was 2004s old (> relax_time 300s), so the on-chain fee was the policy fallback
// fee 1.75e8, not the native mid/out fee (1.36e8 .. 2.82e8). Expected values are pool.get_dy
// at that block.
func ybPoolV3(t *testing.T, mutate func(*Extra)) *PoolSimulator {
	var extra Extra
	extraJSON := `{"InitialA":"50000","InitialGamma":"11111111111","FutureA":"50000","FutureGamma":"11111111111",
"D":"28938780018324405319723192","PriceScale":["2228955360196288804697"],"PriceOracle":["2230536246020201183769"],
"LastPrices":["2229015759977619397980"],"FeeGamma":"4961947600000000","MidFee":"136000000","OutFee":"282000000",
"LpSupply":"290331327017939586975544","XcpProfit":"1098920055000892490","VirtualPrice":"1055615207701318178",
"AdjustmentStep":"100000000","UseCustomMath":true,"PolicyFee":"175000000"}`
	require.NoError(t, json.Unmarshal([]byte(extraJSON), &extra))
	mutate(&extra)
	extraBytes, err := json.Marshal(extra)
	require.NoError(t, err)
	sim, err := NewPoolSimulator(entity.Pool{
		Address: "0x656341ef90b622c6634e0573772ffb7f3669b9f3", Exchange: "curve-twocrypto-ng", Type: DexType,
		Reserves: []string{"14419389231653018033640479", "6514001967067202183561"},
		Tokens: []*entity.PoolToken{
			{Address: ybCrvUSD, Decimals: 18, Swappable: true}, {Address: ybWETH, Decimals: 18, Swappable: true},
		},
		Extra: string(extraBytes), StaticExtra: `{"IsNativeCoins":[false,false]}`,
	})
	require.NoError(t, err)
	return sim
}

var ybV3Cases = []struct{ in, out, amtIn, want string }{
	{ybCrvUSD, ybWETH, "1000000000000000000000", "441292542159488383"},
	{ybCrvUSD, ybWETH, "100000000000000000000000", "44078960083304243416"},
	{ybCrvUSD, ybWETH, "3000000000000000000000000", "1278470531041561629091"},
	{ybWETH, ybCrvUSD, "1000000000000000000", "2187370649395601421760"},
	{ybWETH, ybCrvUSD, "100000000000000000000", "218182627792624168929194"},
	{ybWETH, ybCrvUSD, "1000000000000000000000", "2131753825773942196382449"},
}

func quoteYB(sim *PoolSimulator, in, out, amt string) (string, error) {
	res, err := sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{Token: in, Amount: bigFrom(amt)}, TokenOut: out,
	})
	if err != nil {
		return "", err
	}
	return res.TokenAmountOut.Amount.String(), nil
}

func bigFrom(s string) *big.Int { v, _ := new(big.Int).SetString(s, 10); return v }

// Exact match with the chain needs both the v0.1.1 stableswap math and the policy fee.
func TestYieldBasisV3MatchesChain(t *testing.T) {
	sim := ybPoolV3(t, func(*Extra) {})
	for _, tc := range ybV3Cases {
		got, err := quoteYB(sim, tc.in, tc.out, tc.amtIn)
		require.NoError(t, err)
		require.Equal(t, tc.want, got, "amtIn=%s", tc.amtIn)
	}
}

func TestYieldBasisV3Guards(t *testing.T) {
	// v0.1.1 is registered as the custom (stableswap) math
	require.True(t, UseCustomMath(common.HexToAddress("0xBfDdF58Cb6ef84e115fF47c10e49A80B2653EA13")))

	// Without the policy fee the native fee applies and the quote moves away from the chain
	sim := ybPoolV3(t, func(e *Extra) { e.PolicyFee = nil })
	got, err := quoteYB(sim, ybCrvUSD, ybWETH, "1000000000000000000000")
	require.NoError(t, err)
	require.NotEqual(t, "441292542159488383", got)

	// v3 pool.vy _assert_balance: a 1M WETH swap leaves max/min xp >= 1000 and reverts on chain
	sim = ybPoolV3(t, func(*Extra) {})
	_, err = quoteYB(sim, ybWETH, ybCrvUSD, "1000000000000000000000000")
	require.ErrorIs(t, err, ErrUnsafeXi)
}
