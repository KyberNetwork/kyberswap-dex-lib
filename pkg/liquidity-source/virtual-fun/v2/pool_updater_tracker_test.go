package v2

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"testing/synctest"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test/ethrpcmock"
	abipkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListTrackerTestSuite struct {
	suite.Suite

	updater *PoolsListUpdater
	tracker *PoolTracker
}

func TestGetNewPoolStatePinsTaxReads(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var mockBondingABI abi.ABI
		var err error
		mockBondingJSON := bytes.ReplaceAll(bondingABIJson, []byte(`"name": "_name"`), []byte(`"name": "internalName"`))
		mockBondingABI, err = abi.JSON(bytes.NewReader(mockBondingJSON))
		require.NoError(t, err)
		srv, recorder := ethrpcmock.New(t, factoryABI, mockBondingABI, pairABI, abipkg.Erc20ABI)
		rpcClient := ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress("0x4444444444444444444444444444444444444444"))
		cfg := &Config{Factory: "0x5555555555555555555555555555555555555555"}
		tracker := NewPoolTracker(cfg, rpcClient)
		staticExtra, err := json.Marshal(StaticExtra{Bonding: "0x6666666666666666666666666666666666666666"})
		require.NoError(t, err)
		p := entity.Pool{Address: "0x7777777777777777777777777777777777777777", Tokens: []*entity.PoolToken{{Address: "0x8888888888888888888888888888888888888888"}, {Address: "0x9999999999999999999999999999999999999999"}}, StaticExtra: string(staticExtra)}
		got, err := tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		require.Equal(t, uint64(ethrpcmock.Block), got.BlockNumber)
		require.NotEmpty(t, got.Reserves)
		ethrpcmock.CheckPool(t, "virtual-fun-v2_TestGetNewPoolStatePinsTaxReads", got)
		ethrpcmock.CheckRequests(t, "virtual-fun-v2_TestGetNewPoolStatePinsTaxReads", recorder)
		ethrpcmock.RequireRequestsPinned(t, recorder, 2)
	})
}

func (ts *PoolListTrackerTestSuite) SetupTest() {
	rpcClient := ethrpc.New("https://base-rpc.kyberswap.com").
		SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11"))

	config := Config{
		ChainID:      valueobject.ChainIDBase,
		DexId:        DexType,
		Factory:      "0xD7D3C85B4f2e9bee1998cD2E98820e647792d284",
		NewPoolLimit: 100,
	}

	ts.updater = NewPoolsListUpdater(&config, rpcClient)
	ts.tracker = NewPoolTracker(&config, rpcClient)
}

func (ts *PoolListTrackerTestSuite) TestGetNewPoolState() {
	var metadataBytes []byte
	for {
		pools, newMetadataBytes, err := ts.updater.GetNewPools(ts.T().Context(), metadataBytes)
		require.NoError(ts.T(), err)
		require.Greater(ts.T(), len(pools), 0)

		metadataBytes = newMetadataBytes
	}
}

func TestPoolListTrackerTestSuite(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	suite.Run(t, new(PoolListTrackerTestSuite))
}
