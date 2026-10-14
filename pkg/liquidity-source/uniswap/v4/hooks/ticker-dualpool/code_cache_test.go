package tickerdualpool

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

// Deployed code is immutable, so a verified address is not downloaded again, but code
// supplied by a state override is always rechecked and never trusted from the cache.
func TestRequireCodeDownloadsVerifiedCodeOnce(t *testing.T) {
	verifiedCode.Clear() // package-global: stay repeatable under -count
	fetches := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage
			Method string
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, "eth_getCode", req.Method)
		fetches++
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x6001"})
	}))
	defer server.Close()
	address := common.HexToAddress("0xc0de0001")
	p := &uniswapv4.HookParam{RpcClient: ethrpc.New(server.URL)}
	matchGood := func(code []byte) bool { return len(code) == 2 }
	require.NoError(t, requireCode(context.Background(), p, address, "test", matchGood))
	require.NoError(t, requireCode(context.Background(), p, address, "test", matchGood))
	require.Equal(t, 1, fetches)

	// A failed match is never cached, and a verified label does not vouch for another.
	err := requireCode(context.Background(), p, address, "other", func([]byte) bool { return false })
	require.ErrorIs(t, err, ErrInvalidState)
	require.Equal(t, 2, fetches)

	p.Overrides = map[common.Address]gethclient.OverrideAccount{address: {Code: []byte{1, 2, 3}}}
	err = requireCode(context.Background(), p, address, "test", matchGood)
	require.ErrorIs(t, err, ErrInvalidState)
	require.Equal(t, 2, fetches)
}
