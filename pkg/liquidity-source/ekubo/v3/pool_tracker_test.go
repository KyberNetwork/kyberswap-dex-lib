package ekubov3

import (
	"context"
	"encoding/binary"
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

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/pools"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type (
	PoolTrackerTestSuite struct {
		suite.Suite

		tracker *PoolTracker
	}

	testcase struct {
		name               string
		txHash             string
		blockTimestamp     uint64
		poolKey            *pools.AnyPoolKey
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
				Core:          common.HexToAddress("0x00000000000014aA86C5d3c41765bb24e11bd701"),
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

			blockNumber, logs := ts.getTxLogs(t, tc.txHash)
			newPoolState, err := ts.tracker.GetNewPoolState(
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
				name:   "Add base pool liquidity",
				txHash: "0x2757427086944621c7fb8eca63a01809be4c76bb5b7b32596ced53d7fd17a691",
				poolKey: &pools.AnyPoolKey{PoolKey: pools.NewPoolKey(
					common.HexToAddress(valueobject.ZeroAddress),
					common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
					pools.NewPoolConfig(common.Address{}, 9223372036854775, pools.PoolTypeConfig(pools.NewConcentratedPoolTypeConfig(1000))),
				)},
				extensionType: ExtensionTypeBase,
				// State after pool initialization https://etherscan.io/tx/0x2757427086944621c7fb8eca63a01809be4c76bb5b7b32596ced53d7fd17a691#eventlog#114
				stateBefore: &pools.BasePoolState{
					BasePoolSwapState: &pools.BasePoolSwapState{
						SqrtRatio:       math.FloatSqrtRatioToFixed(uint256.MustFromHex("0x4000e4ac4ee732e5c7c0529d")),
						Liquidity:       new(uint256.Int),
						ActiveTickIndex: 0,
					},
					SortedTicks: []pools.Tick{
						{Number: math.MinTick, LiquidityDelta: new(int256.Int)},
						{Number: math.MaxTick, LiquidityDelta: new(int256.Int)},
					},
					TickBounds: [2]int32{math.MinTick, math.MaxTick},
					ActiveTick: int32(binary.BigEndian.Uint32([]byte("\xfe\xd4\x69\x15"))),
				},
				// Position update https://etherscan.io/tx/0x2757427086944621c7fb8eca63a01809be4c76bb5b7b32596ced53d7fd17a691#eventlog#116
				expectedStateAfter: &pools.BasePoolState{
					BasePoolSwapState: &pools.BasePoolSwapState{
						SqrtRatio:       math.FloatSqrtRatioToFixed(uint256.MustFromHex("0x4000e4ac4ee732e5c7c0529d")),
						Liquidity:       uint256.NewInt(22875426408333),
						ActiveTickIndex: 1,
					},
					SortedTicks: []pools.Tick{
						{Number: math.MinTick, LiquidityDelta: new(int256.Int)},
						{Number: int32(binary.BigEndian.Uint32([]byte("\xFE\xD4\x2A\x30"))), LiquidityDelta: int256.NewInt(22875426408333)},
						{Number: int32(binary.BigEndian.Uint32([]byte("\xFE\xD4\xA7\x30"))), LiquidityDelta: int256.NewInt(-22875426408333)},
						{Number: math.MaxTick, LiquidityDelta: new(int256.Int)},
					},
					TickBounds: [2]int32{math.MinTick, math.MaxTick},
					ActiveTick: int32(binary.BigEndian.Uint32([]byte("\xfe\xd4\x69\x15"))),
				},
			},
		})
	})
}

func (ts *PoolTrackerTestSuite) TestSwapped() {
	ts.run([]*testcase{
		{
			name:   "Swap",
			txHash: "0xee56e1f3bad803bd857fb118e55d7eabb5368a94ae8f11e83724278f474294ca",
			poolKey: &pools.AnyPoolKey{PoolKey: pools.NewPoolKey(
				common.Address{},
				common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
				pools.NewPoolConfig(common.Address{}, 9223372036854775, pools.PoolTypeConfig(pools.NewConcentratedPoolTypeConfig(1000))),
			),
			},
			extensionType: ExtensionTypeBase,
			stateBefore: &pools.BasePoolState{
				BasePoolSwapState: &pools.BasePoolSwapState{
					SqrtRatio:       big256.New("18552164211672086963009471320686592"),
					Liquidity:       uint256.NewInt(17156571521907),
					ActiveTickIndex: 1,
				},
				SortedTicks: []pools.Tick{
					{Number: math.MinTick, LiquidityDelta: new(int256.Int)},
					{Number: -19650000, LiquidityDelta: int256.NewInt(17156571521907)},
					{Number: -19618000, LiquidityDelta: int256.NewInt(-17156571521907)},
					{Number: math.MaxTick, LiquidityDelta: new(int256.Int)},
				},
				TickBounds: [2]int32{math.MinTick, math.MaxTick},
				ActiveTick: -19633899,
			},
			expectedStateAfter: &pools.BasePoolState{
				BasePoolSwapState: &pools.BasePoolSwapState{
					SqrtRatio:       big256.New("18551574977108402281445297823416320"),
					Liquidity:       uint256.NewInt(17156571521907),
					ActiveTickIndex: 1,
				},
				SortedTicks: []pools.Tick{
					{Number: math.MinTick, LiquidityDelta: new(int256.Int)},
					{Number: -19650000, LiquidityDelta: int256.NewInt(17156571521907)},
					{Number: -19618000, LiquidityDelta: int256.NewInt(-17156571521907)},
					{Number: math.MaxTick, LiquidityDelta: new(int256.Int)},
				},
				TickBounds: [2]int32{math.MinTick, math.MaxTick},
				ActiveTick: -19633963,
			},
		},
	})
}

func (ts *PoolTrackerTestSuite) TestVirtualOrdersExecutedAndOrderUpdated() {
	ts.run([]*testcase{
		{
			name:           "Execute virtual orders & stop order",
			txHash:         "0xde6812e959a49e245f15714d1b50571f43ca7711c91d2df1087178a38bc554b7",
			blockTimestamp: 1767625571,
			poolKey: &pools.AnyPoolKey{PoolKey: pools.NewPoolKey(
				common.Address{},
				common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
				pools.NewPoolConfig(common.HexToAddress("0xd4F1060cB9c1A13e1d2d20379b8aa2cF7541eD9b"),
					55340232221128654, pools.PoolTypeConfig(pools.NewFullRangePoolTypeConfig())),
			),
			},
			extensionType: ExtensionTypeTwamm,
			stateBefore: &pools.TwammPoolState{
				FullRangePoolState: &pools.FullRangePoolState{
					FullRangePoolSwapState: &pools.FullRangePoolSwapState{
						SqrtRatio: big256.New("19112726775014745474545736843526144"),
					},
					Liquidity: uint256.NewInt(2670594),
				},
				Token0SaleRate: new(uint256.Int),
				Token1SaleRate: uint256.NewInt(3744848),
				VirtualOrderDeltas: []pools.TwammSaleRateDelta{
					{Time: 1767636992, SaleRateDelta0: new(int256.Int), SaleRateDelta1: int256.NewInt(-3744848)},
				},
				LastExecutionTime: 1767625523,
			},
			expectedStateAfter: &pools.TwammPoolState{
				FullRangePoolState: &pools.FullRangePoolState{
					FullRangePoolSwapState: &pools.FullRangePoolSwapState{
						SqrtRatio: big256.New("19112726775014745474545736843526144"),
					},
					Liquidity: uint256.NewInt(2670594),
				},
				Token0SaleRate:     new(uint256.Int),
				Token1SaleRate:     new(uint256.Int),
				VirtualOrderDeltas: []pools.TwammSaleRateDelta{},
				LastExecutionTime:  1767625571,
			},
		},
	})
}

func (ts *PoolTrackerTestSuite) getTxLogs(t *testing.T, txHash string) (uint64, []types.Log) {
	receipt, err := ts.tracker.ethrpcClient.
		GetETHClient().
		TransactionReceipt(context.Background(), common.HexToHash(txHash))
	require.NoError(t, err)

	logs := lo.Map(receipt.Logs, func(log *types.Log, _ int) types.Log {
		return *log
	})

	return receipt.BlockNumber.Uint64(), logs
}

func (ts *PoolTrackerTestSuite) SetupSuite() {
	ts.tracker = NewPoolTracker(
		MainnetConfig,
		ethrpc.New("https://ethereum.drpc.org").
			SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")),
	)
}

func TestPoolTrackerTestSuite(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	suite.Run(t, new(PoolTrackerTestSuite))
}
