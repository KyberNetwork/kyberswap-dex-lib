package carbon

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"reflect"
	"testing"
	"testing/synctest"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test/ethrpcmock"
)

type PoolListTrackerTestSuite struct {
	suite.Suite

	updater *PoolsListUpdater
	tracker *PoolTracker
}

func TestGetNewPoolStatePinsPaginationAndIncrementalPaths(t *testing.T) {
	for _, tc := range []struct {
		name        string
		mode        string
		count       int64
		previous    int64
		previousIDs []int64
	}{
		{name: "full-scan-pagination", mode: "full", count: 201},
		{name: "incremental-update", mode: "update", count: 1, previous: 1, previousIDs: []int64{1}},
		{name: "incremental-add", mode: "add", count: 2, previous: 1, previousIDs: []int64{1}},
		{name: "incremental-delete-rescan", mode: "delete", count: 1, previous: 2, previousIDs: []int64{1, 2}},
		{name: "fee-error-default", mode: "fee-error", count: 1},
		{name: "fee-global-fallback", mode: "fee-global", count: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				const (
					controller  = "0x0000000000000000000000000000000000000001"
					poolAddress = "0x0000000000000000000000000000000000000004"
				)
				token0 := common.HexToAddress("0x0000000000000000000000000000000000000002")
				token1 := common.HexToAddress("0x0000000000000000000000000000000000000003")
				resolver := func(_ common.Address, method abi.Method, inputs []any) ([]any, error) {
					if tc.mode == "fee-error" && method.RawName == "pairTradingFeePPM" {
						return nil, ethrpcmock.ErrRPCError
					}
					values := make([]any, len(method.Outputs))
					for i, output := range method.Outputs {
						values[i] = ethrpcmock.Value(output.Type)
					}
					switch method.RawName {
					case "pairTradingFeePPM":
						if tc.mode == "fee-global" {
							values[0] = uint32(0)
						} else {
							values[0] = uint32(123)
						}
					case "tradingFeePPM":
						values[0] = uint32(456)
					case "strategiesByPairCount":
						values[0] = big.NewInt(tc.count)
					case "strategiesByPair":
						start := inputs[2].(*big.Int).Int64()
						end := inputs[3].(*big.Int).Int64()
						items := reflect.MakeSlice(method.Outputs[0].Type.GetType(), int(end-start), int(end-start))
						for i := range end - start {
							y := int64(100 * (start + i + 1))
							if tc.mode == "full" {
								y = 0
								if start+i == 200 {
									y = 1000
								}
							}
							if tc.mode == "add" {
								y = 0
							}
							setCarbonStrategyTuple(items.Index(int(i)), big.NewInt(start+i+1), token0, token1, y)
						}
						values[0] = items.Interface()
					case "strategy":
						id := inputs[0].(*big.Int)
						item := reflect.New(method.Outputs[0].Type.GetType()).Elem()
						y := id.Int64() * 100
						setCarbonStrategyTuple(item, id, token0, token1, y)
						values[0] = item.Interface()
					}
					return values, nil
				}
				srv, recorder := ethrpcmock.NewWithResolver(t, resolver, controllerABI)
				client := ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress("0x0000000000000000000000000000000000000005"))
				config := Config{DexId: DexType, Controller: common.HexToAddress(controller)}
				tracker := NewPoolTracker(&config, client)
				staticBytes, err := json.Marshal(StaticExtra{Token0: token0.Hex(), Token1: token1.Hex()})
				require.NoError(t, err)
				p := entity.Pool{Address: poolAddress, StaticExtra: string(staticBytes)}
				if tc.previous > 0 {
					prev := Extra{TradingFeePpm: 123, LastFullScanTime: time.Now().Unix(), StrategyCount: tc.previous}
					for _, id := range tc.previousIDs {
						prev.Strategies = append(prev.Strategies, Strategy{Id: big.NewInt(id), Orders: [2]Order{{Y: uint256.NewInt(uint64(id * 100)), Z: uint256.NewInt(100)}, {Y: uint256.NewInt(0), Z: uint256.NewInt(100)}}})
					}
					prevBytes, marshalErr := json.Marshal(prev)
					require.NoError(t, marshalErr)
					p.Extra = string(prevBytes)
				}
				updated, err := tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
				if tc.mode == "update" || tc.mode == "add" {
					require.ErrorContains(t, err, "cannot unmarshal struct")
					ethrpcmock.RequireRequestsPinned(t, recorder, 3)
					ethrpcmock.CheckPool(t, "carbon-tracker-"+tc.mode, updated)
					ethrpcmock.CheckRequests(t, "carbon-tracker-"+tc.mode, recorder)
					return
				}
				require.NoError(t, err)
				require.Equal(t, ethrpcmock.Block, updated.BlockNumber)
				var extra Extra
				require.NoError(t, json.Unmarshal([]byte(updated.Extra), &extra))
				if tc.mode == "fee-error" {
					require.Equal(t, uint32(defaultTradingFeePpm), extra.TradingFeePpm)
				} else if tc.mode == "fee-global" {
					require.Equal(t, uint32(456), extra.TradingFeePpm)
				} else {
					require.Equal(t, uint32(123), extra.TradingFeePpm)
				}
				require.Equal(t, tc.count, extra.StrategyCount)
				switch tc.mode {
				case "full":
					require.Len(t, recorder.Snapshot(), 4)
					require.Len(t, extra.Strategies, 1)
					require.Equal(t, entity.PoolReserves{"1000", "0"}, updated.Reserves)
				case "update":
					require.Len(t, recorder.Snapshot(), 3)
					require.Equal(t, "100", extra.Strategies[0].Orders[0].Y.String())
				case "add":
					require.Len(t, recorder.Snapshot(), 4)
					require.Len(t, extra.Strategies, 2)
				case "delete":
					require.Len(t, recorder.Snapshot(), 4)
					require.Len(t, extra.Strategies, 1)
				case "fee-error":
					require.Len(t, recorder.Snapshot(), 3)
					require.Equal(t, "latest", recorder.Snapshot()[0].BlockTag)
					require.Equal(t, "latest", recorder.Snapshot()[1].BlockTag)
					require.Equal(t, "0x3039", recorder.Snapshot()[2].BlockTag)
				case "fee-global":
					require.Len(t, recorder.Snapshot(), 2)
					if os.Getenv("BLOCKPIN_BASELINE") == "1" {
						require.Equal(t, "latest", recorder.Snapshot()[1].BlockTag)
					} else {
						require.Equal(t, "0x3039", recorder.Snapshot()[1].BlockTag)
					}
				}
				if tc.mode != "fee-error" {
					ethrpcmock.RequireRequestsPinned(t, recorder, 2)
				}
				ethrpcmock.CheckPool(t, "carbon-tracker-"+tc.mode, updated)
				ethrpcmock.CheckRequests(t, "carbon-tracker-"+tc.mode, recorder)
			})
		})
	}
}

func setCarbonStrategyTuple(value reflect.Value, id *big.Int, token0, token1 common.Address, y0 int64) {
	value.Field(0).Set(reflect.ValueOf(id))
	value.Field(2).Index(0).Set(reflect.ValueOf(token0))
	value.Field(2).Index(1).Set(reflect.ValueOf(token1))
	for i := 0; i < 2; i++ {
		order := value.Field(3).Index(i)
		y := int64(0)
		if i == 0 {
			y = y0
		}
		order.Field(0).Set(reflect.ValueOf(big.NewInt(y)))
		order.Field(1).Set(reflect.ValueOf(big.NewInt(100)))
		order.Field(2).SetUint(1)
		order.Field(3).SetUint(1)
	}
}

func (ts *PoolListTrackerTestSuite) SetupTest() {
	rpcUrl := lo.CoalesceOrEmpty(os.Getenv("RPC_1"), "https://ethereum-rpc.kyberswap.com")
	rpcClient := ethrpc.New(rpcUrl).
		SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11"))

	config := Config{
		DexId:      DexType,
		Controller: common.HexToAddress("0xc537e898cd774e2dcba3b14ea6f34c93d5ea45e1"),
	}

	ts.updater = NewPoolsListUpdater(&config, rpcClient)
	ts.tracker = NewPoolTracker(&config, rpcClient)
}

func (ts *PoolListTrackerTestSuite) TestGetNewPoolState() {
	pools, _, err := ts.updater.GetNewPools(context.Background(), nil)
	require.NoError(ts.T(), err)
	require.Greater(ts.T(), len(pools), 0)

	for _, p := range pools {
		newPoolState, err := ts.tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
		require.NoError(ts.T(), err)
		require.NotNil(ts.T(), newPoolState)
	}
}

func TestPoolListTrackerTestSuite(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	suite.Run(t, new(PoolListTrackerTestSuite))
}
