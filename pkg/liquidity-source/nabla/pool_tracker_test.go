package nabla

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListTrackerTestSuite struct {
	suite.Suite

	lister  *PoolsListUpdater
	tracker *PoolTracker
}

func (ts *PoolListTrackerTestSuite) SetupTest() {
	rpcClient := ethrpc.New("https://berachain.drpc.org").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	config := Config{
		DexId:           DexType,
		ChainId:         valueobject.ChainIDBerachain,
		Portal:          "0x1F917Fe724F186a1fFA7744A73afed18C335b9eC",
		Oracle:          "0x6d6190Da8fD73E0C911929DED2D6B47cE066e441",
		PythAdapterV2:   "0x9B5a425a9F4b4411D42B21caacf86d026dce43Ec",
		SkipPriceUpdate: false,
		PriceAPI:        "https://antenna.nabla.fi/v1/updates/price/latest",
		PriceTimeout:    durationjson.Duration{Duration: 10 * time.Second},
	}

	ts.lister = NewPoolsListUpdater(&config, rpcClient)
	ts.tracker = NewPoolTracker(&config, rpcClient)
}

func (ts *PoolListTrackerTestSuite) TestGetNewPoolState() {
	pools, _, err := ts.lister.GetNewPools(context.Background(), nil)
	require.NoError(ts.T(), err)
	require.Greater(ts.T(), len(pools), 0)

	for _, p := range pools {
		newPoolState, err := ts.tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{
			Logs: []types.Log{
				{
					// PriceFeedUpdate
					Address: common.HexToAddress("0x6d6190Da8fD73E0C911929DED2D6B47cE066e441"),
					Topics: []common.Hash{
						common.HexToHash("0xd06a6b7f4918494b3719217d1802786c1f5112a6c1d88fe2cfec00b4584f6aec"),
						common.HexToHash("0xe62df6c8b4a85fe1a67db44dc12de5db330f7ac66b72dc658afedf0f4a415b43"),
					},
					Data:        common.FromHex("0x00000000000000000000000000000000000000000000000000000000694c90bd000000000000000000000000000000000000000000000000000007f684fca8a00000000000000000000000000000000000000000000000000000000000000000"),
					BlockNumber: math.MaxUint64,
				},
				{
					// ReserveUpdated
					Address: common.HexToAddress("0xe971445787dcb0bb577610126287ded493dddae7"),
					Topics: []common.Hash{
						common.HexToHash("0x736a4a5812ced57865d349f18ffc358079c6b479326c0dfd1dae30c465b1daf2"),
					},
					Data:        common.FromHex("0x00000000000000000000000000000000000000000000000000000004ff89c5a900000000000000000000000000000000000000000000000000000004ffb8521a00000000000000000000000000000000000000000000000000000009fe47c170"),
					BlockNumber: math.MaxUint64,
				},
				{
					// SwapFeesSet
					Address: common.HexToAddress("0xe971445787dcb0bb577610126287ded493dddae7"),
					Topics: []common.Hash{
						common.HexToHash("0xd51891e6ac27da6065760e4843c63beb01795531a5c017b29f959a4c1055c498"),
						common.HexToHash("0x0000000000000000000000008f7447a6a04857855caf75ecb1600d2984a7285d"),
					},
					Data:        common.FromHex("0x000000000000000000000000000000000000000000000000000000000000008700000000000000000000000000000000000000000000000000000000000000b40000000000000000000000000000000000000000000000000000000000000087"),
					BlockNumber: math.MaxUint64,
				},
			},
		})
		require.NoError(ts.T(), err)

		poolBytes, err := json.Marshal(newPoolState)
		require.NoError(ts.T(), err)
		ts.T().Log(string(poolBytes))
	}
}

func TestPoolListTrackerTestSuite(t *testing.T) {
	t.Parallel()

	test.SkipCI(t)

	suite.Run(t, new(PoolListTrackerTestSuite))
}
