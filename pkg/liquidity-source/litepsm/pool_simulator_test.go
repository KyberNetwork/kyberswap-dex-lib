package litepsm

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCloneState(t *testing.T) {
	t.Parallel()
	var ep entity.Pool
	require.NoError(t, json.Unmarshal([]byte(`{
		"address": "0xf6e72db5454dd049d0788e411b06cfaf16853042",
		"exchange": "lite-psm",
		"type": "lite-psm",
		"reserves": ["414303017692629270548465826","3850977525217835"],
		"tokens": [
			{"address": "0x6b175474e89094c44da98b954eedeac495271d0f","symbol": "DAI","decimals": 18,"swappable": true},
			{"address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol": "USDC","decimals": 6,"swappable": true}
		],
		"extra": "{}",
		"staticExtra": "{\"poc\":\"0x37305b1cd40574e4c5ce33f8e8306be057fd7341\"}"
	}`), &ep))

	p, err := NewPoolSimulator(ep)
	require.NoError(t, err)

	// DAI in, USDC out (buyGem): 1 DAI → ~1 USDC
	testutil.TestCloneState(t, p, poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{
			Token:  "0x6b175474e89094c44da98b954eedeac495271d0f",
			Amount: new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
		},
		TokenOut: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
	}, nil)
}

// The expected gas is a literal, so a changed constant fails here. Direct: the measured hop gas (buy 62,003,
// sell 74,818). Wrapper: direct + gasWrap 116,098 (178,101 / 190,916), within 1.3% of the measured 176,903 / 192,113.
func TestGasByPath(t *testing.T) {
	t.Parallel()
	staticExtra := func(s StaticExtra) string {
		b, err := json.Marshal(s)
		require.NoError(t, err)
		return string(b)
	}
	const (
		dai  = "0x6b175474e89094c44da98b954eedeac495271d0f"
		usdc = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	)
	for name, tc := range map[string]struct {
		address, dai string
		static       string
		buyGas       int64 // dai/usds -> usdc
		sellGas      int64 // usdc -> dai/usds
	}{
		"direct": {"0xf6e72db5454dd049d0788e411b06cfaf16853042", dai, staticExtra(StaticExtra{}), 62003, 74818},
		"wrapper": {"0xa188eec8f81263234da3622a406892f3d630f98c", "0xdc035d45d973e3ec169d2276ddab16f1e407384f",
			staticExtra(StaticExtra{Dai: &liteDai, InnerPsm: &liteDai}), 178101, 190916},
	} {
		t.Run(name, func(t *testing.T) {
			p, err := NewPoolSimulator(entity.Pool{
				Address:  tc.address,
				Exchange: "lite-psm",
				Reserves: entity.PoolReserves{"1000000000000000000000000", "1000000000000"},
				Tokens: []*entity.PoolToken{
					{Address: tc.dai, Decimals: 18},
					{Address: usdc, Decimals: 6},
				},
				Extra:       "{}",
				StaticExtra: tc.static,
			})
			require.NoError(t, err)

			buy, err := p.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{Token: tc.dai, Amount: big.NewInt(1e18)},
				TokenOut:      usdc,
			})
			require.NoError(t, err)
			require.Equal(t, tc.buyGas, buy.Gas)

			sell, err := p.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{Token: usdc, Amount: big.NewInt(1e6)},
				TokenOut:      tc.dai,
			})
			require.NoError(t, err)
			require.Equal(t, tc.sellGas, sell.Gas)
		})
	}
}
