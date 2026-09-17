package deepstateob

import (
	"context"
	"net/http"
	"os"
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	// defaultRobinhoodRPCURL and multicallAddress match
	// pkg/liquidity-source/stonkbrokers-fun/v2's established Robinhood Chain
	// test setup -- ArbMulticall2, not Multicall3.
	defaultRobinhoodRPCURL = "https://rpc.mainnet.chain.robinhood.com"
	multicallAddress       = "0x2cAC2D899eCC914d704FeaAE33ac1bF36277DaD1"

	liveRouter = "0x6cf19308c22fc82ea620fa0b3e94948d20f27b96"
	liveLens   = "0x76f0257f524133cf41e8abcb694ac70ea3e8feca"
	liveUSDG   = "0x5fc5360d0400a0fd4f2af552add042d716f1d168" // token0, 6 decimals
	liveNVDA   = "0xd0601ce157db5bdc3162bbac2a2c8af5320d9eec" // token1, 18 decimals
)

func robinhoodRPCURL() string {
	if u := os.Getenv("DEEPSTATE_RPC_URL"); u != "" {
		return u
	}
	return defaultRobinhoodRPCURL
}

// countingTransport counts HTTP requests without altering them, so the round
// trip count reflects exactly what production traffic would send.
type countingTransport struct {
	inner http.RoundTripper
	count int64
}

func (c *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	atomic.AddInt64(&c.count, 1)
	return c.inner.RoundTrip(req)
}

// TestGetNewPoolState_ViaLens_IsOneRoundTrip is a live-network dex-verify
// step 3/4 test against the real deployed DeepstateBookLens
// (0x76f0257f524133cF41e8ABcb694Ac70EA3e8FEcA) and the real NVDA/USDG book on
// Robinhood Chain. It counts HTTP round trips at the transport layer to
// prove -- not assert -- the lens path is exactly one round trip per poll.
func TestGetNewPoolState_ViaLens_IsOneRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("live-network test, skipped under -short")
	}

	transport := &countingTransport{inner: http.DefaultTransport}
	httpClient := &http.Client{Transport: transport}

	rpcClient, err := gethrpc.DialOptions(context.Background(), robinhoodRPCURL(), gethrpc.WithHTTPClient(httpClient))
	require.NoError(t, err)

	client := ethrpc.NewWithClient(ethclient.NewClient(rpcClient)).
		SetMulticallContract(common.HexToAddress(multicallAddress))

	tracker, err := NewPoolTracker(&Config{DexID: DexType}, client)
	require.NoError(t, err)

	staticExtraBytes, err := json.Marshal(StaticExtra{Router: liveRouter, Lens: liveLens, Decimals: [2]uint8{6, 18}})
	require.NoError(t, err)

	pool := entity.Pool{
		Address:  "0xtest",
		Exchange: DexType,
		Type:     DexType,
		Reserves: entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: liveUSDG, Decimals: 6, Swappable: true},
			{Address: liveNVDA, Decimals: 18, Swappable: true},
		},
		Extra:       "{}",
		StaticExtra: string(staticExtraBytes),
	}

	got, err := tracker.GetNewPoolState(context.Background(), pool, poolpkg.GetNewPoolStateParams{})
	require.NoError(t, err)

	require.Equal(t, int64(1), atomic.LoadInt64(&transport.count), "lens path must be exactly one HTTP round trip")
	require.InDelta(t, 0.001, got.SwapFee, 1e-9, "swapFee should reflect the live feeBps=10")
	require.NotZero(t, got.BlockNumber)

	var extra struct {
		L [2][][2]float64 `json:"l"`
	}
	require.NoError(t, json.Unmarshal([]byte(got.Extra), &extra))
	require.Greater(t, len(extra.L[0]), 1, "expected resting bid levels beyond the dummy sentinel")
	require.Greater(t, len(extra.L[1]), 1, "expected resting ask levels beyond the dummy sentinel")
}
