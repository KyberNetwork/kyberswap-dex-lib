package ekubov3

import (
	"context"
	"encoding/binary"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/pools"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type (
	PoolTrackerTestSuite struct {
		suite.Suite

		trackers map[valueobject.ChainID]*PoolTracker
	}

	testcase struct {
		name               string
		chainID            valueobject.ChainID
		txHash             string
		blockTimestamp     uint64
		poolKey            pools.AnyPoolKey
		extensionType      ExtensionType
		stateBefore        any
		expectedStateAfter any
	}
)

func (ts *PoolTrackerTestSuite) run(cases []*testcase) {
	t := ts.T()

	for _, tc := range cases {
		ts.Run(tc.name, func() {
			extraJson, err := json.Marshal(tc.stateBefore)
			require.NoError(t, err)

			staticExtra := StaticExtra{
				Core:          MainnetConfig.Core,
				ExtensionType: tc.extensionType,
				PoolKey:       tc.poolKey,
			}

			staticExtraJson, err := json.Marshal(&staticExtra)
			require.NoError(t, err)

			p := entity.Pool{
				Tokens: []*entity.PoolToken{
					{Address: valueobject.ZeroToWrappedLower(tc.poolKey.Token0.String(), MainnetConfig.ChainId)},
					{Address: valueobject.ZeroToWrappedLower(tc.poolKey.Token1.String(), MainnetConfig.ChainId)},
				},
				Extra:       string(extraJson),
				StaticExtra: string(staticExtraJson),
			}

			tracker := ts.trackers[tc.chainID]

			receipt, err := tracker.ethrpcClient.
				GetETHClient().
				TransactionReceipt(context.Background(), common.HexToHash(tc.txHash))
			require.NoError(t, err)

			blockNumber := receipt.BlockNumber.Uint64()

			logs := lo.Map(receipt.Logs, func(log *types.Log, _ int) types.Log {
				return *log
			})

			newPoolState, err := tracker.GetNewPoolState(
				context.Background(),
				p,
				pool.GetNewPoolStateParams{
					Logs: logs,
					BlockHeaders: map[uint64]entity.BlockHeader{
						blockNumber: {Timestamp: tc.blockTimestamp},
					},
				},
			)
			require.NoError(t, err)

			poolAfter, err := unmarshalPool([]byte(newPoolState.Extra), &staticExtra)
			require.NoError(ts.T(), err, "Failed to unmarshal pool")

			require.Equal(t, tc.expectedStateAfter, poolAfter.GetState())
		})
	}
}

func (ts *PoolTrackerTestSuite) TestPositionUpdated() {
	ts.Run("PositionUpdated", func() {
		ts.run([]*testcase{
			{
				name:    "Add concentrated pool liquidity",
				txHash:  "0x2757427086944621c7fb8eca63a01809be4c76bb5b7b32596ced53d7fd17a691",
				chainID: valueobject.ChainIDEthereum,
				poolKey: anyPoolKey(
					valueobject.ZeroAddress,
					"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					common.Address{}.Hex(),
					9223372036854775,
					pools.NewConcentratedPoolTypeConfig(1000),
				),
				extensionType: ExtensionTypeNoSwapCallPoints,
				// State after pool initialization https://etherscan.io/tx/0x2757427086944621c7fb8eca63a01809be4c76bb5b7b32596ced53d7fd17a691#eventlog#114
				stateBefore: pools.NewConcentratedPoolState(
					pools.NewConcentratedPoolSwapState(
						math.FloatSqrtRatioToFixed(uint256.MustFromHex("0x4000e4ac4ee732e5c7c0529d")),
						new(uint256.Int),
						0,
					),
					[]pools.Tick{
						{Number: math.MinTick, LiquidityDelta: new(int256.Int)},
						{Number: math.MaxTick, LiquidityDelta: new(int256.Int)},
					},
					[2]int32{math.MinTick, math.MaxTick},
					int32(binary.BigEndian.Uint32([]byte("\xfe\xd4\x69\x15"))),
				),
				// Position update https://etherscan.io/tx/0x2757427086944621c7fb8eca63a01809be4c76bb5b7b32596ced53d7fd17a691#eventlog#116
				expectedStateAfter: pools.NewConcentratedPoolState(
					pools.NewConcentratedPoolSwapState(
						math.FloatSqrtRatioToFixed(uint256.MustFromHex("0x4000e4ac4ee732e5c7c0529d")),
						uint256.NewInt(22875426408333),
						1,
					),
					[]pools.Tick{
						{Number: math.MinTick, LiquidityDelta: new(int256.Int)},
						{Number: int32(binary.BigEndian.Uint32([]byte("\xFE\xD4\x2A\x30"))), LiquidityDelta: int256.NewInt(22875426408333)},
						{Number: int32(binary.BigEndian.Uint32([]byte("\xFE\xD4\xA7\x30"))), LiquidityDelta: int256.NewInt(-22875426408333)},
						{Number: math.MaxTick, LiquidityDelta: new(int256.Int)},
					},
					[2]int32{math.MinTick, math.MaxTick},
					int32(binary.BigEndian.Uint32([]byte("\xfe\xd4\x69\x15"))),
				),
			},
		})
	})
}

func (ts *PoolTrackerTestSuite) TestSwapped() {
	ts.run([]*testcase{
		{
			name:    "Swap",
			chainID: valueobject.ChainIDEthereum,
			txHash:  "0xee56e1f3bad803bd857fb118e55d7eabb5368a94ae8f11e83724278f474294ca",
			poolKey: anyPoolKey(
				common.Address{}.Hex(),
				"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				common.Address{}.Hex(),
				9223372036854775,
				pools.NewConcentratedPoolTypeConfig(1000),
			),
			extensionType: ExtensionTypeNoSwapCallPoints,
			stateBefore: pools.NewConcentratedPoolState(
				pools.NewConcentratedPoolSwapState(
					big256.New("18552164211672086963009471320686592"),
					uint256.NewInt(17156571521907),
					1,
				),
				[]pools.Tick{
					{Number: math.MinTick, LiquidityDelta: new(int256.Int)},
					{Number: -19650000, LiquidityDelta: int256.NewInt(17156571521907)},
					{Number: -19618000, LiquidityDelta: int256.NewInt(-17156571521907)},
					{Number: math.MaxTick, LiquidityDelta: new(int256.Int)},
				},
				[2]int32{math.MinTick, math.MaxTick},
				-19633899,
			),
			expectedStateAfter: pools.NewConcentratedPoolState(
				pools.NewConcentratedPoolSwapState(
					big256.New("18551574977108402281445297823416320"),
					uint256.NewInt(17156571521907),
					1,
				),
				[]pools.Tick{
					{Number: math.MinTick, LiquidityDelta: new(int256.Int)},
					{Number: -19650000, LiquidityDelta: int256.NewInt(17156571521907)},
					{Number: -19618000, LiquidityDelta: int256.NewInt(-17156571521907)},
					{Number: math.MaxTick, LiquidityDelta: new(int256.Int)},
				},
				[2]int32{math.MinTick, math.MaxTick},
				-19633963,
			),
		},
	})
}

func (ts *PoolTrackerTestSuite) TestVirtualOrdersExecutedAndOrderUpdated() {
	ts.run([]*testcase{
		{
			name:           "Execute virtual orders & stop order",
			txHash:         "0xde6812e959a49e245f15714d1b50571f43ca7711c91d2df1087178a38bc554b7",
			chainID:        valueobject.ChainIDEthereum,
			blockTimestamp: 1767625571,
			poolKey: anyPoolKey(
				common.Address{}.Hex(),
				"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"0xd4F1060cB9c1A13e1d2d20379b8aa2cF7541eD9b",
				55340232221128654,
				pools.NewFullRangePoolTypeConfig(),
			),
			extensionType: ExtensionTypeTwamm,
			stateBefore: pools.NewTwammPoolState(
				pools.NewFullRangePoolState(
					pools.NewFullRangePoolSwapState(big256.New("19112726775014745474545736843526144")),
					uint256.NewInt(2670594),
				),
				pools.NewTimedPoolState(
					pools.NewTimedPoolSwapState(new(uint256.Int), uint256.NewInt(3744848), 1767625523),
					[]pools.TimeRateDelta{
						pools.NewTimeRateDelta(1767636992, new(int256.Int), int256.NewInt(-3744848)),
					},
				),
			),
			expectedStateAfter: pools.NewTwammPoolState(
				pools.NewFullRangePoolState(
					pools.NewFullRangePoolSwapState(big256.New("19112726775014745474545736843526144")),
					uint256.NewInt(2670594),
				),
				pools.NewTimedPoolState(
					pools.NewTimedPoolSwapState(new(uint256.Int), new(uint256.Int), 1767625571),
					[]pools.TimeRateDelta{},
				),
			),
		},
	})
}

func (ts *PoolTrackerTestSuite) TestFeesDonatedAndPoolBoosted() {
	ts.run([]*testcase{
		{
			name:           "Boost pool",
			txHash:         "0x50d4a090a2c8eb375efa4a5980a7f6274d2bc89f316f474aea6d27c202eb87e3",
			chainID:        chainIDSepolia,
			blockTimestamp: 1769448828,
			poolKey: anyPoolKey(
				"0x31f017286c6932a1713a31024ab14b9dc7f3ba6c",
				"0xf45a9c4f3be630703aa89c213bcda334e5a2d803",
				"0xd4b54d0ca6979da05f25895e6e269e678ba00f9e",
				0,
				pools.NewConcentratedPoolTypeConfig(50),
			),
			extensionType: ExtensionTypeBoostedFeesConcentrated,
			stateBefore: pools.NewBoostedFeesPoolState(
				pools.NewConcentratedPoolState(
					pools.NewConcentratedPoolSwapState(
						big256.U2Pow128,
						new(uint256.Int),
						-1,
					),
					[]pools.Tick{},
					[2]int32{math.MinTick, math.MaxTick},
					0,
				),
				pools.NewTimedPoolState(
					pools.NewTimedPoolSwapState(new(uint256.Int), new(uint256.Int), 0),
					[]pools.TimeRateDelta{},
				),
			),
			expectedStateAfter: pools.NewBoostedFeesPoolState(
				pools.NewConcentratedPoolState(
					pools.NewConcentratedPoolSwapState(
						big256.U2Pow128,
						new(uint256.Int),
						-1,
					),
					[]pools.Tick{},
					[2]int32{math.MinTick, math.MaxTick},
					0,
				),
				pools.NewTimedPoolState(
					pools.NewTimedPoolSwapState(
						big256.New("150000000000000000000000"),
						new(uint256.Int),
						1769448828,
					),
					[]pools.TimeRateDelta{
						pools.NewTimeRateDelta(
							1795162112,
							big256.SNew("-150000000000000000000000"),
							new(int256.Int),
						),
					},
				),
			),
		},
	})
}

func (ts *PoolTrackerTestSuite) SetupSuite() {
	ts.trackers = map[valueobject.ChainID]*PoolTracker{
		valueobject.ChainIDEthereum: NewPoolTracker(
			MainnetConfig,
			ethrpc.New("https://ethereum.drpc.org").
				SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")),
		),
		chainIDSepolia: NewPoolTracker(SepoliaConfig, ethrpc.New("https://sepolia.drpc.org").
			SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))),
	}
}

func TestPoolTrackerTestSuite(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	suite.Run(t, new(PoolTrackerTestSuite))
}

// Replays an Ethereum incident on full range pool 0xcd214dce…1a17: block 26036223 initializes the pool and adds
// liquidity L in one tx, block 26036511 removes all of it. pool-service creates the pool from an RPC fetch at block
// 26036223 (the state already includes +L), then hands the tracker that same block's PositionUpdated(+L). Applying
// it again left liquidity L after the full withdrawal, so the drained pool kept quoting reserves it no longer had.
func TestPoolTracker_SkipsLogsAlreadyIncludedInState(t *testing.T) {
	t.Parallel()

	const (
		createdBlock   = 26036223
		withdrawnBlock = 26036511
	)

	poolKey := anyPoolKey(
		valueobject.ZeroAddress,
		"0x04c46e830bb56ce22735d5d8fc9cb90309317d0f",
		"0x948b9c2c99718034954110cb61a6e08e107745f9",
		9223372036854775,
		pools.NewFullRangePoolTypeConfig(),
	)
	poolAddress, err := poolKey.ToPoolAddress()
	require.NoError(t, err)
	require.Equal(t, "0xcd214dceb5fd68a6c4cc931946f3e1a2bd85e42bbff6839013b30af733561a17", poolAddress)

	positionUpdated := func(blockNumber uint64, data string) types.Log {
		return types.Log{
			Address:     MainnetConfig.Core,
			Topics:      []common.Hash{abis.PositionUpdatedEvent.ID},
			Data:        common.FromHex(data),
			BlockNumber: blockNumber,
		}
	}
	// https://etherscan.io/tx/0x26a00e60d08436c2f55d7798982a057bd6098752144c894e539164f12ebcb55b, liquidityDelta +L
	addLiquidity := positionUpdated(createdBlock, "0x00000000000000000000000002d9876a21af7545f8632c3af76ec90b5ad4b66dcd214dceb5fd68a6c4cc931946f3e1a2bd85e42bbff6839013b30af733561a1708925ac08f40a8fba639449e7ea7519e926687cd08482aa2fab6326d0549cd930000000000000000000000000000000000000000000000007721285763e3760b000000000000000001e9aebbcaa9bbab000000000000001cfb520bb767ef77228000000f91e26cec29913069007e166300000000000000007721285763e3760b")
	// https://etherscan.io/tx/0x521dcfd7ba10d290cc95655edc21078dce0eced8955d48c6c50e663325f64c72, liquidityDelta -L
	removeLiquidity := positionUpdated(withdrawnBlock, "0x00000000000000000000000002d9876a21af7545f8632c3af76ec90b5ad4b66dcd214dceb5fd68a6c4cc931946f3e1a2bd85e42bbff6839013b30af733561a1708925ac08f40a8fba639449e7ea7519e926687cd08482aa2fab6326d0549cd93ffffffffffffffffffffffffffffffffffffffffffffffff88ded7a89c1c89f5fffffffffffffffffe16514435564456ffffffffffffffe304adf448981088df8000000f91e26cec29913069007e166300000000000000000000000000000000")

	sqrtRatio := math.FloatSqrtRatioToFixed(uint256.MustFromHex("0x8000000f91e26cec29913069"))
	liquidityL := uint256.MustFromHex("0x7721285763e3760b")

	staticExtraJson, err := json.Marshal(&StaticExtra{
		Core:          MainnetConfig.Core,
		ExtensionType: ExtensionTypeNoSwapCallPoints,
		PoolKey:       poolKey,
	})
	require.NoError(t, err)

	newEntityPool := func(t *testing.T, liquidity *uint256.Int, blockNumber uint64) entity.Pool {
		extraJson, err := json.Marshal(pools.NewFullRangePoolState(
			pools.NewFullRangePoolSwapState(sqrtRatio.Clone()), liquidity.Clone()))
		require.NoError(t, err)

		return entity.Pool{
			Address:     poolAddress,
			Extra:       string(extraJson),
			StaticExtra: string(staticExtraJson),
			BlockNumber: blockNumber,
		}
	}

	tracker := NewPoolTracker(MainnetConfig, nil)
	update := func(t *testing.T, p entity.Pool, logs ...types.Log) entity.Pool {
		p, err := tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{Logs: logs})
		require.NoError(t, err)
		return p
	}

	liquidityOf := func(t *testing.T, p entity.Pool) string {
		var state pools.FullRangePoolState
		require.NoError(t, json.Unmarshal([]byte(p.Extra), &state))
		return state.Liquidity.Dec()
	}

	t.Run("creation block log already in the RPC-fetched state is not applied again", func(t *testing.T) {
		p := update(t, newEntityPool(t, liquidityL, createdBlock), addLiquidity)

		require.Equal(t, liquidityL.Dec(), liquidityOf(t, p))
		require.EqualValues(t, createdBlock, p.BlockNumber)
	})

	t.Run("full withdrawal after creation leaves no liquidity and no reserves", func(t *testing.T) {
		p := update(t, newEntityPool(t, liquidityL, createdBlock), addLiquidity)
		p = update(t, p, removeLiquidity)

		require.Equal(t, "0", liquidityOf(t, p))
		require.Equal(t, []string{"0", "0"}, []string(p.Reserves))
		require.EqualValues(t, withdrawnBlock, p.BlockNumber)
	})

	t.Run("logs from blocks after the state are still applied", func(t *testing.T) {
		p := update(t, newEntityPool(t, new(uint256.Int), createdBlock-1), addLiquidity)

		require.Equal(t, liquidityL.Dec(), liquidityOf(t, p))
		require.EqualValues(t, createdBlock, p.BlockNumber)
	})

	t.Run("late logs older than the state do not move its block number back", func(t *testing.T) {
		// A block number moved back would let a later task re-apply logs the state already includes.
		p := update(t, newEntityPool(t, new(uint256.Int), withdrawnBlock), addLiquidity)

		require.Equal(t, "0", liquidityOf(t, p))
		require.EqualValues(t, withdrawnBlock, p.BlockNumber)
	})
}

// Ethereum USDe/sUSDe pool 0x0fcb46a4…4990 (tick spacing 10, no extension), as tracked at
// block 26283908: tick -227046 inside the fetched tick window [-227050, -216810].
const (
	tickWindowPoolAddress     = "0x0fcb46a418903bca6840ea3801dc09b2f46b5f423450e5772fb1c9a0af3d4990"
	tickWindowPoolBlockNumber = 26283908
	tickWindowPoolExtra       = `{"sqrtRatio":"303764564999549016502112032689601118208","liquidity":"8153170015316476136074403","activeTickIndex":0,"sortedTicks":[{"number":-227050,"liquidityDelta":"8153170015316476136074403"},{"number":-227030,"liquidityDelta":"-2662"},{"number":-226880,"liquidityDelta":"-756961236783954980362751"},{"number":-226730,"liquidityDelta":"-1569"},{"number":-226580,"liquidityDelta":"-6812651131055594823264187"},{"number":-224430,"liquidityDelta":"-583557647476926332443234"},{"number":-216810,"liquidityDelta":"0"}],"tickBounds":[-227050,-216810],"activeTick":-227046}`
	tickWindowPoolStaticExtra = `{"core":"0x00000000000014aa86c5d3c41765bb24e11bd701","extensionType":1,"poolKey":{"token0":"0x4c9edd5852cd905f086c759e8383e09bff1e68b3","token1":"0x9d39a5de30e57443bff2a8307a4256c8797a3497","config":{"fee":922337203685478,"typeConfig":{"tickSpacing":10},"extension":"0x0000000000000000000000000000000000000000"}}}`

	// Swapped logs of that pool on Core. Block 26285633 (tx 0xe8774c9183b1ca377bf58bd31cbfd5034eaff96f14c418b3734360bd8985cee4) moves the tick to -227148, below the window;
	// block 26288407 (tx 0xd8980983d760a8ba335d6bbeb6e81da6159927448bc8ad5f57bdf803e4432cf5) moves it to -227115.
	tickWindowSwapLeavingWindow         = "0xd26f20001a72a18c002b00e6710000d68700ce000fcb46a418903bca6840ea3801dc09b2f46b5f423450e5772fb1c9a0af3d499000000000000000194024622e052ef4f1ffffffffffffffebe1589be6233c9d377920fc68a551b81d9397bb70fffc88b4000000000006be803b6f003d3d552ca3"
	tickWindowSwapInsideRefetchedWindow = "0xc3ba641e5ee7441c4d60b76d04e6e3876973dd140fcb46a418903bca6840ea3801dc09b2f46b5f423450e5772fb1c9a0af3d4990fffffffffffffff7e2f433911b258b540000000000000006771b52347ced96c8792139c2599d9bd0ab900578fffc88d5000000000006be803b6f003d3d552ca3"

	// Multicall aggregate result of QuoteDataFetcher.getQuoteData([pool], 2) at block 26285633, recorded through
	// an anvil fork of Ethereum at that block. It returns tick -227148 and the window [-232268, -222028].
	tickWindowQuoteDataAt26285633 = "0x0000000000000000000000000000000000000000000000000000000001911641000000000000000000000000000000000000000000000000000000" +
		"000000004000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000" +
		"000000000000000020000000000000000000000000000000000000000000000000000000000000030000000000000000000000000000000000000000" +
		"000000000000000000000000200000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000" +
		"0000000000000000000000000000000020fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc88b40000000000000000000000" +
		"0000000000000000007920fc68a551b81d9397bb7000000000000000000000000000000000000000000006be803b6f003d3d552ca3ffffffffffffff" +
		"fffffffffffffffffffffffffffffffffffffffffffffc74b4fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc9cb4000000" +
		"00000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000000" +
		"07fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc7f52000000000000000000000000000000000000000000007b92b9348f" +
		"56b5429a62fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc88940000000000000000000000000000000000000000000642" +
		"ed823a70e688129241fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc892affffffffffffffffffffffffffffffffffffff" +
		"fffffffffffffffffffffff59afffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc89c0ffffffffffffffffffffffffffffff" +
		"ffffffffffffff5fb50c93c1828bfe2601fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc8a56ffffffffffffffffffffff" +
		"fffffffffffffffffffffffffffffffffffffff9dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc8aecffffffffffffff" +
		"fffffffffffffffffffffffffffffa5d5d7131cd96ebef5845fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc9352ffffff" +
		"ffffffffffffffffffffffffffffffffffffff846d46cb70a94abd659e"
)

// quoteDataReplayServer answers every eth_call with the recorded QuoteDataFetcher result and counts the calls.
func quoteDataReplayServer(t *testing.T, result string) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		body, err := io.ReadAll(r.Body)
		if err == nil {
			err = json.Unmarshal(body, &req)
		}
		if err != nil || req.Method != "eth_call" {
			t.Errorf("unexpected RPC request %s: %v", body, err)
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		calls.Add(1)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":"` + result + `"}`))
	}))
	t.Cleanup(srv.Close)

	return srv, &calls
}

// The tracker kept the tick window fetched at pool creation or at the last RPC fallback. Swapped logs moved the
// active tick out of it, and the stale window then set the price limit of swaps toward the price on the wrong side
// of the current price, so they reverted on chain with SqrtRatioLimitWrongDirection. The tracker must refetch the
// pool once a log leaves the active tick outside the window.
//
// Not parallel: GetNewPoolState logs through klog, whose global logger is set lazily on first use without
// synchronization (kutils v0.4.2 klog.Log). TestPoolTracker_SkipsLogsAlreadyIncludedInState is parallel, and a
// second parallel test reaching that first use races with it under -race.
func TestPoolTracker_RefetchesWhenActiveTickLeavesTickWindow(t *testing.T) {
	srv, rpcCalls := quoteDataReplayServer(t, tickWindowQuoteDataAt26285633)
	tracker := NewPoolTracker(MainnetConfig, ethrpc.New(srv.URL).
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")))

	swapped := func(blockNumber uint64, data string) types.Log {
		return types.Log{Address: MainnetConfig.Core, Data: common.FromHex(data), BlockNumber: blockNumber}
	}
	update := func(t *testing.T, p entity.Pool, logs ...types.Log) entity.Pool {
		p, err := tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{Logs: logs})
		require.NoError(t, err)
		return p
	}
	stateOf := func(t *testing.T, p entity.Pool) *pools.ConcentratedPoolState {
		var state pools.ConcentratedPoolState
		require.NoError(t, json.Unmarshal([]byte(p.Extra), &state))
		return &state
	}

	p := entity.Pool{
		Address:     tickWindowPoolAddress,
		Extra:       tickWindowPoolExtra,
		StaticExtra: tickWindowPoolStaticExtra,
		BlockNumber: tickWindowPoolBlockNumber,
	}
	require.EqualValues(t, [2]int32{-227050, -216810}, stateOf(t, p).TickBounds)

	p = update(t, p, swapped(26285633, tickWindowSwapLeavingWindow))

	state := stateOf(t, p)
	require.EqualValues(t, 1, rpcCalls.Load(), "a swap that leaves the tick window must refetch the pool")
	require.EqualValues(t, -227148, state.ActiveTick)
	require.EqualValues(t, [2]int32{-232268, -222028}, state.TickBounds)
	require.EqualValues(t, 26285633, p.BlockNumber)

	// The symptom: a USDe -> sUSDe quote moves the price down, so its price limit must lie below the current price.
	var staticExtra StaticExtra
	require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &staticExtra))
	ekuboPool, err := unmarshalPool([]byte(p.Extra), &staticExtra)
	require.NoError(t, err)
	quote, err := ekuboPool.Quote(uint256.MustFromDecimal("1000000000000000000000"), false)
	require.NoError(t, err)
	require.True(t, math.FloatSqrtRatioToFixed(quote.SwapInfo.PriceLimit).Lt(state.SqrtRatio),
		"price limit of a price-decreasing swap is above the current price")

	t.Run("a swap that stays inside the refetched window does not refetch", func(t *testing.T) {
		p := update(t, p, swapped(26288407, tickWindowSwapInsideRefetchedWindow))

		state := stateOf(t, p)
		require.EqualValues(t, 1, rpcCalls.Load())
		require.EqualValues(t, -227115, state.ActiveTick)
		require.EqualValues(t, [2]int32{-232268, -222028}, state.TickBounds)
		require.EqualValues(t, 26288407, p.BlockNumber)
	})
}

// Every pool type that quotes from a fetched tick window is checked, including a ve33 pool wrapping one; full range
// pools have no window.
func TestActiveTickOutsideTickBoundsPoolTypes(t *testing.T) {
	t.Parallel()

	token0, token1 := common.HexToAddress("0x1"), common.HexToAddress("0x2")
	concentratedKey := pools.NewPoolKey(token0, token1,
		pools.NewPoolConfig(common.Address{}, 0, pools.NewConcentratedPoolTypeConfig(10)))
	fullRangeKey := pools.NewPoolKey(token0, token1,
		pools.NewPoolConfig(common.Address{}, 0, pools.NewFullRangePoolTypeConfig()))

	concentratedState := func(activeTick int32) *pools.ConcentratedPoolState {
		return pools.NewConcentratedPoolState(nil, nil, [2]int32{-227050, -216810}, activeTick)
	}
	timedState := pools.NewTimedPoolState(pools.NewTimedPoolSwapState(new(uint256.Int), new(uint256.Int), 0), nil)
	fullRange := pools.NewFullRangePool(fullRangeKey,
		pools.NewFullRangePoolState(pools.NewFullRangePoolSwapState(new(uint256.Int)), new(uint256.Int)))

	for _, activeTick := range []int32{-227148, -227046} {
		outside := activeTick == -227148
		for name, p := range map[string]pools.Pool{
			"concentrated": pools.NewConcentratedPool(concentratedKey, concentratedState(activeTick)),
			"mev capture":  pools.NewMevCapturePool(concentratedKey, concentratedState(activeTick)),
			"boosted fees": pools.NewBoostedFeesPool(concentratedKey,
				pools.NewBoostedFeesPoolState(concentratedState(activeTick), timedState)),
			"ve33 concentrated": pools.NewVe33Pool(pools.NewConcentratedPool(concentratedKey,
				concentratedState(activeTick)), 0),
		} {
			require.Equal(t, outside, activeTickOutsideTickBounds(p), "%s at tick %d", name, activeTick)
		}
	}

	require.False(t, activeTickOutsideTickBounds(fullRange))
	require.False(t, activeTickOutsideTickBounds(pools.NewVe33Pool(fullRange, 0)))
}
