package ephemeral

import (
	"bytes"
	"context"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// slowNode answers every lens call with a result after a while, and counts the calls running at
// once; it answers a gas cap probe at once.
type slowNode struct {
	took  time.Duration
	calls atomic.Int32

	mu            sync.Mutex
	running, most int
}

func (n *slowNode) CallContract(_ context.Context, msg ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	if bytes.Equal(msg.Data, probeCode) {
		out := make([]byte, 32)
		new(big.Int).SetUint64(msg.Gas - probeIntrinsic).FillBytes(out)
		return out, nil
	}
	n.calls.Add(1)
	n.mu.Lock()
	n.running++
	n.most = max(n.most, n.running)
	n.mu.Unlock()
	time.Sleep(n.took)
	n.mu.Lock()
	n.running--
	n.mu.Unlock()
	return nil, dataError{result(7, -5)}
}

func callMany(t *testing.T, c Caller, n int) {
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			var got testResult
			assert.NoError(t, Call(t.Context(), c, testContract, &got, 1_000_000, big.NewInt(1), big.NewInt(1)))
		})
	}
	wg.Wait()
}

// TestLimitNodeCapsTheCallsAtOnce: past the cap, a call waits for one to end.
func TestLimitNodeCapsTheCallsAtOnce(t *testing.T) {
	node := &slowNode{took: 20 * time.Millisecond}
	LimitNode(node, NodeLimit{Concurrency: 2})
	t.Cleanup(func() { LimitNode(node, NodeLimit{}) })

	callMany(t, node, 10)
	assert.EqualValues(t, 10, node.calls.Load())
	assert.Equal(t, 2, node.most)
}

// TestLimitNodeKeepsToTheRate: past the burst, a call waits for the rate.
func TestLimitNodeKeepsToTheRate(t *testing.T) {
	node := &slowNode{}
	LimitNode(node, NodeLimit{Rate: 20, Burst: 1})
	t.Cleanup(func() { LimitNode(node, NodeLimit{}) })

	start := time.Now()
	callMany(t, node, 5)
	assert.GreaterOrEqual(t, time.Since(start), 200*time.Millisecond, "4 waits of 50ms")
}

// TestLimitNodeIsPerNode: a node without a limit, or whose limit was removed, is not held.
func TestLimitNodeIsPerNode(t *testing.T) {
	limited, other := &slowNode{took: 20 * time.Millisecond}, &slowNode{took: 20 * time.Millisecond}
	LimitNode(limited, NodeLimit{Concurrency: 1})
	callMany(t, other, 5)
	assert.Equal(t, 5, other.most)

	LimitNode(limited, NodeLimit{})
	callMany(t, limited, 5)
	assert.Equal(t, 5, limited.most)
}

// TestLimitNodeWaitsOnTheContext: a call waiting for its turn ends with its context.
func TestLimitNodeWaitsOnTheContext(t *testing.T) {
	node := &slowNode{}
	LimitNode(node, NodeLimit{Rate: 0.001, Burst: 1})
	t.Cleanup(func() { LimitNode(node, NodeLimit{}) })

	var got testResult
	require.NoError(t, Call(t.Context(), node, testContract, &got, 1_000_000, big.NewInt(1), big.NewInt(1)))
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	err := Call(ctx, node, testContract, &got, 1_000_000, big.NewInt(1), big.NewInt(1))
	require.Error(t, err)
	assert.EqualValues(t, 1, node.calls.Load())
}

// TestLimitNodeLeavesFuncCallers: a Caller that cannot be a map key is never limited, nor panics.
func TestLimitNodeLeavesFuncCallers(t *testing.T) {
	c := reverting(result(7, -5))
	LimitNode(c, NodeLimit{Concurrency: 1})
	var got testResult
	require.NoError(t, Call(t.Context(), c, testContract, &got, 0, nil, big.NewInt(1)))
}
