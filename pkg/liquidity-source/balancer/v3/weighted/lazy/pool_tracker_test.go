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

var testMulticallABI = lo.Must(abi.JSON(strings.NewReader(`[
 {"type":"function","name":"tryBlockAndAggregate","inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"blockHash","type":"bytes32"},{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}
]`)))

type mockWeightedRPC struct {
	version          string
	minTokenBalances []*big.Int
}

func (s mockWeightedRPC) handle(t *testing.T, callData []byte) (bool, []byte) {
	t.Helper()
	if m, err := weighted.PoolABI.MethodById(callData[:4]); err == nil {
		switch m.Name {
		case weighted.PoolMethodGetNormalizedWeights:
			return true, lo.Must(m.Outputs.Pack([]*big.Int{big.NewInt(8e17), big.NewInt(2e17)}))
		case weighted.PoolMethodVersion:
			if s.version == "" {
				return false, nil
			}
			return true, lo.Must(m.Outputs.Pack(s.version))
		case weighted.PoolMethodGetMinTokenBalances:
			if s.minTokenBalances == nil {
				return false, nil
			}
			return true, lo.Must(m.Outputs.Pack(s.minTokenBalances))
		}
	}
	m := lo.Must(shared.VaultExplorerABI.MethodById(callData[:4]))
	switch m.Name {
	case shared.VaultMethodGetHooksConfig:
		return true, lo.Must(m.Outputs.Pack(shared.HooksConfigRPC{}.HooksConfigData))
	case shared.VaultMethodGetStaticSwapFeePercentage:
		return true, lo.Must(m.Outputs.Pack(big.NewInt(3e15)))
	case shared.VaultMethodGetPoolData:
		var pd shared.PoolDataRPC
		bal1 := new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18))
		pd.PoolData.Tokens = []common.Address{common.HexToAddress("0xaa"), common.HexToAddress("0xbb")}
		pd.PoolData.TokenInfo = make([]shared.TokenInfo, 2)
		pd.PoolData.BalancesRaw = []*big.Int{big.NewInt(10_000_000), bal1}
		pd.PoolData.BalancesLiveScaled18 = []*big.Int{big.NewInt(10_000_000), bal1}
		pd.PoolData.TokenRates = []*big.Int{big.NewInt(1e18), big.NewInt(1e18)}
		pd.PoolData.DecimalScalingFactors = []*big.Int{big.NewInt(1), big.NewInt(1)}
		return true, lo.Must(m.Outputs.Pack(pd.PoolData))
	default:
		return true, lo.Must(m.Outputs.Pack(false))
	}
}

func (s mockWeightedRPC) client(t *testing.T) *ethrpc.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ID     json.RawMessage
			Params []json.RawMessage
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		var call struct{ Data, Input hexutil.Bytes }
		require.NoError(t, json.Unmarshal(body.Params[0], &call))
		data := append(call.Input, call.Data...)
		m := lo.Must(testMulticallABI.MethodById(data[:4]))
		args := lo.Must(m.Inputs.Unpack(data[4:]))
		type param struct {
			Target   common.Address
			CallData []byte
		}
		type res struct {
			Success    bool
			ReturnData []byte
		}
		calls := *abi.ConvertType(args[1], new([]param)).(*[]param)
		results := lo.Map(calls, func(c param, _ int) res {
			ok, ret := s.handle(t, c.CallData)
			return res{ok, ret}
		})
		enc := lo.Must(m.Outputs.Pack(big.NewInt(42_000_000), common.Hash{}, results))
		w.Header().Set("content-type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": body.ID, "result": hexutil.Encode(enc)}))
	}))
	t.Cleanup(srv.Close)
	return ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
}

func TestPoolTrackerMinTokenBalances(t *testing.T) {
	t.Parallel()
	const (
		v1JSON = `{"name":"WeightedPool","version":1,"deployment":"20241205-v3-weighted-pool"}`
		v2JSON = `{"name":"WeightedPool","version":2,"deployment":"20261021-weighted-pool"}`
	)
	cfg := &shared.Config{
		DexID:         valueobject.ExchangeFrontierWeighted,
		ChainID:       valueobject.ChainIDBase,
		VaultExplorer: shared.Vault(valueobject.ChainIDBase, valueobject.ExchangeFrontierWeighted).Hex(),
	}
	basePool := entity.Pool{
		Address: "0x2222222222222222222222222222222222222222", Exchange: valueobject.ExchangeFrontierWeighted, Type: weighted.DexType,
		Reserves: entity.PoolReserves{"0", "0"}, StaticExtra: `{"buffs":["",""]}`,
		Tokens: []*entity.PoolToken{{Address: "0x00000000000000000000000000000000000000aa", Swappable: true}, {Address: "0x00000000000000000000000000000000000000bb", Swappable: true}},
	}
	v2Min := []*big.Int{big.NewInt(1_000_000), big.NewInt(1_000_000)}
	v2MinU256 := []*uint256.Int{uint256.NewInt(1_000_000), uint256.NewInt(1_000_000)}

	trackAll := func(t *testing.T, p entity.Pool, mock mockWeightedRPC) ([]entity.Pool, error) {
		c := mock.client(t)
		lt, bt := lo.Must(NewPoolTracker(cfg, c)), lo.Must(weighted.NewPoolTracker(cfg, c))
		p1, err1 := lt.GetNewPoolState(t.Context(), p, pool.GetNewPoolStateParams{})
		p2, err2 := bt.GetNewPoolState(t.Context(), p, pool.GetNewPoolStateParams{})
		req, apply := lo.Must2(lt.LazyNewPoolState(t.Context(), p, pool.GetNewPoolStateParams{}))
		for i, msg := range req.GetCallMsgs() {
			if ok, ret := mock.handle(t, msg.Data); ok {
				require.NoError(t, req.GetUnpacks()[i](ret))
			}
		}
		p3, err3 := apply(big.NewInt(42_000_000))
		require.Equal(t, err1, err2)
		require.Equal(t, err1, err3)
		return []entity.Pool{p1, p2, p3}, err1
	}

	// v1: getMinTokenBalances reverts, version() confirms v1 -> omitted from Extra.
	v1Pools, err := trackAll(t, basePool, mockWeightedRPC{version: v1JSON})
	require.NoError(t, err)
	for _, p := range v1Pools {
		require.NotContains(t, p.Extra, "minTokenBalances")
	}

	// v2: populates minTokenBalances, and preserves them if a later refresh loses getMinTokenBalances/version.
	v2Pools, err := trackAll(t, basePool, mockWeightedRPC{version: v2JSON, minTokenBalances: v2Min})
	require.NoError(t, err)
	refreshed, err := trackAll(t, v2Pools[0], mockWeightedRPC{})
	require.NoError(t, err)
	for _, p := range append(v2Pools, refreshed...) {
		var extra weighted.Extra
		require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
		require.Equal(t, v2MinU256, extra.MinTokenBalances)
	}

	// Transient RPC failures on initial sync never quietly downgrade a v2 pool to v1.
	_, err = trackAll(t, basePool, mockWeightedRPC{version: v2JSON})
	require.ErrorIs(t, err, weighted.ErrMissingMinTokenBalances)
	_, err = trackAll(t, basePool, mockWeightedRPC{})
	require.ErrorIs(t, err, weighted.ErrMissingPoolVersion)
}
