package ekubo

import (
	"bytes"
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/pools"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListTrackerTestSuite struct {
	suite.Suite

	tracker *PoolTracker
}

type testcase struct {
	name               string
	txHash             string
	blockTimestamp     uint64
	poolKey            *pools.PoolKey
	extensionType      ExtensionType
	stateBefore        any
	expectedStateAfter any
}

func (ts *PoolListTrackerTestSuite) run(cases []*testcase) {
	t := ts.T()

	for _, tc := range cases {
		ts.Run(tc.name, func() {
			extraJson, err := json.Marshal(tc.stateBefore)
			require.NoError(t, err)

			staticExtra := StaticExtra{
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

func (ts *PoolListTrackerTestSuite) TestPositionUpdated() {
	ts.Run("PositionUpdated", func() {
		ts.run([]*testcase{
			{
				name:   "Add base pool liquidity",
				txHash: "0x6746c17c05cf4e8ba61dd57ef617fbe722b54e21b2ee98607b95fccb8f1a9ab0",
				poolKey: &pools.PoolKey{
					Token0: common.Address{},
					Token1: common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
					Config: pools.PoolConfig{
						Fee:         55340232221128654,
						TickSpacing: 5982,
						Extension:   common.Address{},
					},
				},
				extensionType: ExtensionTypeBase,
				// State after pool initialization https://etherscan.io/tx/0x6746c17c05cf4e8ba61dd57ef617fbe722b54e21b2ee98607b95fccb8f1a9ab0#eventlog#423
				stateBefore: &pools.BasePoolState{
					BasePoolSwapState: &pools.BasePoolSwapState{
						SqrtRatio:       big256.New("14918731339943421144221696791674880"),
						Liquidity:       new(uint256.Int),
						ActiveTickIndex: 0,
					},
					SortedTicks: []pools.Tick{
						{Number: math.MinTick, LiquidityDelta: new(int256.Int)},
						{Number: math.MaxTick, LiquidityDelta: new(int256.Int)},
					},
					TickBounds: [2]int32{math.MinTick, math.MaxTick},
					ActiveTick: -20069837,
				},
				// Position update https://etherscan.io/tx/0x6746c17c05cf4e8ba61dd57ef617fbe722b54e21b2ee98607b95fccb8f1a9ab0#eventlog#425
				expectedStateAfter: &pools.BasePoolState{
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
			},
		})
	})
}

func (ts *PoolListTrackerTestSuite) TestSwapped() {
	ts.run([]*testcase{
		{
			name:   "Multiswap",
			txHash: "0xc401cc3007a2c0efd705c4c0dee5690ce8592858476b32cda8a4b000ceda0f24",
			poolKey: &pools.PoolKey{
				Token0: common.Address{},
				Token1: common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
				Config: pools.PoolConfig{
					Fee:         55340232221128654,
					TickSpacing: 5982,
					Extension:   common.Address{},
				},
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
		},
	})
}

func (ts *PoolListTrackerTestSuite) TestVirtualOrdersExecutedAndOrderUpdated() {
	ts.run([]*testcase{
		{
			name:           "Execute virtual orders & create order",
			txHash:         "0xbd9e24145c6e3c936c7617d2a7756a0a7d1b3cf491e145d21f201a06899b1f01",
			blockTimestamp: 1743800039,
			poolKey: &pools.PoolKey{
				Token0: common.Address{},
				Token1: common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
				Config: pools.PoolConfig{
					Fee:         9223372036854775,
					TickSpacing: 0,
					Extension:   MainnetConfig.Twamm,
				},
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
		},
	})
}

func (ts *PoolListTrackerTestSuite) getTxLogs(t *testing.T, txHash string) (uint64, []types.Log) {
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

func (ts *PoolListTrackerTestSuite) SetupSuite() {
	ts.tracker = NewPoolTracker(
		&MainnetConfig,
		ethrpc.New("https://ethereum.drpc.org").
			SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")),
	)
}

func TestPoolListTrackerTestSuite(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	suite.Run(t, new(PoolListTrackerTestSuite))
}

type testBasicQuoteTick struct {
	Number         int32
	LiquidityDelta *big.Int
}

type testBasicQuoteData struct {
	Tick      int32
	SqrtRatio *big.Int
	Liquidity *big.Int
	MinTick   int32
	MaxTick   int32
	Ticks     []testBasicQuoteTick
}

func TestFetchPoolsBasicBatchReturnsMulticallBlockNumber(t *testing.T) {
	t.Parallel()

	const wantBlockNumber = 12345

	quoteMethod := abis.BasicDataFetcherABI.Methods[basicDataFetcherMethod]
	quoteBytes, err := quoteMethod.Outputs.Pack([]testBasicQuoteData{{
		Tick:      0,
		SqrtRatio: new(big.Int).Lsh(big.NewInt(1), 95),
		Liquidity: big.NewInt(1_000_000),
		MinTick:   -887272,
		MaxTick:   887272,
		Ticks:     []testBasicQuoteTick{},
	}})
	require.NoError(t, err)

	uint256Type, err := abi.NewType("uint256", "", nil)
	require.NoError(t, err)
	bytesArrType, err := abi.NewType("bytes[]", "", nil)
	require.NoError(t, err)
	aggregateBytes, err := abi.Arguments{
		{Name: "blockNumber", Type: uint256Type},
		{Name: "returnData", Type: bytesArrType},
	}.Pack(big.NewInt(wantBlockNumber), [][]byte{quoteBytes})
	require.NoError(t, err)

	aggregateSelector := crypto.Keccak256([]byte("aggregate((address,bytes)[])"))[:4]
	aggregateResult := hexutil.Encode(aggregateBytes)
	quoteResult := hexutil.Encode(quoteBytes)

	var aggregateCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var result any = "0x1"
		if req.Method == "eth_call" {
			var call struct {
				Input string `json:"input"`
				Data  string `json:"data"`
			}
			if err := json.Unmarshal(req.Params[0], &call); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			input := call.Input
			if input == "" {
				input = call.Data
			}
			data, err := hexutil.Decode(input)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			switch {
			case bytes.HasPrefix(data, aggregateSelector):
				aggregateCalls.Add(1)
				result = aggregateResult
			case bytes.HasPrefix(data, quoteMethod.ID):
				result = quoteResult
			default:
				http.Error(w, "unexpected call data", http.StatusBadRequest)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	t.Cleanup(srv.Close)

	client := ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress("0x4444444444444444444444444444444444444444"))
	cfg := MainnetConfig
	fetcher := NewDataFetchers(client, &cfg)

	key := pools.NewPoolKey(
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		common.HexToAddress("0x0000000000000000000000000000000000000002"),
		pools.PoolConfig{Fee: 3000, TickSpacing: 60},
	)

	states, err := fetcher.fetchPools(context.Background(), []*pools.PoolKey{key}, nil)
	require.NoError(t, err)
	require.Len(t, states, 1)
	require.Equal(t, int32(1), aggregateCalls.Load())
	require.Equal(t, uint64(wantBlockNumber), states[0].blockNumber)
}
