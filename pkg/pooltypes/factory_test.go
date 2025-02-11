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
	excludedPoolTypes := []string{
		"curve-lending", // not implemented
		"ambient",       // aevm
		"maverick-v2",   // aevm
		"uniswap-v4",    // aevm
	}
	var poolTypesMap map[string]string
	assert.NoError(t, mapstructure.Decode(PoolTypes, &poolTypesMap))
	poolTypes := lo.OmitByValues(poolTypesMap, excludedPoolTypes)

	for _, poolType := range poolTypes {
		t.Run(poolType, func(t *testing.T) {
			got := pool.Factory(poolType)
			assert.NotNil(t, got)
		})
	}
}

func TestPoolListerFactory(t *testing.T) {
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
		"etherfi-ebtc", "uniswap-v4"}

	for _, poolLister := range poolListers {
		t.Run(poolLister, func(t *testing.T) {
			got := poollist.Factory(poolLister)
			assert.NotNil(t, got)
		})
	}
}

func TestPoolTrackerFactory(t *testing.T) {
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
		"etherfi-ebtc", "uniswap-v4"}
	t.Logf("%#v", poolTrackers)

	for _, poolTracker := range poolTrackers {
		t.Run(poolTracker, func(t *testing.T) {
			got := pooltrack.Factory(poolTracker)
			assert.NotNil(t, got)
		})
	}
}
