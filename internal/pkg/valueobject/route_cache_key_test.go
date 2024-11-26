package valueobject

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouteCacheKey_String(t *testing.T) {
	t.Run("it should return correct key", func(t *testing.T) {
		key := RouteCacheKey{
			TokenIn:       "0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be",
			TokenOut:      "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
			SaveGas:       true,
			CacheMode:     RouteCacheModePoint,
			AmountIn:      "5000000000000000000000",
			Dexes:         []string{"gmx", "uniswap"},
			GasInclude:    true,
			ExcludedPools: []string{"0x"},
		}

		assert.Equal(t, key.String("prefix"), "prefix:0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be-0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7:1:token:5000000000000000000000:gmx-uniswap:1:0x")
	})
}

func TestRouteCacheKey_Hash(t *testing.T) {
	t.Run("it should return same hash", func(t *testing.T) {
		key1 := RouteCacheKey{
			TokenIn:       "0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be",
			TokenOut:      "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
			SaveGas:       true,
			CacheMode:     RouteCacheModePoint,
			AmountIn:      "5000000000000000000000",
			Dexes:         []string{"gmx", "uniswap"},
			GasInclude:    true,
			ExcludedPools: []string{"0x1", "0x2"},
		}

		key2 := RouteCacheKey{
			TokenIn:       "0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be",
			TokenOut:      "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
			SaveGas:       true,
			CacheMode:     RouteCacheModePoint,
			AmountIn:      "5000000000000000000000",
			Dexes:         []string{"uniswap", "gmx"},
			GasInclude:    true,
			ExcludedPools: []string{"0x2", "0x1"},
		}

		assert.Equal(t, key1.Hash("prefix"), key2.Hash("prefix"))
	})
}

// goos: linux
// goarch: amd64
// pkg: github.com/KyberNetwork/router-service/internal/pkg/valueobject
// cpu: AMD Ryzen 7 5800H with Radeon Graphics
// BenchmarkRouteCacheKey_Hash
// BenchmarkRouteCacheKey_Hash-16 (commit d9b1181)  	  922318	      1331   ns/op
// BenchmarkRouteCacheKey_Hash-16 (current)         	 1999945	       521.9 ns/op
func BenchmarkRouteCacheKey_Hash(b *testing.B) {
	key := RouteCacheKey{
		TokenIn:       "0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be",
		TokenOut:      "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
		SaveGas:       true,
		CacheMode:     RouteCacheModePoint,
		AmountIn:      "5000000000000000000000",
		Dexes:         []string{"lido-steth", "curve", "synapse", "maker-psm", "lido", "fraxswap", "maverick-v1", "wombat", "pol-matic", "smardex", "traderjoe-v21", "kyberswap", "kyberswap-static", "uniswap", "sushiswap", "shibaswap", "defiswap", "verse", "pancake", "crowdswap-v2", "balancer-v2-weighted", "balancer-v2-stable", "balancer-v2-composable-stable", "balancer-v1", "solidly-v3", "uniswapv3", "sushiswap-v3", "pancake-v3", "wagmi", "gyroscope-2clp", "gyroscope-3clp", "gyroscope-eclp", "blueprint", "curve-stable-plain", "etherfi-eeth", "etherfi-weeth", "rocketpool-reth", "ethena-susde", "maker-savingsdai", "bancor-v3", "curve-stable-ng", "kyberswap-limit-order-v2", "curve-tricrypto-ng", "curve-stable-meta-ng", "renzo-ezeth", "swell-rsweth", "swell-sweth", "bedrock-unieth", "puffer-pufeth", "swaap-v2", "dodo-classical", "dodo-dpp", "dodo-dsp", "dodo-dvm", "kyber-pmm", "native-v1", "ambient", "ether-vista", "maverick-v2", "lite-psm", "mkr-sky", "dai-usds", "curve-twocrypto-ng", "usd0pp", "integral", "fluid-vault-t1", "hashflow-v3", "uniswap-v1", "fluid-dex-t1", "clipper", "bebop", "wbeth", "oeth", "primeeth", "staderethx", "frxeth", "meth", "deltaswap-v1", "sfrxeth-convertor", "sfrxeth", "ondo-usdy", "ringswap"},
		GasInclude:    true,
		ExcludedPools: []string{"0x3dabc75ffd70695852da101b0c0d018781320551"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = key.Hash("ethereum")
	}
}
