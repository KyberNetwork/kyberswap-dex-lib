package erc7575

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
)

// The share token is not the vault, so discovery must not run the maxRedeem balance probe: it would write
// bK/nbK into StaticExtra. A probe shows up as a stateDiff override on the vault.
func TestGetNewPoolNoProbe(t *testing.T) {
	t.Parallel()
	var calls, probed atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		calls.Add(1)
		if strings.Contains(string(body), "stateDiff") {
			probed.Add(1)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"stop"}}`))
	}))
	defer srv.Close()

	u := NewPoolsListUpdater(&Config{DexId: "test"}, ethrpc.New(srv.URL))
	const vault = "0x00000000000000000000000000000000000000aa"
	cfg := erc4626.VaultCfg{Gas: erc4626.GasCfg{Deposit: 1, Redeem: 1}}
	_, err := u.getNewPool(context.Background(), vault, cfg)

	assert.Error(t, err)
	assert.Positive(t, calls.Load())
	assert.Zero(t, probed.Load(), "discovery must not send probe overrides")
}
