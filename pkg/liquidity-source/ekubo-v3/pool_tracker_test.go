package ekubov3

import (
	"context"
	"encoding/binary"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo-v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo-v3/pools"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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
				Core:          MainnetConfig.Core,
				ExtensionType: tc.extensionType,
				PoolKey:       tc.poolKey,
			}

			staticExtraJson, err := json.Marshal(&staticExtra)
			require.NoError(t, err)

			p := entity.Pool{
				Tokens: []*entity.PoolToken{
					{Address: FromEkuboAddress(tc.poolKey.Token0.String(), MainnetConfig.ChainId)},
					{Address: FromEkuboAddress(tc.poolKey.Token1.String(), MainnetConfig.ChainId)},
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
		/*{
			name:   "Multiswap",
			txHash: "0xc401cc3007a2c0efd705c4c0dee5690ce8592858476b32cda8a4b000ceda0f24",
			poolKey: &pools.AnyPoolKey{PoolKey: pools.NewPoolKey(
				common.Address{},
				common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
				pools.NewPoolConfig(common.Address{}, 55340232221128654, pools.PoolTypeConfig(pools.NewConcentratedPoolTypeConfig(5982))),
			),
			},
			extensionType: ExtensionTypeBase,
			// State after position update https://etherscan.io/tx/0x6746c17c05cf4e8ba61dd57ef617fbe722b54e21b2ee98607b95fccb8f1a9ab0#eventlog#425
			stateBefore: &pools.BasePoolState{
				BasePoolSwapState: &pools.BasePoolSwapState{
					SqrtRatio:       big256.New("14918731339943421144221696791674880"),
					Liquidity:       uint256.NewInt(65496697411278),
					ActiveTickIndex: 1,
				},
				SortedTicks: []pools.Tick{
					{Number: math.MinTick, LiquidityDelta: new(int256.Int)},
					{Number: -20452458, LiquidityDelta: int256.NewInt(65496697411278)},
					{Number: -19686762, LiquidityDelta: int256.NewInt(-65496697411278)},
					{Number: math.MaxTick, LiquidityDelta: new(int256.Int)},
				},
				TickBounds: [2]int32{math.MinTick, math.MaxTick},
				ActiveTick: -20069837,
			},
			expectedStateAfter: &pools.BasePoolState{
				BasePoolSwapState: &pools.BasePoolSwapState{
					SqrtRatio:       big256.New("14918630557421420908805229423624192"),
					Liquidity:       uint256.NewInt(65496697411278),
					ActiveTickIndex: 1,
				},
				SortedTicks: []pools.Tick{
					{Number: math.MinTick, LiquidityDelta: new(int256.Int)},
					{Number: -20452458, LiquidityDelta: int256.NewInt(65496697411278)},
					{Number: -19686762, LiquidityDelta: int256.NewInt(-65496697411278)},
					{Number: math.MaxTick, LiquidityDelta: new(int256.Int)},
				},
				TickBounds: [2]int32{math.MinTick, math.MaxTick},
				ActiveTick: -20069851,
			},
		},*/
	})
}

func (ts *PoolTrackerTestSuite) TestVirtualOrdersExecutedAndOrderUpdated() {
	ts.run([]*testcase{
		/*{
			name:           "Execute virtual orders & create order",
			txHash:         "0xbd9e24145c6e3c936c7617d2a7756a0a7d1b3cf491e145d21f201a06899b1f01",
			blockTimestamp: 1743800039,
			poolKey: &pools.AnyPoolKey{PoolKey: pools.NewPoolKey(
				common.Address{},
				common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
				pools.NewPoolConfig(MainnetConfig.Twamm, 9223372036854775, pools.PoolTypeConfig(pools.NewFullRangePoolTypeConfig())),
			),
			},

			extensionType: ExtensionTypeTwamm,
			stateBefore: &pools.TwammPoolState{
				FullRangePoolState: &pools.FullRangePoolState{
					FullRangePoolSwapState: &pools.FullRangePoolSwapState{
						SqrtRatio: big256.New("14505089304818766342758077000843264"),
					},
					Liquidity: uint256.NewInt(42626626336982),
				},
				Token0SaleRate:     new(uint256.Int),
				Token1SaleRate:     new(uint256.Int),
				VirtualOrderDeltas: []pools.TwammSaleRateDelta{},
				LastExecutionTime:  1743799559,
			},
			expectedStateAfter: &pools.TwammPoolState{
				FullRangePoolState: &pools.FullRangePoolState{
					FullRangePoolSwapState: &pools.FullRangePoolSwapState{
						SqrtRatio: big256.New("14505089304818766342758077000843264"),
					},
					Liquidity: uint256.NewInt(42626626336982),
				},
				Token0SaleRate: big256.New("90639807871689353170834"),
				Token1SaleRate: new(uint256.Int),
				VirtualOrderDeltas: []pools.TwammSaleRateDelta{
					{
						Time:           1743847424,
						SaleRateDelta0: big256.SNew("-90639807871689353170834"),
						SaleRateDelta1: new(int256.Int),
					},
				},
				LastExecutionTime: 1743800039,
			},
		},*/
	})
}

func (ts *PoolTrackerTestSuite) getTxLogs(t *testing.T, txHash string) (uint64, []types.Log) {
	receipt, err := ts.tracker.ethrpcClient.
		GetETHClient().
		TransactionReceipt(context.Background(), common.HexToHash(txHash))
	require.NoError(t, err)

	logs := make([]types.Log, len(receipt.Logs))
	for _, log := range receipt.Logs {
		logs = append(logs, *log)
	}

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
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
	suite.Run(t, new(PoolTrackerTestSuite))
}
