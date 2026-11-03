package kuruobv2

import (
	"context"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	testMulticall = "0xca11bde05977b3631167028862be2a173976ca11"
	testCore      = "0x264123a9b96cc1193944cc25c188f1bd3dfa670e"
	testLimiter   = "0xf059f4aab49bd3afb62134145eeaf38fe103f0ea"
	testBase      = "0x01bff41798a0bcf287b996046ca68b395dbc1071"
	testQuote     = "0x754704bc059f8c67012fed69bc8a327a5aafb603"
)

var multicallABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(`[{"name":"tryBlockAndAggregate","type":"function","inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"blockHash","type":"bytes32"},{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}]`))
	if err != nil {
		panic(err)
	}
	return a
}()

// answer fakes one subcall; stale lists the tokens whose requireFreshPrice reverts InvalidWithdrawalPrice.
func answer(t *testing.T, target common.Address, data []byte, stale map[common.Address]bool) (bool, []byte) {
	method, err := kuruV2ABI.MethodById(data)
	require.NoError(t, err)
	pack := func(v ...any) (bool, []byte) {
		out, err := method.Outputs.Pack(v...)
		require.NoError(t, err)
		return true, out
	}
	switch method.Name {
	case "getL2Book":
		return pack([]uint32{100, 0}, []*big.Int{big.NewInt(1e15), big.NewInt(0)},
			[]uint32{101, 0}, []*big.Int{big.NewInt(1e15), big.NewInt(0)})
	case "takerFeePps", "defaultSpotTakerFeePps":
		return pack(big.NewInt(1000))
	case "marketState":
		return pack(uint8(marketStateLive))
	case "protocolPaused":
		return pack(false)
	case "withdrawalLimiter":
		return pack(common.HexToAddress(testLimiter))
	case "requireFreshPrice":
		require.Equal(t, common.HexToAddress(testLimiter), target)
		args, err := method.Inputs.Unpack(data[4:])
		require.NoError(t, err)
		if stale[args[0].(common.Address)] {
			return false, common.FromHex("0x393843a9")
		}
		return true, nil
	}
	t.Fatalf("unexpected method %s", method.Name)
	return false, nil
}

func fakeClient(t *testing.T, stale map[common.Address]bool) *ethrpc.Client {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.Unmarshal(body, &req))
		var call struct {
			To    common.Address `json:"to"`
			Input hexutil.Bytes  `json:"input"`
			Data  hexutil.Bytes  `json:"data"`
		}
		require.NoError(t, json.Unmarshal(req.Params[0], &call))
		data := append(call.Input, call.Data...)
		var result []byte
		if call.To == common.HexToAddress(testMulticall) {
			in, err := multicallABI.Methods["tryBlockAndAggregate"].Inputs.Unpack(data[4:])
			require.NoError(t, err)
			calls := *abi.ConvertType(in[1], new([]ethrpc.MultiCallParam)).(*[]ethrpc.MultiCallParam)
			type item struct {
				Success    bool
				ReturnData []byte
			}
			items := make([]item, len(calls))
			for i, c := range calls {
				items[i].Success, items[i].ReturnData = answer(t, c.Target, c.CallData, stale)
			}
			result, err = multicallABI.Methods["tryBlockAndAggregate"].Outputs.Pack(big.NewInt(7), [32]byte{}, items)
			require.NoError(t, err)
		} else {
			_, result = answer(t, call.To, data, stale)
		}
		out, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": hexutil.Encode(result)})
		_, _ = w.Write(out)
	}))
	t.Cleanup(srv.Close)
	return ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress(testMulticall))
}

// Every swap withdraws both tokens, and the limiter reverts InvalidWithdrawalPrice for a token with a
// stale price feed. The tracker must then publish an empty book instead of a quote that can't settle.
func TestTrackerEmptiesBookOnStaleFeed(t *testing.T) {
	native := valueobject.WrappedNativeMap[valueobject.ChainIDMonad]
	for name, tc := range map[string]struct {
		stale    []string
		live     bool
		tokens   [2]string
		hasNativ bool
	}{
		"fresh":                {nil, true, [2]string{testBase, testQuote}, false},
		"stale base":           {[]string{testBase}, false, [2]string{testBase, testQuote}, false},
		"stale quote":          {[]string{testQuote}, false, [2]string{testBase, testQuote}, false},
		"native keyed as zero": {[]string{valueobject.ZeroAddress}, false, [2]string{native, testQuote}, true},
		"wrapped is not zero":  {[]string{native}, true, [2]string{native, testQuote}, true},
	} {
		t.Run(name, func(t *testing.T) {
			stale := map[common.Address]bool{}
			for _, s := range tc.stale {
				stale[common.HexToAddress(s)] = true
			}
			se, _ := json.Marshal(StaticExtra{PricePrecision: 100, SizePrecision: uint256.NewInt(1e10), BaseSizeMultiplier: uint256.NewInt(1e6),
				QuoteDecimals: 6, HasNative: tc.hasNativ})
			tracker := NewPoolTracker(&Config{DexID: DexType, ChainId: valueobject.ChainIDMonad, AccountCore: testCore},
				fakeClient(t, stale))
			p, err := tracker.GetNewPoolState(context.Background(), entity.Pool{Address: "0xabc",
				Tokens: []*entity.PoolToken{{Address: tc.tokens[0]}, {Address: tc.tokens[1]}}, StaticExtra: string(se)},
				poolpkg.GetNewPoolStateParams{})
			require.NoError(t, err)
			var extra Extra
			require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
			require.Equal(t, tc.live, len(extra.Bids) > 0 && len(extra.Asks) > 0)
			require.Equal(t, tc.live, p.Reserves[0] != "0" && p.Reserves[1] != "0")
			require.EqualValues(t, 7, p.BlockNumber)
			require.InDelta(t, time.Now().Unix(), p.Timestamp, 5) // the simulator's StaleCheck reads this
		})
	}
}
