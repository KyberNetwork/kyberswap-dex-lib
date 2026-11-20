package lglclob

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestPoolTracker_GetNewPoolState(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip()
	}

	rpcURL := "https://rpc.soniclabs.com"
	multicallAddress := common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")
	poolAddress := "0xc7723fe3df538f76a063eb5e62867960d236accf"
	helperAddress := "0x38e577290CAf18d07b5719cc9da1E91Bd753f8c0"
	chainId := valueobject.ChainID(146)

	pt := &PoolTracker{
		config: &Config{
			DexID:         DexType,
			HelperAddress: helperAddress,
			ChainId:       chainId,
		},
		ethrpcClient: ethrpc.New(rpcURL).SetMulticallContract(multicallAddress),
	}
	// The reserves scale the book by the pool's scaling factors, so the pool needs them.
	p, err := pt.GetNewPoolState(context.Background(),
		entity.Pool{Address: poolAddress, StaticExtra: `{"sX":"1","sY":"1"}`},
		pool.GetNewPoolStateParams{})
	require.NoError(t, err)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	assert.NotZero(t, p.BlockNumber)
	assert.NotZero(t, extra.MeasuredAt, "a pool never measured is measured")
	assert.Empty(t, extra.Maker, "the pool has no market maker")
}

// TestPoolTracker_SharedMaker checks that pools quoted by one market maker are
// tracked with it, which is what lets them share a limit.
func TestPoolTracker_SharedMaker(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip()
	}

	pt := &PoolTracker{
		config: &Config{
			DexID:         DexType,
			HelperAddress: "0x237dB58fea34A35A8543b44C217d221606cE7788",
			ChainId:       valueobject.ChainIDMonad,
		},
		ethrpcClient: ethrpc.New("https://rpc.monad.xyz").
			SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")),
	}
	var makers []string
	// MON/WETH and cbBTC/WETH, whose bids both pay WETH.
	for _, poolAddress := range []string{"0xc4adcb94d0dce7fc84e180fae3621df13ad37e06",
		"0x05e028d33fa727168bc9396bdac5cab387c8a196"} {
		p, err := pt.GetNewPoolState(context.Background(),
			entity.Pool{Address: poolAddress, StaticExtra: `{"sX":"1","sY":"1"}`},
			pool.GetNewPoolStateParams{})
		require.NoError(t, err)

		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
		require.NotEmpty(t, extra.Maker)
		makers = append(makers, extra.Maker)
	}
	assert.Equal(t, makers[0], makers[1])
}

// TestPoolTracker_Timestamp checks that a refresh stamps the pool with the time
// it was read. Consumers treat a pool whose Timestamp has not moved as unchanged
// and keep quoting from the copy they already hold, so a pool without one keeps
// its first order book there forever.
func TestPoolTracker_Timestamp(t *testing.T) {
	t.Parallel()

	server := newLensServer(t, loadLensAnswer(t, "testdata/lens_base_book.hex"))
	pt := &PoolTracker{
		config:       &Config{DexID: DexType},
		ethrpcClient: ethrpc.New(server.URL),
	}
	// Measured a minute ago, so this refresh reads the book only.
	measuredAt := time.Now().Add(-time.Minute).Unix()
	extra, err := json.Marshal(Extra{MakerQuoteGas: &MakerQuoteGas{Bids: &LevelGas{First: 1, Next: 1},
		Asks: &LevelGas{First: 1, Next: 1}}, MeasuredAt: measuredAt})
	require.NoError(t, err)

	before := time.Now().Unix()
	p, err := pt.GetNewPoolState(context.Background(),
		entity.Pool{Address: "0xc7723fe3df538f76a063eb5e62867960d236accf", Extra: string(extra),
			StaticExtra: `{"sX":"1","sY":"1"}`},
		pool.GetNewPoolStateParams{})
	require.NoError(t, err)

	assert.GreaterOrEqual(t, p.Timestamp, before, "stamped when the book was read")
	assert.LessOrEqual(t, p.Timestamp, time.Now().Unix())
}

// newLensServer is an RPC node that answers Lens's deployless call by reverting
// with answer, and fails any other request.
func newLensServer(t *testing.T, answer []byte) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&req)) {
			return
		}
		var call struct {
			To *common.Address `json:"to"`
		}
		if req.Method != "eth_call" || len(req.Params) == 0 || json.Unmarshal(req.Params[0], &call) != nil ||
			call.To != nil {
			t.Errorf("unexpected request %s %s", req.Method, req.Params)
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID,
			"error": map[string]any{"code": 3, "message": "execution reverted", "data": hexutil.Encode(answer)}}))
	}))
	t.Cleanup(server.Close)
	return server
}
