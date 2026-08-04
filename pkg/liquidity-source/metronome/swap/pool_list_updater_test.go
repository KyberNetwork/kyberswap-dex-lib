package metronomeswap

import (
	"context"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListUpdaterTestSuite struct {
	suite.Suite

	updater *PoolsListUpdater
}

func (ts *PoolListUpdaterTestSuite) SetupTest() {
	rpcClient := ethrpc.New("https://ethereum-rpc.kyberswap.com")
	rpcClient.SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696"))

	cfg := &Config{
		DexID:        DexType,
		ChainID:      valueobject.ChainIDEthereum,
		PoolRegistry: "0x11ead85c679eaf528c9c1fe094bf538db880048a",
	}
	ts.updater = NewPoolsListUpdater(cfg, rpcClient)
}

func (ts *PoolListUpdaterTestSuite) TestGetNewPools_FromScratch() {
	pools, metadataBytes, err := ts.updater.GetNewPools(context.Background(), nil)
	require.NoError(ts.T(), err)
	require.NotEmpty(ts.T(), pools, "PoolRegistry.getPools() is known to return at least the reference pool")

	var foundReference bool
	for _, p := range pools {
		// A registered pool can have as few as 1 synthetic today (observed live for the
		// non-reference pool) — such a pool simply has no swappable pairs (CanSwapTo returns
		// empty for its only token), which is valid state, not a lister bug.
		require.GreaterOrEqualf(ts.T(), len(p.Tokens), 1, "pool %s must have at least 1 synthetic", p.Address)
		for _, token := range p.Tokens {
			require.Equal(ts.T(), uint8(18), token.Decimals, "Metronome synthetics are documented/observed as always 18 decimals")
			require.True(ts.T(), token.Swappable)
		}
		if strings.EqualFold(p.Address, referencePool) {
			foundReference = true
		}
	}
	require.True(ts.T(), foundReference, "expected the known reference pool among getPools() results")

	var metadata PoolsListUpdaterMetadata
	require.NoError(ts.T(), json.Unmarshal(metadataBytes, &metadata))
	require.Equal(ts.T(), len(pools), metadata.Offset)
}

func (ts *PoolListUpdaterTestSuite) TestGetNewPools_OffsetSkipsKnownPools() {
	firstMeta, err := json.Marshal(PoolsListUpdaterMetadata{Offset: 1 << 30}) // past the end
	require.NoError(ts.T(), err)

	pools, metadataBytes, err := ts.updater.GetNewPools(context.Background(), firstMeta)
	require.NoError(ts.T(), err)
	require.Empty(ts.T(), pools)
	require.Equal(ts.T(), string(firstMeta), string(metadataBytes), "offset already past the end must be returned unchanged")
}

func TestPoolListUpdaterTestSuite(t *testing.T) {
	suite.Run(t, new(PoolListUpdaterTestSuite))
}
