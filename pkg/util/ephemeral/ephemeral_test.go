package ephemeral

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dataError is how go-ethereum's rpc client reports an error carrying data, a revert's included.
type dataError struct{ data any }

func (e dataError) Error() string  { return "execution reverted" }
func (e dataError) ErrorData() any { return e.data }

// failing is a JSON-RPC error without data, as go-ethereum reports any error the node returns.
type failing struct{ msg string }

func (e failing) Error() string  { return e.msg }
func (e failing) ErrorData() any { return nil }

type callerFunc func(msg ethereum.CallMsg, block *big.Int) ([]byte, error)

func (f callerFunc) CallContract(_ context.Context, msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
	return f(msg, block)
}

func reverting(data any) Caller {
	return callerFunc(func(ethereum.CallMsg, *big.Int) ([]byte, error) { return nil, dataError{data} })
}

// testContract has a constructor of one uint256, a result of a uint256 and an int24, and one error.
var testContract = MustLoad("Test", []byte(`{
  "bytecode": "0xfe",
  "abi": [
    {"type": "constructor", "inputs": [{"name": "x", "type": "uint256"}]},
    {"type": "function", "name": "get", "stateMutability": "view", "inputs": [],
     "outputs": [{"name": "value", "type": "uint256"}, {"name": "tick", "type": "int24"}]},
    {"type": "error", "name": "Known", "inputs": []}
  ]
}`))

type testResult struct {
	Value *big.Int
	Tick  *big.Int
}

// result is the payload of a get result, the tick sign-extended as the ABI wants it.
func result(value uint64, tick int64) string {
	return hexutil.Encode(append(common32(value), int32Word(tick)...))
}

func int32Word(v int64) []byte {
	b := make([]byte, 32)
	new(big.Int).And(big.NewInt(v), new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))).FillBytes(b)
	return b
}

func TestCall(t *testing.T) {
	ctx := t.Context()
	payload := result(7, -5)

	var got testResult
	require.NoError(t, Call(ctx, reverting(payload), testContract, &got, 0, nil, big.NewInt(1)))
	assert.EqualValues(t, 7, got.Value.Int64())
	assert.EqualValues(t, -5, got.Tick.Int64())

	// Nodes that wrap the revert data in an object.
	for _, wrapped := range []map[string]any{{"data": payload}, {"result": payload}} {
		got = testResult{}
		require.NoError(t, Call(ctx, reverting(wrapped), testContract, &got, 0, nil, big.NewInt(1)))
		assert.EqualValues(t, 7, got.Value.Int64())
	}

	known := hexutil.Encode(testContract.abi.Errors["Known"].ID.Bytes()[:4])
	err := Call(ctx, reverting(known), testContract, &got, 0, nil, big.NewInt(1))
	assert.True(t, IsContractError(err, "Known"), "%v", err)

	// Error(string) "nope", passed up from a contract the lens called.
	reason := "0x08c379a0" +
		"0000000000000000000000000000000000000000000000000000000000000020" +
		"0000000000000000000000000000000000000000000000000000000000000004" +
		"6e6f706500000000000000000000000000000000000000000000000000000000"
	var revert *RevertError
	err = Call(ctx, reverting(reason), testContract, &got, 0, nil, big.NewInt(1))
	require.ErrorAs(t, err, &revert)
	assert.Equal(t, "nope", revert.Err)

	assert.ErrorIs(t, Call(ctx, reverting("0x"), testContract, &got, 0, nil, big.NewInt(1)), ErrNoRevertData)
	assert.ErrorIs(t, Call(ctx, reverting(nil), testContract, &got, 0, nil, big.NewInt(1)), ErrNoRevertData,
		"a bare execution reverted")

	// Every JSON-RPC error carries ErrorData, those of calls that never ran too: an unknown block
	// is the node's answer, not a revert, and must not read as one.
	unknown := failing{msg: "header not found"}
	err = Call(ctx, callerFunc(func(ethereum.CallMsg, *big.Int) ([]byte, error) { return nil, unknown }),
		testContract, &got, 0, nil, big.NewInt(1))
	assert.ErrorIs(t, err, unknown)
	assert.NotErrorIs(t, err, ErrNoRevertData)

	err = Call(ctx, callerFunc(func(ethereum.CallMsg, *big.Int) ([]byte, error) { return nil, nil }),
		testContract, &got, 0, nil, big.NewInt(1))
	assert.ErrorIs(t, err, ErrNotReverted)

	assert.Error(t, Call(ctx, reverting(payload), testContract, &got, 0, nil, "not a uint256"),
		"arguments that do not fit the constructor")
}

// TestLoadNeedsOneResult: the result is the ABI's one function, so a file with two is refused
// rather than decoded with whichever comes first.
func TestLoadNeedsOneResult(t *testing.T) {
	_, err := Load("Two", []byte(`{"bytecode": "0xfe", "abi": [
		{"type": "function", "name": "a", "inputs": [], "outputs": []},
		{"type": "function", "name": "b", "inputs": [], "outputs": []}]}`))
	assert.Error(t, err)
	_, err = Load("Empty", []byte(`{"bytecode": "0x", "abi": []}`))
	assert.Error(t, err)
}

// TestPackCall: the call is the creation code followed by the constructor's arguments.
func TestPackCall(t *testing.T) {
	data, err := testContract.PackCall(big.NewInt(7))
	require.NoError(t, err)
	assert.Equal(t, append([]byte{0xfe}, common32(7)...), data)
}

// TestUnpackCallTakesCanonicalResultsOnly: each payload here is a result with one thing wrong;
// reading any of them as a result would hand a tracker made-up state.
func TestUnpackCallTakesCanonicalResultsOnly(t *testing.T) {
	good := hexutil.MustDecode(result(7, -5))
	var got testResult
	require.NoError(t, testContract.UnpackCall(good, &got))

	tickWord := func(hexWord string) []byte {
		return append(common32(7), hexutil.MustDecode("0x"+strings.Repeat("0", 64-len(hexWord))+hexWord)...)
	}
	for name, payload := range map[string][]byte{
		"empty":            nil,
		"one word":         good[:32],
		"trailing word":    append(bytes.Clone(good), make([]byte, 32)...),
		"not whole words":  append(bytes.Clone(good), 0),
		"tick past int24":  tickWord("800000"),
		"tick high bits":   tickWord("1000000000000000000000000000000000000000000000000000000000000005"),
		"negative unclean": tickWord("00fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffb"),
	} {
		assert.Error(t, testContract.UnpackCall(payload, &got), name)
	}
}

// TestScanPinsTheFirstBlock: pages after the first read the block the first one read, and a page
// that does not move is an error rather than a loop.
func TestScanPinsTheFirstBlock(t *testing.T) {
	var blocks []*big.Int
	block, err := Scan(context.Background(), 0, nil,
		func(_ context.Context, cursor int, block *big.Int) (Page[int], error) {
			blocks = append(blocks, block)
			return Page[int]{Block: 100, Next: cursor + 1, Done: cursor == 2}, nil
		})
	require.NoError(t, err)
	assert.EqualValues(t, 100, block)
	require.Len(t, blocks, 3)
	assert.Nil(t, blocks[0])
	assert.EqualValues(t, 100, blocks[1].Uint64())

	_, err = Scan(context.Background(), 0, nil,
		func(_ context.Context, cursor int, _ *big.Int) (Page[int], error) {
			return Page[int]{Block: 100, Next: cursor}, nil
		})
	assert.ErrorIs(t, err, ErrNoProgress)

	// A chain that runs a call in the block after the one it is pinned to reports that block; the
	// read is still the pinned block's state. Any other block is a node that ignored the pin.
	block, err = Scan(context.Background(), 0, big.NewInt(100),
		func(_ context.Context, cursor int, _ *big.Int) (Page[int], error) {
			return Page[int]{Block: 101, Next: cursor + 1, Done: cursor == 1}, nil
		})
	require.NoError(t, err)
	assert.EqualValues(t, 100, block)
	_, err = Scan(context.Background(), 0, big.NewInt(100),
		func(_ context.Context, cursor int, _ *big.Int) (Page[int], error) {
			return Page[int]{Block: 107, Done: true}, nil
		})
	assert.ErrorIs(t, err, ErrUnpinned)
}

type head uint64

func (h head) BlockNumber(context.Context) (uint64, error) { return uint64(h), nil }

// TestPinNamesTheBlockFirst: a read with no block is pinned to the head before its first call, and
// one with a block keeps it.
func TestPinNamesTheBlockFirst(t *testing.T) {
	block, err := Pin(context.Background(), head(500), nil)
	require.NoError(t, err)
	assert.EqualValues(t, 500, block.Uint64())

	block, err = Pin(context.Background(), head(500), big.NewInt(7))
	require.NoError(t, err)
	assert.EqualValues(t, 7, block.Uint64())
}

func TestCheckTicks(t *testing.T) {
	tick := func(i int, net int64) Tick { return Tick{Index: i, LiquidityNet: big.NewInt(net)} }
	// Two positions: [-10, 10) with 5, [0, 20) with 3. At tick 5 both are in range.
	ticks := []Tick{tick(-10, 5), tick(0, 3), tick(10, -5), tick(20, -3)}
	assert.NoError(t, CheckTicks(ticks, 5, big.NewInt(8)))

	// One end missing: the sum over all ticks is off.
	assert.ErrorIs(t, CheckTicks(ticks[:3], 5, big.NewInt(8)), ErrTicksInconsistent)
	// Both ends of the straddling [0, 20) missing: only the in-range sum notices.
	assert.ErrorIs(t, CheckTicks([]Tick{tick(-10, 5), tick(10, -5)}, 5, big.NewInt(8)), ErrTicksInconsistent)
}

func common32(v uint64) []byte {
	b := make([]byte, 32)
	new(big.Int).SetUint64(v).FillBytes(b)
	return b
}

// TestSplitHalvesAndPinsTheBlock: a part too large for one call is halved until it fits, every
// part after the first reads the first one's block, and the results keep the items' order.
func TestSplitHalvesAndPinsTheBlock(t *testing.T) {
	tooLarge := errors.New("too large")
	var blocks []*big.Int
	block, got, err := Split(context.Background(), []int{1, 2, 3, 4, 5}, nil,
		func(err error) bool { return errors.Is(err, tooLarge) },
		func(_ context.Context, part []int, block *big.Int) (uint64, []int, error) {
			if len(part) > 2 {
				return 0, nil, tooLarge
			}
			blocks = append(blocks, block)
			return 100, part, nil
		})
	require.NoError(t, err)
	assert.EqualValues(t, 100, block)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, got)
	require.Len(t, blocks, 3, "split into [1 2] [3] [4 5]")
	assert.Nil(t, blocks[0])
	assert.EqualValues(t, 100, blocks[1].Uint64())
	assert.EqualValues(t, 100, blocks[2].Uint64())

	// One item still too large, and an error that is not about size, end the read.
	_, _, err = Split(context.Background(), []int{1}, nil,
		func(err error) bool { return errors.Is(err, tooLarge) },
		func(context.Context, []int, *big.Int) (uint64, []int, error) { return 0, nil, tooLarge })
	assert.ErrorIs(t, err, tooLarge)
	calls := 0
	limited := errors.New("429 too many requests")
	_, _, err = Split(context.Background(), []int{1, 2, 3, 4}, nil,
		func(err error) bool { return errors.Is(err, tooLarge) },
		func(context.Context, []int, *big.Int) (uint64, []int, error) { calls++; return 0, nil, limited })
	assert.ErrorIs(t, err, limited)
	assert.Equal(t, 1, calls, "a rate limit is not answered with more calls")

	// A later part on another block would mix two states.
	_, _, err = Split(context.Background(), []int{1, 2}, nil,
		func(error) bool { return true },
		func(_ context.Context, part []int, _ *big.Int) (uint64, []int, error) {
			if len(part) > 1 {
				return 0, nil, tooLarge
			}
			return uint64(100 + 5*part[0]), part, nil
		})
	assert.ErrorIs(t, err, ErrUnpinned)
}

// TestGuardNeverWaits: past its burst, its concurrency or a key's cooldown, the guard declines at
// once instead of making the update wait.
func TestGuardNeverWaits(t *testing.T) {
	ctx := t.Context()
	ok := func(context.Context) error { return nil }

	g := NewGuard(GuardConfig{Rate: 0.001, Burst: 2, Concurrency: 2})
	for i, want := range []bool{true, true, false} {
		ran, err := g.Try(ctx, fmt.Sprint("pool", i), ok)
		require.NoError(t, err)
		assert.Equal(t, want, ran, "call %d", i)
	}

	g = NewGuard(GuardConfig{Burst: 10, CooldownSecs: 60})
	ran, _ := g.Try(ctx, "pool", ok)
	assert.True(t, ran)
	ran, _ = g.Try(ctx, "pool", ok)
	assert.False(t, ran, "the same pool again within its cooldown")
	ran, _ = g.Try(ctx, "other", ok)
	assert.True(t, ran)

	g = NewGuard(GuardConfig{Burst: 10, Concurrency: 1})
	inside, release := make(chan struct{}), make(chan struct{})
	go func() {
		_, _ = g.Try(ctx, "slow", func(context.Context) error { close(inside); <-release; return nil })
	}()
	<-inside
	ran, _ = g.Try(ctx, "fast", ok)
	assert.False(t, ran, "a call while the only slot is taken")
	close(release)

	g = NewGuard(GuardConfig{TimeoutSecs: 1})
	ran, err := g.Try(ctx, "pool", func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() })
	assert.True(t, ran)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
