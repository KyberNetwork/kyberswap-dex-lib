package arcade

import (
	"bytes"
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

const multicallAggregateABIJson = `[{"type":"function","name":"aggregate","stateMutability":"payable",
	"inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],
	"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"returnData","type":"bytes[]"}]}]`

const (
	stubUsdc  = "0x3600000000000000000000000000000000000000"
	stubToken = "0x43caace3d7bc72b25e32d2b81b1ee28f447e4ffd"
)

type multicallCall struct {
	Target   common.Address
	CallData []byte
}

// hookStub answers Multicall3.aggregate the way an ArcadeHook deployment would for one
// graduated PUMP launch. hasFeeObs false is v2: the getter does not exist there, the
// inner call reverts and aggregate reverts with it, failing the whole batch.
type hookStub struct {
	t         *testing.T
	hasFeeObs bool
	called    []string
}

func (s *hookStub) answer(method string) []any {
	switch method {
	case "curveStates":
		return []any{big.NewInt(5_500_000_000), big.NewInt(13_472_572_511), tokens(777_000_000),
			uint8(ModePump), uint8(StatusGraduated), common.Address{}, common.Address{}, uint16(0)}
	case "feeObs":
		return []any{int64(-366_089_500), big.NewInt(-373_573), uint32(1_789_000_000), true}
	case "USDC":
		return []any{common.HexToAddress(stubUsdc)}
	case "registeredLaunches":
		return []any{false} // asked about token0, which is USDC here
	case "clankerMaxBuyBps":
		return []any{uint16(100)}
	case "quoteAssetOf":
		return []any{common.Address{}}
	case "clankerPos":
		return []any{big.NewInt(-887_200), big.NewInt(887_200), true, uint64(1_789_446_515)}
	}
	s.t.Fatalf("unexpected call %s", method)
	return nil
}

func (s *hookStub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID     json.RawMessage   `json:"id"`
		Method string            `json:"method"`
		Params []json.RawMessage `json:"params"`
	}
	require.NoError(s.t, json.NewDecoder(r.Body).Decode(&req))
	require.Equal(s.t, "eth_call", req.Method)
	var msg struct {
		Input hexutil.Bytes `json:"input"`
		Data  hexutil.Bytes `json:"data"`
	}
	require.NoError(s.t, json.Unmarshal(req.Params[0], &msg))
	calldata := msg.Input
	if len(calldata) == 0 {
		calldata = msg.Data
	}

	multicall, err := abi.JSON(strings.NewReader(multicallAggregateABIJson))
	require.NoError(s.t, err)
	aggregate := multicall.Methods["aggregate"]
	require.True(s.t, bytes.Equal(aggregate.ID, calldata[:4]))
	in, err := aggregate.Inputs.Unpack(calldata[4:])
	require.NoError(s.t, err)
	calls := *abi.ConvertType(in[0], new([]multicallCall)).(*[]multicallCall)

	reply := func(body string) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,` + body + `}`))
	}
	returnData := make([][]byte, 0, len(calls))
	for _, c := range calls {
		m, err := ArcadeHookABI.MethodById(c.CallData[:4])
		require.NoError(s.t, err)
		s.called = append(s.called, m.Name)
		if m.Name == "feeObs" && !s.hasFeeObs {
			reply(`"error":{"code":3,"message":"execution reverted"}`)
			return
		}
		out, err := m.Outputs.Pack(s.answer(m.Name)...)
		require.NoError(s.t, err)
		returnData = append(returnData, out)
	}
	out, err := aggregate.Outputs.Pack(big.NewInt(21_600_000), returnData)
	require.NoError(s.t, err)
	reply(`"result":"` + hexutil.Encode(out) + `"`)
}

func trackAgainst(t *testing.T, hook common.Address, stub *hookStub) (*Hook, error) {
	t.Helper()
	server := httptest.NewServer(stub)
	defer server.Close()
	client := ethrpc.New(server.URL).
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	h := &Hook{Hook: &uniswapv4.BaseHook{}}
	_, err := h.Track(context.Background(), &uniswapv4.HookParam{
		RpcClient:   client,
		HookAddress: hook,
		Pool: &entity.Pool{
			Address: "0x0299bf2d50bf71d6708c6e98f17f1c2b073cbe0db090629198602efb51cea1f2",
			Tokens:  []*entity.PoolToken{{Address: stubUsdc}, {Address: stubToken}},
		},
	})
	return h, err
}

// v2 removed the feeObs getter. Track must not ask for it there, or every v2 pool would
// stay untracked; on v1 it is still what prices a graduated PUMP pool.
func TestTrack_FeeObsOnlyOnV1(t *testing.T) {
	v2 := &hookStub{t: t, hasFeeObs: false}
	h, err := trackAgainst(t, HookV2, v2)
	require.NoError(t, err)
	assert.NotContains(t, v2.called, "feeObs")
	assert.Equal(t, Extra{
		Tracked: true, Mode: ModePump, Status: StatusGraduated, NoSwapDelta: true,
		UsdcIsCurrency0: true, QuoteIsCurrency0: true, LaunchedAt: 1_789_446_515, BuyCapEnabled: true,
	}, h.Extra)

	v1 := &hookStub{t: t, hasFeeObs: true}
	h, err = trackAgainst(t, HookV1, v1)
	require.NoError(t, err)
	assert.Contains(t, v1.called, "feeObs")
	assert.Equal(t, Extra{
		Tracked: true, Mode: ModePump, Status: StatusGraduated,
		UsdcIsCurrency0: true, ObsInit: true, EmaTickE3: -366_089_500, GradMcapTick: -373_573,
		QuoteIsCurrency0: true, LaunchedAt: 1_789_446_515, BuyCapEnabled: true,
	}, h.Extra)
	assert.Equal(t, int64(78), h.PumpFeeBps())

	// The stub is faithful: the v1 batch really does fail against a v2 hook.
	_, err = trackAgainst(t, HookV1, &hookStub{t: t, hasFeeObs: false})
	require.Error(t, err)
}
