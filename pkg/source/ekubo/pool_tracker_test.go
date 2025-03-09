package ekubo

import (
	"context"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ekubo/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ekubo/quoting"
	ekubo_pool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ekubo/quoting/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolListTrackerTestSuite struct {
	suite.Suite

	client  *ethclient.Client
	tracker *PoolTracker
}

type testcase struct {
	name               string
	txHash             string
	poolKey            quoting.PoolKey
	extension          ekubo_pool.Extension
	stateBefore        quoting.PoolState
	expectedStateAfter quoting.PoolState
}

func (ts *PoolListTrackerTestSuite) run(cases []*testcase) {
	t := ts.T()

	for _, tc := range cases {
		ts.Run(tc.name, func() {
			extra := Extra{
				State: tc.stateBefore,
			}
			staticExtra := StaticExtra{
				PoolKey:   tc.poolKey,
				Extension: tc.extension,
			}
			newPoolState, err := ts.tracker.GetNewPoolState(
				context.Background(),
				newPool(t, &extra, &staticExtra),
				pool.GetNewPoolStateParams{
					Logs: ts.getTxLogs(t, tc.txHash),
				},
			)
			require.NoError(t, err)

			var poolExtra Extra
			err = json.Unmarshal([]byte(newPoolState.Extra), &poolExtra)
			require.NoError(ts.T(), err, "Failed to unmarshal pool extra")

			require.Equal(t, tc.expectedStateAfter, poolExtra.State)
		})
	}
}

func (ts *PoolListTrackerTestSuite) SetupSuite() {
	ethclient, err := clientFromEnv()
	require.NoError(ts.T(), err)

	ts.client = ethclient

	ethrpc := ethrpc.NewWithClient(ethclient)

	ts.tracker = NewPoolTracker(&SepoliaConfig, ethrpc)
}

func (ts *PoolListTrackerTestSuite) TestSwapped() {
	ts.run([]*testcase{
		{
			name:   "Exact in",
			txHash: "0xd331ee9950e326da0aa886efb8015c82a2b64d6be4ebf0fefba0b8a7eab72fb3",
			poolKey: quoting.NewPoolKey(
				common.HexToAddress("0xd876ec2ee0816c019cc54299a8184e8111694865"),
				common.HexToAddress("0xf7b3e9697fd769104cd6cf653c179fb452505a3e"),
				quoting.Config{
					Fee:         9223372036854775,
					TickSpacing: 1000,
					Extension:   common.Address{},
				},
			),
			extension: ekubo_pool.Base,
			stateBefore: quoting.NewPoolState(
				math.IntFromString("13805080208217298875668"),
				math.IntFromString("340282366920938463463374607431768211456"),
				0,
				[]quoting.Tick{
					{
						Number:         -16000,
						LiquidityDelta: math.IntFromString("13805080208217298875668"),
					},
					{
						Number:         16000,
						LiquidityDelta: math.IntFromString("-13805080208217298875668"),
					},
				},
				[2]int32{-16000, 16000},
			),
			expectedStateAfter: quoting.NewPoolState(
				math.IntFromString("13805080208217298875668"),
				math.IntFromString("340257731960622028004688875521658847232"),
				-145,
				[]quoting.Tick{
					{
						Number:         -16000,
						LiquidityDelta: math.IntFromString("13805080208217298875668"),
					},
					{
						Number:         16000,
						LiquidityDelta: math.IntFromString("-13805080208217298875668"),
					},
				},
				[2]int32{-16000, 16000},
			),
		},
		{
			name:   "Exact out",
			txHash: "0xdca418e6a533c7c53b9a3978c415bac8c594776f82d1848e425a10621682f461",
			poolKey: quoting.NewPoolKey(
				common.HexToAddress("0xd876ec2ee0816c019cc54299a8184e8111694865"),
				common.HexToAddress("0xf7b3e9697fd769104cd6cf653c179fb452505a3e"),
				quoting.Config{
					Fee:         9223372036854775,
					TickSpacing: 1000,
					Extension:   common.Address{},
				},
			),
			extension: ekubo_pool.Base,
			stateBefore: quoting.NewPoolState(
				math.IntFromString("13805080208217298875668"),
				math.IntFromString("340257731960622028004688875521658847232"),
				-145,
				[]quoting.Tick{
					{
						Number:         -16000,
						LiquidityDelta: math.IntFromString("13805080208217298875668"),
					},
					{
						Number:         16000,
						LiquidityDelta: math.IntFromString("-13805080208217298875668"),
					},
				},
				[2]int32{-16000, 16000},
			),
			expectedStateAfter: quoting.NewPoolState(
				math.IntFromString("13805080208217298875668"),
				math.IntFromString("340233082892178485771514743615339364352"),
				-290,
				[]quoting.Tick{
					{
						Number:         -16000,
						LiquidityDelta: math.IntFromString("13805080208217298875668"),
					},
					{
						Number:         16000,
						LiquidityDelta: math.IntFromString("-13805080208217298875668"),
					},
				},
				[2]int32{-16000, 16000},
			),
		},
	})
}

func (ts *PoolListTrackerTestSuite) TestPositionUpdated() {
	ts.Run("PositionUpdated", func() {
		ts.run([]*testcase{
			{
				name:   "Add liquidity",
				txHash: "0x11893f22c56e1f114311edcf23ebb8751f4202a5f7fe9e7a79295b6fd3e263ba",
				poolKey: quoting.NewPoolKey(
					common.Address{},
					common.HexToAddress("0xd876ec2ee0816c019cc54299a8184e8111694865"),
					quoting.Config{
						Fee:         0,
						TickSpacing: 0,
						Extension:   common.HexToAddress(SepoliaConfig.Oracle),
					},
				),
				extension: ekubo_pool.Oracle,
				stateBefore: quoting.NewPoolState(
					math.IntFromString("31622773100538380"),
					math.IntFromString("107606720792549838337692509122489386795008"),
					11512931,
					[]quoting.Tick{
						{
							Number:         math.MinTick,
							LiquidityDelta: math.IntFromString("31622773100538380"),
						},
						{
							Number:         math.MaxTick,
							LiquidityDelta: math.IntFromString("-31622773100538380"),
						},
					},
					[2]int32{math.MinTick, math.MaxTick},
				),
				expectedStateAfter: quoting.NewPoolState(
					math.IntFromString("63245553203367807"),
					math.IntFromString("107606720792549838337692509122489386795008"),
					11512931,
					[]quoting.Tick{
						{
							Number:         math.MinTick,
							LiquidityDelta: math.IntFromString("63245553203367807"),
						},
						{
							Number:         math.MaxTick,
							LiquidityDelta: math.IntFromString("-63245553203367807"),
						},
					},
					[2]int32{math.MinTick, math.MaxTick},
				),
			},
		})
	})
}

func (ts *PoolListTrackerTestSuite) getTxLogs(t *testing.T, txHash string) []types.Log {
	receipt, err := ts.client.TransactionReceipt(context.Background(), common.HexToHash(txHash))
	require.NoError(t, err)

	logs := make([]types.Log, len(receipt.Logs))
	for _, log := range receipt.Logs {
		logs = append(logs, *log)
	}

	return logs
}

func newPool(t *testing.T, extra *Extra, staticExtra *StaticExtra) entity.Pool {
	extraJson, err := json.Marshal(extra)
	require.NoError(t, err)

	staticExtraJson, err := json.Marshal(staticExtra)
	require.NoError(t, err)

	return entity.Pool{
		Extra:       string(extraJson),
		StaticExtra: string(staticExtraJson),
	}
}

func TestPoolListTrackerTestSuite(t *testing.T) {
	suite.Run(t, new(PoolListTrackerTestSuite))
}
