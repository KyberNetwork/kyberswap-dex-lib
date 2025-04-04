package ekubo

import (
	"context"
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	math2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	quoting2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
	ekubo_pool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting/pool"
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
	poolKey            quoting2.PoolKey
	extension          ekubo_pool.Extension
	stateBefore        quoting2.PoolState
	expectedStateAfter quoting2.PoolState
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

	ts.tracker = NewPoolTracker(&MainnetConfig, ethrpc)
}

func (ts *PoolListTrackerTestSuite) TestPositionUpdated() {
	ts.Run("PositionUpdated", func() {
		ts.run([]*testcase{
			{
				name:   "Add liquidity",
				txHash: "0x6746c17c05cf4e8ba61dd57ef617fbe722b54e21b2ee98607b95fccb8f1a9ab0",
				poolKey: quoting2.PoolKey{
					Token0: common.Address{},
					Token1: common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
					Config: quoting2.Config{
						Fee:         55340232221128654,
						TickSpacing: 5982,
						Extension:   common.Address{},
					},
				},
				extension: ekubo_pool.Base,
				// State after pool initialization https://etherscan.io/tx/0x6746c17c05cf4e8ba61dd57ef617fbe722b54e21b2ee98607b95fccb8f1a9ab0#eventlog#423
				stateBefore: quoting2.NewPoolState(
					new(big.Int),
					math2.IntFromString("14918731339943421144221696791674880"),
					-20069837,
					[]quoting2.Tick{
						{
							Number:         math2.MinTick,
							LiquidityDelta: new(big.Int),
						},
						{
							Number:         math2.MaxTick,
							LiquidityDelta: new(big.Int),
						},
					},
					[2]int32{math2.MinTick, math2.MaxTick},
				),
				// Position update https://etherscan.io/tx/0x6746c17c05cf4e8ba61dd57ef617fbe722b54e21b2ee98607b95fccb8f1a9ab0#eventlog#425
				expectedStateAfter: quoting2.NewPoolState(
					big.NewInt(65496697411278),
					math2.IntFromString("14918731339943421144221696791674880"),
					-20069837,
					[]quoting2.Tick{
						{
							Number:         math2.MinTick,
							LiquidityDelta: new(big.Int),
						},
						{
							Number:         -20452458,
							LiquidityDelta: big.NewInt(65496697411278),
						},
						{
							Number:         -19686762,
							LiquidityDelta: big.NewInt(-65496697411278),
						},
						{
							Number:         math2.MaxTick,
							LiquidityDelta: new(big.Int),
						},
					},
					[2]int32{math2.MinTick, math2.MaxTick},
				),
			},
		})
	})
}

func (ts *PoolListTrackerTestSuite) TestSwapped() {
	ts.run([]*testcase{
		{
			name:   "Multiswap",
			txHash: "0xc401cc3007a2c0efd705c4c0dee5690ce8592858476b32cda8a4b000ceda0f24",
			poolKey: quoting2.PoolKey{
				Token0: common.Address{},
				Token1: common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
				Config: quoting2.Config{
					Fee:         55340232221128654,
					TickSpacing: 5982,
					Extension:   common.Address{},
				},
			},
			extension: ekubo_pool.Base,
			// State after position update https://etherscan.io/tx/0x6746c17c05cf4e8ba61dd57ef617fbe722b54e21b2ee98607b95fccb8f1a9ab0#eventlog#425
			stateBefore: quoting2.NewPoolState(
				big.NewInt(65496697411278),
				math2.IntFromString("14918731339943421144221696791674880"),
				-20069837,
				[]quoting2.Tick{
					{
						Number:         math2.MinTick,
						LiquidityDelta: new(big.Int),
					},
					{
						Number:         -20452458,
						LiquidityDelta: big.NewInt(65496697411278),
					},
					{
						Number:         -19686762,
						LiquidityDelta: big.NewInt(-65496697411278),
					},
					{
						Number:         math2.MaxTick,
						LiquidityDelta: new(big.Int),
					},
				},
				[2]int32{math2.MinTick, math2.MaxTick},
			),
			expectedStateAfter: quoting2.NewPoolState(
				big.NewInt(65496697411278),
				math2.IntFromString("14918630557421420908805229423624192"),
				-20069851,
				[]quoting2.Tick{
					{
						Number:         math2.MinTick,
						LiquidityDelta: new(big.Int),
					},
					{
						Number:         -20452458,
						LiquidityDelta: big.NewInt(65496697411278),
					},
					{
						Number:         -19686762,
						LiquidityDelta: big.NewInt(-65496697411278),
					},
					{
						Number:         math2.MaxTick,
						LiquidityDelta: new(big.Int),
					},
				},
				[2]int32{math2.MinTick, math2.MaxTick},
			),
		},
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
