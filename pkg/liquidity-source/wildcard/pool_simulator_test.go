package wildcard

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test/ethrpcmock"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestTrackPoolsPinsSamplesForBatch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		srv, recorder := ethrpcmock.New(t, pairABI, erc20ABI)
		client := ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress("0x4444444444444444444444444444444444444444"))
		extra, err := json.Marshal(Extra{IsNative: []bool{false, false}})
		require.NoError(t, err)
		pools := []entity.Pool{
			{Address: "0x1111111111111111111111111111111111111111", Tokens: []*entity.PoolToken{{Address: "0x2222222222222222222222222222222222222222", Decimals: 0}, {Address: "0x3333333333333333333333333333333333333333", Decimals: 0}}, Extra: string(extra)},
			{Address: "0x4444444444444444444444444444444444444445", Tokens: []*entity.PoolToken{{Address: "0x5555555555555555555555555555555555555555", Decimals: 0}, {Address: "0x6666666666666666666666666666666666666666", Decimals: 0}}, Extra: string(extra)},
		}
		got, err := TrackPools(context.Background(), pools, client, &Config{PriceTolerance: 10})
		require.NoError(t, err)
		require.Len(t, got, 2)
		for _, p := range got {
			require.Equal(t, uint64(ethrpcmock.Block), p.BlockNumber)
			require.NotEmpty(t, p.Reserves)
		}
		ethrpcmock.CheckPool(t, "wildcard_TestTrackPoolsPinsSamplesForBatch_0", got[0])
		ethrpcmock.CheckPool(t, "wildcard_TestTrackPoolsPinsSamplesForBatch_1", got[1])
		ethrpcmock.CheckRequests(t, "wildcard_TestTrackPoolsPinsSamplesForBatch", recorder)
		ethrpcmock.RequireRequestsPinned(t, recorder, 2)
	})
}

func TestGetNewPoolStatePinsNativeBalanceAndSamples(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const poolAddress = "0x1111111111111111111111111111111111111111"
		const multicallAddress = "0x4444444444444444444444444444444444444444"
		srv, recorder := ethrpcmock.New(t, pairABI, erc20ABI, multicallABI)
		client := ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress(multicallAddress))
		tracker := NewPoolTracker(&Config{MulticallAddress: multicallAddress, PriceTolerance: 10}, client)
		extra, err := json.Marshal(Extra{IsNative: []bool{true, false}})
		require.NoError(t, err)
		updated, err := tracker.GetNewPoolState(context.Background(), entity.Pool{
			Address: poolAddress,
			Tokens: []*entity.PoolToken{
				{Address: "0x2222222222222222222222222222222222222222", Decimals: 6},
				{Address: "0x3333333333333333333333333333333333333333", Decimals: 6},
			},
			Extra: string(extra),
		}, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		require.Equal(t, uint64(ethrpcmock.Block), updated.BlockNumber)
		require.Equal(t, entity.PoolReserves{"1", "1"}, updated.Reserves)
		requests := recorder.Snapshot()
		require.Len(t, requests, 2)
		require.Equal(t, "getEthBalance", requests[0].Calls[0].Method)
		require.Equal(t, "balanceOf", requests[0].Calls[1].Method)
		ethrpcmock.RequireRequestsPinned(t, recorder, 2)
		ethrpcmock.CheckPool(t, "wildcard-native-balance", updated)
		ethrpcmock.CheckRequests(t, "wildcard-native-balance", recorder)
	})
}

var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte(`{"address":"0xcc8e05f2d5e28b70214d3d384bada2bd0ae99cda","exchange":"wildcard","type":"wildcard","timestamp":1767595770,"reserves":["30699993136664026623","31490510717"],"tokens":[{"address":"0x4200000000000000000000000000000000000006","symbol":"WETH","decimals":18,"swappable":true},{"address":"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"rates\":[18332721688954665668,70366686069],\"isNative\":[true,false],\"samples\":[[[1000000000000000,3154701],[10000000000000000,31547016],[100000000000000000,315470163],[1000000000000000000,3154701633],[10000000000000000000,0],[100000000000000000000,0],[1000000000000000000000,0]],[[1000,0],[10000,0],[100000,31698338375796],[1000000,316983383757966],[10000000,3169833837579669],[100000000,31698338375796699],[1000000000,316983383757966994]]]}"}`),
		&entityPool)
	poolSim = lo.Must(NewPoolSimulator(entityPool))
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, poolSim, map[int]map[int]map[string]string{
		1: {
			0: {
				"3005810": "952791824733531",
			},
		},
		0: {
			1: {
				"1000000000000000": "3154701",
			},
		},
	})
}
