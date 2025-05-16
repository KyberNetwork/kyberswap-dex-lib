package pooltypes

import (
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	_ "github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

func TestPoolFactory(t *testing.T) {
	t.Parallel()
	excludedPoolTypes := []string{
		"ambient",     // private
		"maverick-v2", // private
		"kyber-pmm",   // private
		"pmm-1",       // private
		"pmm-2",       // private
	}
	var poolTypesMap map[string]string
	assert.NoError(t, mapstructure.Decode(PoolTypes, &poolTypesMap))
	poolTypes := lo.OmitByValues(poolTypesMap, excludedPoolTypes)

	for key, poolType := range poolTypes {
		t.Run(poolType, func(t *testing.T) {
			got := pool.Factory(poolType)
			assert.NotNil(t, got, key)
		})
	}
}

func TestCanCalcAmountIn(t *testing.T) {
	t.Parallel()
	dexes := []string{"algebra-integral", "algebra-v1", "balancer-v2-composable-stable", "balancer-v2-stable",
		"balancer-v2-weighted", "balancer-v3-eclp", "balancer-v3-stable", "balancer-v3-weighted", "bancor-v3",
		"curve-compound", "curve-lending", "curve-llamma", "curve-stable-meta-ng", "curve-stable-ng",
		"curve-stable-plain", "curve-tricrypto-ng", "curve-twocrypto-ng", "deltaswap-v1", "dodo-classical", "dystopia",
		"ekubo", "euler-swap", "fluid-dex-t1", "hashflow-v3", "iziswap", "limit-order", "liquiditybook-v21",
		"maverick-v1", "muteswitch", "pancake-v3", "pearl", "ramses", "ringswap", "sky-psm", "slipstream", "solidly-v2",
		"solidly-v3", "swap-x-v2", "syncswap-classic", "syncswap-stable", "syncswapv2-classic", "syncswapv2-stable",
		"uniswap-v1", "uniswap-v2", "uniswap-v4", "uniswapv3", "velodrome", "velodrome-v2", "virtual-fun"}
	for _, tt := range dexes {
		t.Run(tt, func(t *testing.T) {
			assert.Contains(t, pool.CanCalcAmountIn, tt)
		})
	}
}

func TestPoolListerFactory(t *testing.T) {
	t.Parallel()
	poolListers := []string{"uniswap", "uniswapv3", "algebra-v1", "dmm", "velodrome", "velodrome-v2", "velocimeter",
		"muteswitch", "ramses", "ramses-v2", "solidly-v2", "solidly-v3", "platypus", "biswap", "maker-psm", "curve",
		"curve-stable-plain", "curve-stable-ng", "curve-stable-meta-ng", "curve-tricrypto-ng", "curve-twocrypto-ng",
		"oneswap", "saddle", "dodo-classical", "dodo-dpp", "dodo-dsp", "dodo-dvm", "nerve", "synthetix", "dystopia",
		"metavault", "camelot", "lido", "lido-steth", "gmx", "fraxswap", "madmex", "polydex", "iron-stable",
		"limit-order", "syncswap", "syncswapv2-classic", "syncswapv2-stable", "syncswapv2-aqua", "pancake-v3",
		"maverick-v1", "pearl", "iziswap", "wombat", "kokonut-crypto", "woofi-v2", "woofi-v21", "equalizer",
		"mantisswap", "gmx-glp", "swapbased-perp", "usdfi", "vooi", "pol-matic", "liquiditybook-v21",
		"liquiditybook-v20", "smardex", "integral", "fxdx", "uniswap-v1", "uniswap-v2", "quickperps", "balancer-v1",
		"balancer-v2-weighted", "balancer-v2-stable", "balancer-v2-composable-stable", "balancer-v3-stable",
		"balancer-v3-weighted", "velocore-v2-cpmm", "velocore-v2-wombat-stable", "fulcrom", "gyroscope-2clp",
		"gyroscope-3clp", "gyroscope-eclp", "zkera-finance", "bancor-v3", "etherfi-eeth", "etherfi-weeth", "kelp-rseth",
		"rocketpool-reth", "ethena-susde", "maker-savingsdai", "bancor-v21", "nomiswap-stable", "renzo-ezeth",
		"bedrock-unieth", "puffer-pufeth", "swell-rsweth", "swell-sweth", "slipstream", "nuri-v2", "ambient",
		"ether-vista", "maverick-v2", "lite-psm", "mkr-sky", "dai-usds", "fluid-vault-t1", "fluid-dex-t1", "usd0pp",
		"ringswap", "generic-simple-rate", "primeeth", "staderethx", "meth", "ondo-usdy", "deltaswap-v1", "sfrxeth",
		"sfrxeth-convertor", "etherfi-vampire", "algebra-integral", "virtual-fun", "beets-ss", "swap-x-v2",
		"etherfi-ebtc", "uniswap-v4", "sky-psm", "honey", "curve-llamma", "curve-lending", "balancer-v3-eclp", "ekubo",
		"erc4626", "hyeth"}

	for _, poolLister := range poolListers {
		t.Run(poolLister, func(t *testing.T) {
			got := poollist.Factory(poolLister)
			assert.NotNil(t, got)
		})
	}
}

func TestPoolTrackerFactory(t *testing.T) {
	t.Parallel()
	poolTrackers := []string{"uniswap", "uniswapv3", "algebra-v1", "dmm", "velodrome", "velodrome-v2", "velocimeter",
		"muteswitch", "ramses", "ramses-v2", "solidly-v2", "solidly-v3", "dodo-classical", "dodo-dpp", "dodo-dsp",
		"dodo-dvm", "biswap", "platypus", "maker-psm", "curve", "curve-stable-plain", "curve-stable-ng",
		"curve-stable-meta-ng", "curve-tricrypto-ng", "curve-twocrypto-ng", "oneswap", "saddle", "nerve", "dystopia",
		"synthetix", "metavault", "camelot", "lido", "lido-steth", "gmx", "fraxswap", "madmex", "polydex",
		"iron-stable", "limit-order", "syncswap", "syncswapv2-classic", "syncswapv2-stable", "syncswapv2-aqua",
		"pancake-v3", "maverick-v1", "pearl", "iziswap", "kokonut-crypto", "wombat", "woofi-v2", "woofi-v21",
		"equalizer", "mantisswap", "gmx-glp", "swapbased-perp", "usdfi", "vooi", "pol-matic", "liquiditybook-v21",
		"liquiditybook-v20", "smardex", "integral", "fxdx", "uniswap-v1", "uniswap-v2", "quickperps", "balancer-v1",
		"fulcrom", "balancer-v2-weighted", "balancer-v2-stable", "balancer-v2-composable-stable", "balancer-v3-stable",
		"balancer-v3-weighted", "velocore-v2-cpmm", "velocore-v2-wombat-stable", "gyroscope-2clp", "gyroscope-3clp",
		"gyroscope-eclp", "zkera-finance", "bancor-v3", "etherfi-eeth", "etherfi-weeth", "kelp-rseth",
		"rocketpool-reth", "ethena-susde", "maker-savingsdai", "bancor-v21", "nomiswap-stable", "renzo-ezeth",
		"bedrock-unieth", "puffer-pufeth", "swell-rsweth", "swell-sweth", "slipstream", "nuri-v2", "ambient",
		"ether-vista", "maverick-v2", "lite-psm", "mkr-sky", "dai-usds", "fluid-vault-t1", "fluid-dex-t1", "usd0pp",
		"ringswap", "generic-simple-rate", "primeeth", "staderethx", "meth", "ondo-usdy", "deltaswap-v1", "sfrxeth",
		"sfrxeth-convertor", "etherfi-vampire", "algebra-integral", "virtual-fun", "beets-ss", "swap-x-v2",
		"etherfi-ebtc", "uniswap-v4", "sky-psm", "honey", "curve-llamma", "curve-lending", "balancer-v3-eclp", "ekubo",
		"erc4626", "hyeth"}
	t.Logf("%#v", poolTrackers)

	for _, poolTracker := range poolTrackers {
		t.Run(poolTracker, func(t *testing.T) {
			got := pooltrack.Factory(poolTracker)
			assert.NotNil(t, got)
		})
	}
}
