package ephemeral

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeNode answers the gas probe the way one kind of node does, and records the gas of every other
// call.
type fakeNode struct {
	limit     uint64 // 0: no limit
	refuses   bool   // refuses a call above limit instead of capping it
	unreached bool   // fails every call, as a node that cannot be reached
	probes    atomic.Int32
	mu        sync.Mutex
	gas       []uint64
}

func (n *fakeNode) CallContract(_ context.Context, msg ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	if n.unreached {
		return nil, errors.New("dial tcp: connection refused")
	}
	if !bytes.Equal(msg.Data, probeCode) {
		n.mu.Lock()
		n.gas = append(n.gas, msg.Gas)
		n.mu.Unlock()
		return nil, errors.New("not a probe")
	}
	n.probes.Add(1)
	ran := msg.Gas
	if n.limit != 0 && ran > n.limit {
		if n.refuses {
			return nil, errors.New("gas limit too high")
		}
		ran = n.limit
	}
	out := make([]byte, 32)
	new(big.Int).SetUint64(ran - probeIntrinsic).FillBytes(out)
	return out, nil
}

// ctxNode is a fakeNode that fails a call whose context is done, as a client does.
type ctxNode struct{ fakeNode }

func (n *ctxNode) CallContract(ctx context.Context, msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return n.fakeNode.CallContract(ctx, msg, block)
}

// TestGasCapProbeCutShortIsNotKept: a probe its caller's context cut short says nothing of the
// node, and the next call probes again rather than going without the cap for probeRetry.
func TestGasCapProbeCutShortIsNotKept(t *testing.T) {
	node := &ctxNode{limit: 50_000_000}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := GasCap(cancelled, node)
	require.ErrorIs(t, err, context.Canceled)

	got, err := GasCap(t.Context(), node)
	require.NoError(t, err)
	assert.Equal(t, Cap{Gas: 50_000_000}, got)
	assert.EqualValues(t, 50_000_000, callGas(t.Context(), node, 0))
}

// TestGasCapNodeTimeoutIsKept: a node that answers no probe within the HTTP client's timeout, which
// the client reports as a deadline, failed the probe as a node, and the failure is kept for
// probeRetry rather than probed again for as long as the caller's context lives.
func TestGasCapNodeTimeoutIsKept(t *testing.T) {
	var requests atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		requests.Add(1)
		time.Sleep(300 * time.Millisecond)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x"}`))
	}))
	defer srv.Close()
	rc, err := rpc.DialOptions(t.Context(), srv.URL,
		rpc.WithHTTPClient(&http.Client{Timeout: 50 * time.Millisecond}))
	require.NoError(t, err)
	node := ethclient.NewClient(rc)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	_, err = GasCap(ctx, node)
	require.Error(t, err)
	require.NoError(t, ctx.Err(), "the probe fails on the node's timeouts, well before the caller's")
	assert.LessOrEqual(t, requests.Load(), int64(len(probeLadder)), "one ladder")

	asked := requests.Load()
	_, err = GasCap(t.Context(), node)
	require.Error(t, err)
	assert.Equal(t, asked, requests.Load(), "the failure is kept")
}

// TestGasCapPerKindOfNode: a node that caps reports its cap, one that refuses a call above its
// limit reports the highest ask it ran, one without a limit says so, and each is probed once.
func TestGasCapPerKindOfNode(t *testing.T) {
	ctx := t.Context()
	for name, tc := range map[string]struct {
		node *fakeNode
		want Cap
		gas  uint64
	}{
		"caps at 50M (geth's default)":    {&fakeNode{limit: 50_000_000}, Cap{Gas: 50_000_000}, 50_000_000},
		"caps at 550M (Alchemy)":          {&fakeNode{limit: 550_000_000}, Cap{Gas: 550_000_000}, CallGasCeiling},
		"refuses above 150M (Monad)":      {&fakeNode{limit: 150_000_000, refuses: true}, Cap{Gas: 150_000_000}, CallGasCeiling},
		"no limit (MegaETH, Linea)":       {&fakeNode{}, Cap{Unlimited: true}, CallGasCeiling},
		"caps at 30M (HyperEVM)":          {&fakeNode{limit: 30_000_000}, Cap{Gas: 30_000_000}, 30_000_000},
		"refuses above 10M, caps nowhere": {&fakeNode{limit: 10_000_000, refuses: true}, Cap{Gas: 10_000_000}, 10_000_000},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := GasCap(ctx, tc.node)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)

			_, _ = GasCap(ctx, tc.node)
			probes := tc.node.probes.Load()
			assert.Equal(t, callGas(ctx, tc.node, 0), tc.gas, "the gas a call is sent")
			assert.Equal(t, probes, tc.node.probes.Load(), "probed once per client")
		})
	}
}

// TestCallSendsTheProbedGas: a call left to the node's gas is sent the node's cap up to the
// ceiling, a call given gas is sent it as given, and a node that cannot be probed is left to its
// default.
func TestCallSendsTheProbedGas(t *testing.T) {
	ctx := t.Context()
	monad := &fakeNode{limit: 150_000_000, refuses: true}
	_ = Call(ctx, monad, testContract, new(any), 0, nil, big.NewInt(1))
	_ = Call(ctx, monad, testContract, new(any), 20_000_000, nil, big.NewInt(1))
	assert.Equal(t, []uint64{CallGasCeiling, 20_000_000}, monad.gas)

	down := &fakeNode{unreached: true}
	gasCap, err := GasCap(ctx, down)
	assert.Error(t, err)
	assert.Equal(t, Cap{}, gasCap)
	assert.Zero(t, callGas(ctx, down, 0))
}
