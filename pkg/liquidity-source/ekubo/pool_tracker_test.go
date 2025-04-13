package ekubo

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
	ekubopool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolListTrackerTestSuite struct {
	suite.Suite

	tracker *PoolTracker
}

type testcase struct {
	name               string
	txHash             string
	poolKey            *quoting.PoolKey
	extensionType      ekubopool.ExtensionType
	stateBefore        quoting.PoolState
	expectedStateAfter quoting.PoolState
}

func (ts *PoolListTrackerTestSuite) run(cases []*testcase) {
	t := ts.T()

	for _, tc := range cases {
		ts.Run(tc.name, func() {
			extraJson, err := json.Marshal(Extra{
				tc.stateBefore,
			})
			require.NoError(t, err)

			staticExtraJson, err := json.Marshal(StaticExtra{
				ExtensionType: tc.extensionType,
				PoolKey:       tc.poolKey,
			})
			require.NoError(t, err)

			p := entity.Pool{
				Tokens: []*entity.PoolToken{
					{Address: FromEkuboAddress(tc.poolKey.Token0.String(), MainnetConfig.ChainId)},
					{Address: FromEkuboAddress(tc.poolKey.Token1.String(), MainnetConfig.ChainId)},
				},
				Extra:       string(extraJson),
				StaticExtra: string(staticExtraJson),
			}
			newPoolState, err := ts.tracker.GetNewPoolState(
				context.Background(),
				p,
				pool.GetNewPoolStateParams{Logs: ts.getTxLogs(t, tc.txHash)},
			)
			require.NoError(t, err)

			var poolExtra Extra
			err = json.Unmarshal([]byte(newPoolState.Extra), &poolExtra)
			require.NoError(ts.T(), err, "Failed to unmarshal pool extra")

			require.Equal(t, tc.expectedStateAfter, poolExtra.PoolState)
		})
	}
}

func (ts *PoolListTrackerTestSuite) TestPositionUpdated() {
	ts.Run("PositionUpdated", func() {
		ts.run([]*testcase{
			{
				name:   "Add liquidity",
				txHash: "0x6746c17c05cf4e8ba61dd57ef617fbe722b54e21b2ee98607b95fccb8f1a9ab0",
				poolKey: &quoting.PoolKey{
					Token0: common.Address{},
					Token1: common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
					Config: quoting.Config{
						Fee:         55340232221128654,
						TickSpacing: 5982,
						Extension:   common.Address{},
					},
				},
				extensionType: ekubopool.Base,
				// State after pool initialization https://etherscan.io/tx/0x6746c17c05cf4e8ba61dd57ef617fbe722b54e21b2ee98607b95fccb8f1a9ab0#eventlog#423
				stateBefore: quoting.NewPoolState(
					new(big.Int),
					bignum.NewBig("14918731339943421144221696791674880"),
					-20069837,
					[]quoting.Tick{
						{Number: math.MinTick, LiquidityDelta: new(big.Int)},
						{Number: math.MaxTick, LiquidityDelta: new(big.Int)},
					},
					[2]int32{math.MinTick, math.MaxTick},
				),
				// Position update https://etherscan.io/tx/0x6746c17c05cf4e8ba61dd57ef617fbe722b54e21b2ee98607b95fccb8f1a9ab0#eventlog#425
				expectedStateAfter: quoting.NewPoolState(
					big.NewInt(65496697411278),
					bignum.NewBig("14918731339943421144221696791674880"),
					-20069837,
					[]quoting.Tick{
						{Number: math.MinTick, LiquidityDelta: new(big.Int)},
						{Number: -20452458, LiquidityDelta: big.NewInt(65496697411278)},
						{Number: -19686762, LiquidityDelta: big.NewInt(-65496697411278)},
						{Number: math.MaxTick, LiquidityDelta: new(big.Int)},
					},
					[2]int32{math.MinTick, math.MaxTick},
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
			poolKey: &quoting.PoolKey{
				Token0: common.Address{},
				Token1: common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
				Config: quoting.Config{
					Fee:         55340232221128654,
					TickSpacing: 5982,
					Extension:   common.Address{},
				},
			},
			extensionType: ekubopool.Base,
			// State after position update https://etherscan.io/tx/0x6746c17c05cf4e8ba61dd57ef617fbe722b54e21b2ee98607b95fccb8f1a9ab0#eventlog#425
			stateBefore: quoting.NewPoolState(
				big.NewInt(65496697411278),
				bignum.NewBig("14918731339943421144221696791674880"),
				-20069837,
				[]quoting.Tick{
					{Number: math.MinTick, LiquidityDelta: new(big.Int)},
					{Number: -20452458, LiquidityDelta: big.NewInt(65496697411278)},
					{Number: -19686762, LiquidityDelta: big.NewInt(-65496697411278)},
					{Number: math.MaxTick, LiquidityDelta: new(big.Int)},
				},
				[2]int32{math.MinTick, math.MaxTick},
			),
			expectedStateAfter: quoting.NewPoolState(
				big.NewInt(65496697411278),
				bignum.NewBig("14918630557421420908805229423624192"),
				-20069851,
				[]quoting.Tick{
					{Number: math.MinTick, LiquidityDelta: new(big.Int)},
					{Number: -20452458, LiquidityDelta: big.NewInt(65496697411278)},
					{Number: -19686762, LiquidityDelta: big.NewInt(-65496697411278)},
					{Number: math.MaxTick, LiquidityDelta: new(big.Int)},
				},
				[2]int32{math.MinTick, math.MaxTick},
			),
		},
	})
}

func (ts *PoolListTrackerTestSuite) getTxLogs(t *testing.T, txHash string) []types.Log {
	receipt, err := ts.tracker.ethrpcClient.
		GetETHClient().
		TransactionReceipt(context.Background(), common.HexToHash(txHash))
	require.NoError(t, err)

	logs := make([]types.Log, len(receipt.Logs))
	for _, log := range receipt.Logs {
		logs = append(logs, *log)
	}

	return logs
}

func (ts *PoolListTrackerTestSuite) SetupSuite() {
	ts.tracker = NewPoolTracker(&MainnetConfig, ethrpc.New("https://ethereum.kyberengineering.io"))
}

func TestPoolListTrackerTestSuite(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
	suite.Run(t, new(PoolListTrackerTestSuite))
}
