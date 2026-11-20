package lazy

import (
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
	weighted "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/weighted"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const testMulticall = "0xcA11bde05977b3631167028862bE2a173976CA11"

var testMulticallABI = lo.Must(abi.JSON(strings.NewReader(`[
 {"type":"function","name":"tryBlockAndAggregate","inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"blockHash","type":"bytes32"},{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}
]`)))

type mockWeightedState struct {
	tokens           []common.Address
	balancesRaw      []*big.Int
	balancesScaled18 []*big.Int
	decs             []*big.Int
	rates            []*big.Int
	weights          []*big.Int
	minTokenBalances []*big.Int // nil simulates v1 WeightedPool where getMinTokenBalances reverts
	staticSwapFee    *big.Int
	poolConfigBits   [32]byte
	blockNumber      uint64
}

func (s mockWeightedState) handleCall(t *testing.T, callData []byte) (bool, []byte) {
	t.Helper()

	if m, err := weighted.PoolABI.MethodById(callData[:4]); err == nil {
		switch m.Name {
		case weighted.PoolMethodGetNormalizedWeights:
			b, err := m.Outputs.Pack(s.weights)
			require.NoError(t, err)
			return true, b
		case weighted.PoolMethodGetMinTokenBalances:
			if s.minTokenBalances == nil {
				return false, nil
			}
			b, err := m.Outputs.Pack(s.minTokenBalances)
			require.NoError(t, err)
			return true, b
		}
	}

	m, err := shared.VaultExplorerABI.MethodById(callData[:4])
	require.NoError(t, err)

	switch m.Name {
	case shared.VaultMethodGetHooksConfig:
		var cfg shared.HooksConfigRPC
		b, err := m.Outputs.Pack(cfg.HooksConfigData)
		require.NoError(t, err)
		return true, b
	case shared.VaultMethodGetStaticSwapFeePercentage:
		b, err := m.Outputs.Pack(s.staticSwapFee)
		require.NoError(t, err)
		return true, b
	case shared.VaultMethodGetPoolData:
		var pd shared.PoolDataRPC
		pd.PoolData.PoolConfigBits = s.poolConfigBits
		pd.PoolData.Tokens = s.tokens
		pd.PoolData.TokenInfo = make([]shared.TokenInfo, len(s.tokens))
		pd.PoolData.BalancesRaw = s.balancesRaw
		pd.PoolData.BalancesLiveScaled18 = s.balancesScaled18
		pd.PoolData.TokenRates = s.rates
		pd.PoolData.DecimalScalingFactors = s.decs
		b, err := m.Outputs.Pack(pd.PoolData)
		require.NoError(t, err)
		return true, b
	case shared.VaultMethodIsVaultPaused, shared.VaultMethodIsPoolPaused, shared.VaultMethodIsPoolInRecoveryMode:
		b, err := m.Outputs.Pack(false)
		require.NoError(t, err)
		return true, b
	default:
		t.Fatalf("unexpected RPC call: %s", m.Name)
		return false, nil
	}
}

func newMockWeightedRPC(t *testing.T, state mockWeightedState) *ethrpc.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ID     json.RawMessage
			Params []json.RawMessage
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

		var call struct{ Data, Input hexutil.Bytes }
		require.NoError(t, json.Unmarshal(body.Params[0], &call))
		data := append(call.Input, call.Data...)

		method, err := testMulticallABI.MethodById(data[:4])
		require.NoError(t, err)
		args, err := method.Inputs.Unpack(data[4:])
		require.NoError(t, err)

		type callParam struct {
			Target   common.Address
			CallData []byte
		}
		type callResult struct {
			Success    bool
			ReturnData []byte
		}
		calls := *abi.ConvertType(args[1], new([]callParam)).(*[]callParam)
		results := make([]callResult, len(calls))
		for i, c := range calls {
			ok, ret := state.handleCall(t, c.CallData)
			results[i] = callResult{Success: ok, ReturnData: ret}
		}

		encoded, err := method.Outputs.Pack(new(big.Int).SetUint64(state.blockNumber), common.Hash{}, results)
		require.NoError(t, err)

		w.Header().Set("content-type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      body.ID,
			"result":  hexutil.Encode(encoded),
		}))
	}))
	t.Cleanup(server.Close)
	return ethrpc.New(server.URL).SetMulticallContract(common.HexToAddress(testMulticall))
}

func TestPoolTrackerMinTokenBalances(t *testing.T) {
	t.Parallel()

	token0 := common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
	token1 := common.HexToAddress("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2")
	vaultAddr := shared.Vault(valueobject.ChainIDBase, valueobject.ExchangeFrontierWeighted).Hex()

	baseState := mockWeightedState{
		tokens:           []common.Address{token0, token1},
		balancesRaw:      []*big.Int{big.NewInt(10_000_000), new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18))},
		balancesScaled18: []*big.Int{big.NewInt(10_000_000), new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18))},
		decs:             []*big.Int{big.NewInt(1), big.NewInt(1)},
		rates:            []*big.Int{big.NewInt(1e18), big.NewInt(1e18)},
		weights:          []*big.Int{big.NewInt(8e17), big.NewInt(2e17)},
		staticSwapFee:    big.NewInt(3e15),
		blockNumber:      42_000_000,
	}

	inputPool := entity.Pool{
		Address:  "0x2222222222222222222222222222222222222222",
		Exchange: valueobject.ExchangeFrontierWeighted,
		Type:     weighted.DexType,
		Reserves: entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: hexutil.Encode(token0[:]), Decimals: 18, Swappable: true},
			{Address: hexutil.Encode(token1[:]), Decimals: 18, Swappable: true},
		},
		StaticExtra: `{"buffs":["",""]}`,
	}

	cfg := &shared.Config{
		DexID:         valueobject.ExchangeFrontierWeighted,
		ChainID:       valueobject.ChainIDBase,
		VaultExplorer: vaultAddr,
	}

	cases := []struct {
		name           string
		minBals        []*big.Int
		wantMinBals    []*uint256.Int
		wantExactInOut string
	}{
		{
			name:           "v1 pool reverts getMinTokenBalances and keeps v1 pricing",
			minBals:        nil,
			wantMinBals:    nil,
			wantExactInOut: "5167812005680028660",
		},
		{
			name:           "v2 pool populates minTokenBalances and enables v2 pricing",
			minBals:        []*big.Int{big.NewInt(1_000_000), big.NewInt(1_000_000)},
			wantMinBals:    []*uint256.Int{uint256.NewInt(1_000_000), uint256.NewInt(1_000_000)},
			wantExactInOut: "5167811684339948710",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := baseState
			state.minTokenBalances = tc.minBals
			client := newMockWeightedRPC(t, state)

			lazyTracker, err := NewPoolTracker(cfg, client)
			require.NoError(t, err)
			backupTracker, err := weighted.NewPoolTracker(cfg, client)
			require.NoError(t, err)

			// 1. Direct GetNewPoolState on both lazy and backup trackers.
			lazyPool, err := lazyTracker.GetNewPoolState(t.Context(), inputPool, pool.GetNewPoolStateParams{})
			require.NoError(t, err)
			backupPool, err := backupTracker.GetNewPoolState(t.Context(), inputPool, pool.GetNewPoolStateParams{})
			require.NoError(t, err)

			// 2. LazyNewPoolState path simulating pool-service batch RPC dispatch.
			lazyReq, applyResult, err := lazyTracker.LazyNewPoolState(t.Context(), inputPool, pool.GetNewPoolStateParams{})
			require.NoError(t, err)
			callMsgs := lazyReq.GetCallMsgs()
			unpacks := lazyReq.GetUnpacks()
			require.Equal(t, len(callMsgs), len(unpacks))
			for i, msg := range callMsgs {
				if ok, ret := state.handleCall(t, msg.Data); ok {
					require.NoError(t, unpacks[i](ret))
				}
			}
			batchedPool, err := applyResult(new(big.Int).SetUint64(state.blockNumber))
			require.NoError(t, err)

			for _, tracked := range []entity.Pool{lazyPool, backupPool, batchedPool} {
				var extra weighted.Extra
				require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
				require.Equal(t, tc.wantMinBals, extra.MinTokenBalances)
				if tc.wantMinBals == nil {
					require.NotContains(t, tracked.Extra, "minTokenBalances")
				}

				sim, err := weighted.NewPoolSimulator(pool.FactoryParams{
					EntityPool: tracked,
					ChainID:    valueobject.ChainIDBase,
				})
				require.NoError(t, err)

				swapParams := pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: hexutil.Encode(token0[:]), Amount: big.NewInt(2_000_000)},
					TokenOut:      hexutil.Encode(token1[:]),
				}
				res, err := sim.CalcAmountOut(swapParams)
				require.NoError(t, err)
				require.Equal(t, tc.wantExactInOut, res.TokenAmountOut.Amount.String())
			}
		})
	}
}
