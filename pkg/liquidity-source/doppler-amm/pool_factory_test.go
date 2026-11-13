package doppleramm

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

// Topics of real Base logs from tx 0x2c5bb90f (sell on spot market 0xbf674fb6).
const testMarket = "0xbf674fb68023ec12caf9cceaf1432a04ca932d597899df2e2294a029f8a7ebda"

func testLog(amm string, topics ...string) types.Log {
	l := types.Log{Address: common.HexToAddress(amm)}
	for _, t := range topics {
		l.Topics = append(l.Topics, common.HexToHash(t))
	}
	return l
}

func TestPoolFactoryRoutesMarketEvents(t *testing.T) {
	const amm = "0x533b7ee11f4f9be09191ebf6667abe73c01fe2c2"
	f, err := NewPoolFactory(poolfactory.PoolsDecoderParams[Config]{Cfg: testConfig()})
	require.NoError(t, err)
	ctx := context.Background()

	for name, topic0 := range map[string]string{
		"swap":             "0x07af4e23b8e693b486c551709747254c747c07d7f9b3930a72bb6b0c3ac5af83",
		"liquidity add":    "0xdb56780297c6bee5d6fcac88e28b683fc68cb5d85fea9205cd3ce956e8481e19",
		"liquidity remove": "0x33c06bd624e4eb0d07c026198804af254b48b0a4d48f63a88e7a1e967ab7a983",
	} {
		got, err := f.DecodePoolAddressesFromFactoryLog(ctx, testLog(amm, topic0, testMarket))
		require.NoError(t, err, name)
		require.Equal(t, []string{testMarket}, got, name)
	}

	// ERC6909 Transfer (topic1 is an address) and an unrelated emitter must not route.
	got, _ := f.DecodePoolAddressesFromFactoryLog(ctx, testLog(amm,
		"0x1b3d7edb2e9c0b0e7c525b20aaaef0f5940d2ed71663c7d39266ecafac728859", "0x0", "0x1", testMarket))
	require.Empty(t, got)
	got, _ = f.DecodePoolAddressesFromFactoryLog(ctx, testLog("0x1",
		"0x07af4e23b8e693b486c551709747254c747c07d7f9b3930a72bb6b0c3ac5af83", testMarket))
	require.Empty(t, got)
}
