package wombatmain

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// CloneState gives a typed copy whose asset map a swap on the clone does not reach. Without one, pathfinder-lib's
// DefaultClonePool falls back to go-clone's Slowly, which stores the clone's pointers without GC write barriers.
func TestCloneStateIsTypedAndIsolated(t *testing.T) {
	var poolEntity entity.Pool
	require.NoError(t, json.Unmarshal([]byte(`{
				"address": "0xa45c0abeef67c363364e0e73832df9986aba3800",
				"type": "wombat-main",
				"timestamp": 1705358001,
				"reserves": [
					"27437517755",
					"104442256607"
				],
				"tokens": [
					{
						"address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
						"decimals": 6,
						"weight": 50,
						"swappable": true
					},
					{
						"address": "0xdac17f958d2ee523a2206206994597c13d831ec7",
						"decimals": 6,
						"weight": 50,
						"swappable": true
					}
				],
				"extra": "{\"paused\":false,\"haircutRate\":20000000000000,\"ampFactor\":250000000000000,\"startCovRatio\":1500000000000000000,\"endCovRatio\":1800000000000000000,\"assetMap\":{\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\":{\"isPause\":false,\"address\":\"0x6966553568634F4225330D559a8783DE7649C7D3\",\"cash\":27437517755009846067811,\"liability\":46154718915224891070477,\"underlyingTokenDecimals\":6,\"relativePrice\":null},\"0xdac17f958d2ee523a2206206994597c13d831ec7\":{\"isPause\":false,\"address\":\"0x752945079a0446AA7efB6e9E1789751cDD601c95\",\"cash\":104442256607693995284288,\"liability\":69617497322874416078864,\"underlyingTokenDecimals\":6,\"relativePrice\":null}}}"
			}`), &poolEntity))
	sim, err := NewPoolSimulator(poolEntity)
	require.NoError(t, err)

	cloned := sim.CloneState()
	require.NotNil(t, cloned, "CloneState must not leave the clone to go-clone")
	require.IsType(t, sim, cloned)
	testutil.TestCloneState(t, sim, pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Amount: bignumber.NewBig10("1000000000")},
		TokenOut:      "0xdac17f958d2ee523a2206206994597c13d831ec7",
	}, nil)
}
