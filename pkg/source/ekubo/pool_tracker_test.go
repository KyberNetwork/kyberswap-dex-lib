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

func (ts *PoolListTrackerTestSuite) SetupTest() {
	ethclient, err := clientFromEnv()
	require.NoError(ts.T(), err)

	ts.client = ethclient

	ethrpc := ethrpc.NewWithClient(ethclient)

	ts.tracker = NewPoolTracker(&SepoliaConfig, ethrpc)
}

func (ts *PoolListTrackerTestSuite) TestGetNewPoolState() {
	ts.Run("Swapped", func() {
		t := ts.T()

		initialState := quoting.NewPoolState(
			big.NewInt(330076895364),
			math.IntFromString("340282366920938463463374607431768211456"),
			0,
			[]quoting.Tick{
				{
					Number:         -64000,
					LiquidityDelta: big.NewInt(317526822448),
				},
				{
					Number:         -16000,
					LiquidityDelta: big.NewInt(12550072916),
				},
				{
					Number:         16000,
					LiquidityDelta: big.NewInt(12550072916),
				},
				{
					Number:         64000,
					LiquidityDelta: big.NewInt(-317526822448),
				},
			},
			[2]int32{-2560000, 2560000},
		)

		newPoolState, err := ts.tracker.GetNewPoolState(context.Background(), newPool(t, initialState), pool.GetNewPoolStateParams{
			Logs: ts.getTxLogs(t, "0x39571b8569625ee326cc5ba71031ce82466a1256964eb840ec6165aea545f3a7"),
		})
		require.NoError(t, err)

		var poolExtra Extra
		err = json.Unmarshal([]byte(newPoolState.Extra), &poolExtra)
		require.NoError(ts.T(), err, "Failed to unmarshal pool extra")

		require.Equal(t, poolExtra.State.Liquidity, initialState.Liquidity)
		require.Equal(t, poolExtra.State.SqrtRatio, math.IntFromString("340283397323946109287174869415798767616"))
		require.Equal(t, poolExtra.State.ActiveTick, int32(6))
	})

	ts.Run("PositionUpdated", func() {
		t := ts.T()

		initialState := quoting.NewPoolState(
			big.NewInt(12550072916),
			math.IntFromString("340282366920938463463374607431768211456"),
			0,
			[]quoting.Tick{
				{
					Number:         -16000,
					LiquidityDelta: big.NewInt(12550072916),
				},
				{
					Number:         16000,
					LiquidityDelta: big.NewInt(12550072916),
				},
			},
			[2]int32{-2560000, 2560000},
		)

		newPool, err := ts.tracker.GetNewPoolState(context.Background(), newPool(t, initialState), pool.GetNewPoolStateParams{
			Logs: ts.getTxLogs(t, "0x4a0bdc9f301bbc398190b439991e0a3acc40841fe209b73563dbedbf04dfc40d"),
		})
		require.NoError(t, err)

		var poolExtra Extra
		err = json.Unmarshal([]byte(newPool.Extra), &poolExtra)
		require.NoError(ts.T(), err, "Failed to unmarshal pool extra")

		newState := poolExtra.State

		require.Equal(t, newState.Liquidity, big.NewInt(330076895364))
		require.Equal(t, newState.SqrtRatio, initialState.SqrtRatio)
		require.Equal(t, poolExtra.State.ActiveTick, initialState.ActiveTick)
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

func newPool(t *testing.T, state quoting.PoolState) entity.Pool {
	extraJson, err := json.Marshal(Extra{
		State: state,
	})
	require.NoError(t, err)

	staticExtraJson, err := json.Marshal(StaticExtra{
		PoolKey: quoting.PoolKey{
			Token0: common.Address{},
			Token1: common.HexToAddress("0xd876ec2ee0816c019cc54299a8184e8111694865"),
			Config: quoting.Config{
				Fee:         9223372036854775,
				TickSpacing: 1000,
				Extension:   common.Address{},
			},
		},
		Extension: ekubo_pool.Base,
	})
	require.NoError(t, err)

	return entity.Pool{
		Extra:       string(extraJson),
		StaticExtra: string(staticExtraJson),
	}
}

func TestPoolListTrackerTestSuite(t *testing.T) {
	suite.Run(t, new(PoolListTrackerTestSuite))
}
