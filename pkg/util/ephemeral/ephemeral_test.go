package ephemeral

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// dataError is how go-ethereum's rpc client reports an error carrying data, a revert's included.
type dataError struct{ data any }

func (e dataError) Error() string  { return "execution reverted" }
func (e dataError) ErrorData() any { return e.data }

// failing is a JSON-RPC error without data, as go-ethereum reports any error the node returns.
type failing struct{ msg string }

func (e failing) Error() string  { return e.msg }
func (e failing) ErrorData() any { return nil }

// jsonError is a JSON-RPC error with its code, as go-ethereum reports one.
type jsonError struct {
	code int
	msg  string
	data any
}

func (e jsonError) Error() string  { return e.msg }
func (e jsonError) ErrorCode() int { return e.code }
func (e jsonError) ErrorData() any { return e.data }

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

// TestCallAsksASmallerPageOfARefusedResponse: a lens pages by gas, and a node or a proxy in front
// of it refuses a large response, each in its own way. A call so refused is made again with half
// the gas, for a smaller page, down to the least gas a lens makes progress with.
func TestCallAsksASmallerPageOfARefusedResponse(t *testing.T) {
	ctx := t.Context()
	var sent []uint64
	var refuseAbove uint64
	var refusal error
	node := callerFunc(func(msg ethereum.CallMsg, _ *big.Int) ([]byte, error) {
		if bytes.Equal(msg.Data, probeCode) {
			return nil, failing{msg: "no probe"}
		}
		sent = append(sent, msg.Gas)
		// Gas 0 runs with the node's own default, here as much as any.
		if msg.Gas == 0 || msg.Gas > refuseAbove {
			return nil, refusal
		}
		return nil, dataError{result(7, -5)}
	})
	call := func(gas uint64) error {
		sent = nil
		var got testResult
		err := Call(ctx, node, testContract, &got, gas, nil, big.NewInt(1))
		if err == nil {
			assert.EqualValues(t, 7, got.Value.Int64())
		}
		return err
	}

	refuseAbove = 30_000_000
	for _, refusal = range []error{
		// A proxy, for 2 MB of revert data.
		rpc.HTTPError{StatusCode: http.StatusBadGateway, Status: "502 Bad Gateway"},
		// Another proxy, for 1.5 MB.
		rpc.HTTPError{StatusCode: http.StatusUnauthorized, Status: "401 Unauthorized",
			Body: []byte(`{"error":"UNAUTHORIZED"}`)},
		// A body past 1 MiB on HyperEVM, its data no revert data.
		jsonError{code: -32008, msg: "Response is too big", data: "Exceeded max limit of 1048576"},
		rpc.HTTPError{StatusCode: http.StatusRequestEntityTooLarge, Status: "413 Payload Too Large"},
	} {
		require.NoError(t, call(50_000_000), "%v", refusal)
		assert.Equal(t, []uint64{50_000_000, 25_000_000}, sent, "%v", refusal)
	}

	// Refused however small: the error, once half the gas is less than a lens makes progress with.
	refusal = rpc.HTTPError{StatusCode: http.StatusBadGateway, Status: "502 Bad Gateway"}
	refuseAbove = 0
	var refused rpc.HTTPError
	require.ErrorAs(t, call(50_000_000), &refused)
	assert.Equal(t, []uint64{50_000_000, 25_000_000, 12_500_000, 6_250_000}, sent)

	// The gas left to a node whose cap is unknown: there is none to halve.
	require.ErrorAs(t, call(0), &refused)
	assert.Equal(t, []uint64{0}, sent)

	// Any other refusal says nothing of the size.
	for _, refusal = range []error{
		rpc.HTTPError{StatusCode: http.StatusTooManyRequests, Status: "429 Too Many Requests"},
		jsonError{code: -32000, msg: "header not found"},
	} {
		assert.EqualError(t, call(50_000_000), refusal.Error())
		assert.Equal(t, []uint64{50_000_000}, sent, "%v", refusal)
	}
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
	block, err := Scan(t.Context(), 0, Pinned{},
		func(_ context.Context, cursor int, block *big.Int) (Page[int], error) {
			blocks = append(blocks, block)
			return Page[int]{Block: 100, Next: cursor + 1, Done: cursor == 2}, nil
		})
	require.NoError(t, err)
	assert.EqualValues(t, 100, block)
	require.Len(t, blocks, 3)
	assert.Nil(t, blocks[0])
	assert.EqualValues(t, 100, blocks[1].Uint64())

	_, err = Scan(t.Context(), 0, Pinned{},
		func(_ context.Context, cursor int, _ *big.Int) (Page[int], error) {
			return Page[int]{Block: 100, Next: cursor}, nil
		})
	assert.ErrorIs(t, err, ErrNoProgress)

	// A chain that runs a call in the block after the one it is pinned to reports that block; the
	// read is still the pinned block's state. Any other block is a node that ignored the pin.
	next := func(_ context.Context, cursor int, _ *big.Int) (Page[int], error) {
		return Page[int]{Block: 101, Next: cursor + 1, Done: cursor == 1}, nil
	}
	block, err = Scan(t.Context(), 0, Pinned{Block: big.NewInt(100), late: true}, next)
	require.NoError(t, err)
	assert.EqualValues(t, 100, block)
	_, err = Scan(t.Context(), 0, Pinned{Block: big.NewInt(100), late: true},
		func(_ context.Context, cursor int, _ *big.Int) (Page[int], error) {
			return Page[int]{Block: 107, Done: true}, nil
		})
	assert.ErrorIs(t, err, ErrUnpinned)

	// Elsewhere the block after is a head that moved on between Pin and the call: its state is not
	// the pinned block's, and taken for it, the logs of the block after would be applied twice.
	_, err = Scan(t.Context(), 0, Pinned{Block: big.NewInt(100)}, next)
	assert.ErrorIs(t, err, ErrUnpinned)
}

// chain is a node of chain id, counting the times it is asked it.
type chain struct {
	id    uint64
	asked *int
}

func (c chain) ChainID(context.Context) (*big.Int, error) {
	if c.asked != nil {
		*c.asked++
	}
	return new(big.Int).SetUint64(c.id), nil
}

// scanTwoPages scans two pages from pin on a node that reports every call at reported, and returns
// the block read and each page's.
func scanTwoPages(t *testing.T, pin Pinned, reported uint64) (uint64, []*big.Int) {
	var blocks []*big.Int
	read, err := Scan(t.Context(), 0, pin,
		func(_ context.Context, cursor int, block *big.Int) (Page[int], error) {
			blocks = append(blocks, block)
			return Page[int]{Block: reported, Next: cursor + 1, Done: cursor == 1}, nil
		})
	require.NoError(t, err)
	return read, blocks
}

// TestPinAsksNoHead: a read with no block runs its first call at the latest and takes the block
// that call reports, asking the node for no head; one with a block keeps it.
func TestPinAsksNoHead(t *testing.T) {
	pin, err := Pin(t.Context(), chain{id: 1}, nil)
	require.NoError(t, err)
	assert.Nil(t, pin.Block)

	read, blocks := scanTwoPages(t, pin, 500)
	assert.EqualValues(t, 500, read)
	require.Len(t, blocks, 2)
	assert.Nil(t, blocks[0], "the first call runs at the latest")
	assert.EqualValues(t, 500, blocks[1].Uint64(), "the next is pinned to the block the first ran at")

	pin, err = Pin(t.Context(), chain{id: 1}, big.NewInt(7))
	require.NoError(t, err)
	assert.EqualValues(t, 7, pin.Block.Uint64())
}

// TestPinTakesTheNextBlockOnlyOnEtherlink: only on a chain whose node runs a call in the block
// after the one whose state it reads is a report of N+1 the state of N, at the latest and pinned
// alike, and a client's chain is asked once.
func TestPinTakesTheNextBlockOnlyOnEtherlink(t *testing.T) {
	asked := 0
	etherlink := chain{id: uint64(valueobject.ChainIDEtherlink), asked: &asked}

	// At the latest: the node at 500 runs the call in 501 and reports it; the read is 500's, and
	// its next call is pinned to 500, which the node reports as 501 again.
	pin, err := Pin(t.Context(), etherlink, nil)
	require.NoError(t, err)
	read, blocks := scanTwoPages(t, pin, 501)
	assert.EqualValues(t, 500, read)
	assert.EqualValues(t, 500, blocks[1].Uint64())

	pinnedNext := func(c chain) error {
		pin, err := Pin(context.Background(), c, big.NewInt(500))
		require.NoError(t, err)
		_, err = Scan(context.Background(), 0, pin,
			func(_ context.Context, _ int, block *big.Int) (Page[int], error) {
				return Page[int]{Block: block.Uint64() + 1, Done: true}, nil
			})
		return err
	}
	assert.NoError(t, pinnedNext(etherlink))
	assert.Equal(t, 1, asked, "the chain id is asked once per client")
	assert.ErrorIs(t, pinnedNext(chain{id: uint64(valueobject.ChainIDHyperEVM)}), ErrUnpinned)
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
	block, got, err := Split(t.Context(), []int{1, 2, 3, 4, 5}, Pinned{},
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
	_, _, err = Split(t.Context(), []int{1}, Pinned{},
		func(err error) bool { return errors.Is(err, tooLarge) },
		func(context.Context, []int, *big.Int) (uint64, []int, error) { return 0, nil, tooLarge })
	assert.ErrorIs(t, err, tooLarge)
	calls := 0
	limited := errors.New("429 too many requests")
	_, _, err = Split(t.Context(), []int{1, 2, 3, 4}, Pinned{},
		func(err error) bool { return errors.Is(err, tooLarge) },
		func(context.Context, []int, *big.Int) (uint64, []int, error) { calls++; return 0, nil, limited })
	assert.ErrorIs(t, err, limited)
	assert.Equal(t, 1, calls, "a rate limit is not answered with more calls")

	// A later part on another block would mix two states.
	_, _, err = Split(t.Context(), []int{1, 2}, Pinned{},
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

func TestCheckBins(t *testing.T) {
	bin := func(id uint32, x, y int64) Bin { return Bin{ID: id, ReserveX: big.NewInt(x), ReserveY: big.NewInt(y)} }
	bins := []Bin{bin(10, 0, 5), bin(11, 3, 2), bin(12, 4, 0)}

	assert.NoError(t, CheckBins(bins, big.NewInt(7), big.NewInt(7)))
	assert.NoError(t, CheckBins(nil, big.NewInt(0), big.NewInt(0)), "a pair without liquidity")
	assert.NoError(t, CheckBins(bins, nil, nil), "a pool that keeps no reserves")
	for name, c := range map[string]struct {
		bins []Bin
		x, y int64
	}{
		"a bin missing":   {bins[:2], 7, 7},
		"reserve Y off":   {bins, 7, 8},
		"out of order":    {[]Bin{bins[1], bins[0], bins[2]}, 7, 7},
		"a bin repeated":  {[]Bin{bins[0], bins[0], bins[1], bins[2]}, 7, 12},
		"an empty bin":    {append([]Bin{bin(9, 0, 0)}, bins...), 7, 7},
		"reserves absent": {bins, 0, 0},
	} {
		assert.ErrorIs(t, CheckBins(c.bins, big.NewInt(c.x), big.NewInt(c.y)), ErrBinsInconsistent, name)
	}
	for name, bins := range map[string][]Bin{
		"out of order": {bin(5, 1, 0), bin(1, 0, 1)},
		"repeated":     {bin(5, 1, 0), bin(5, 0, 1)},
		"empty":        {bin(5, 0, 0)},
	} {
		assert.ErrorIs(t, CheckBins(bins, nil, nil), ErrBinsInconsistent, "no reserves, "+name)
	}
}

// TestScanWordsReadsWindowsInParallel: the words a call leaves are split and read by calls at once,
// at the first call's block, and the items come back in word order as one scan would give them.
func TestScanWordsReadsWindowsInParallel(t *testing.T) {
	const perCall = 3 // words one call reads
	var inFlight, most atomic.Int32
	var blocks sync.Map
	fetch := func(_ context.Context, lower, upper int16, block *big.Int) (Page[int16], []int16, error) {
		n := inFlight.Add(1)
		defer inFlight.Add(-1)
		for m := most.Load(); n > m && !most.CompareAndSwap(m, n); m = most.Load() {
		}
		if block != nil {
			blocks.Store(block.Uint64(), true)
		}
		// Long enough for the windows' calls to overlap, as a node's round trips do.
		time.Sleep(5 * time.Millisecond)
		var words []int16
		w := lower
		for ; w <= upper && len(words) < perCall; w++ {
			words = append(words, w)
		}
		return Page[int16]{Block: 100, Next: w, Done: w > upper}, words, nil
	}

	block, words, err := ScanWords(t.Context(), -10, 30, Pinned{}, fetch)
	require.NoError(t, err)
	assert.EqualValues(t, 100, block)
	require.Len(t, words, 41)
	for i, w := range words {
		assert.EqualValues(t, -10+i, w, "word order")
	}
	assert.Greater(t, most.Load(), int32(1), "windows read at once")
	assert.LessOrEqual(t, most.Load(), int32(WordsParallel))
	blocks.Range(func(k, _ any) bool {
		assert.EqualValues(t, 100, k, "every window at the first call's block")
		return true
	})

	_, _, err = ScanWords(t.Context(), 0, 5, Pinned{},
		func(context.Context, int16, int16, *big.Int) (Page[int16], []int16, error) {
			return Page[int16]{Block: 100, Next: 0}, nil, nil
		})
	assert.ErrorIs(t, err, ErrNoProgress)

	_, _, err = ScanWords(t.Context(), 0, 5, Pinned{Block: big.NewInt(100)},
		func(context.Context, int16, int16, *big.Int) (Page[int16], []int16, error) {
			return Page[int16]{Block: 107, Done: true}, nil, nil
		})
	assert.ErrorIs(t, err, ErrUnpinned)
}

// countingChain is a Chain that counts the chain ids it is asked.
type countingChain struct {
	asked atomic.Int64
}

func (h *countingChain) ChainID(context.Context) (*big.Int, error) {
	h.asked.Add(1)
	time.Sleep(20 * time.Millisecond)
	return big.NewInt(1), nil
}

// TestPinAsksTheChainOnce: reads that start together share one chain id question.
func TestPinAsksTheChainOnce(t *testing.T) {
	h := &countingChain{}
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			_, err := Pin(context.Background(), h, big.NewInt(7))
			assert.NoError(t, err)
		})
	}
	wg.Wait()
	assert.EqualValues(t, 1, h.asked.Load())
}

// refusingNode serves V3Lens one word a call at block 100, and refuses, as a proxy does for its
// size, a call sent more gas than refuseAbove. It keeps the gas each lens call was sent.
type refusingNode struct {
	t           *testing.T
	lens        *Contract
	refuseAbove uint64
	mu          sync.Mutex
	sent        []uint64
}

func (n *refusingNode) ChainID(context.Context) (*big.Int, error) { return big.NewInt(1), nil }

func (n *refusingNode) CallContract(_ context.Context, msg ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	if bytes.Equal(msg.Data, probeCode) {
		return nil, failing{msg: "no probe"}
	}
	n.mu.Lock()
	n.sent = append(n.sent, msg.Gas)
	n.mu.Unlock()
	if msg.Gas > n.refuseAbove {
		return nil, rpc.HTTPError{StatusCode: http.StatusBadGateway, Status: "502 Bad Gateway"}
	}
	args, err := n.lens.UnpackArgs(msg.Data)
	require.NoError(n.t, err)
	from, to := args[1].(int16), args[2].(int16)
	out, err := n.lens.PackResult(big.NewInt(60), big.NewInt(1<<40), big.NewInt(0), big.NewInt(0),
		big.NewInt(100), from+1, from >= to, []struct{ Tick, LiquidityNet, LiquidityGross *big.Int }{})
	require.NoError(n.t, err)
	return nil, dataError{hexutil.Encode(out)}
}

// TestReadTicksKeepsTheGasARefusalSettles: a node that refuses a large response refuses the first
// page of a read once, and the pages after it are sent the gas that page settled on rather than
// refused again each.
func TestReadTicksKeepsTheGasARefusalSettles(t *testing.T) {
	raw, err := os.ReadFile("../../liquidity-source/uniswap/v3/abis/lens.json")
	require.NoError(t, err)
	node := &refusingNode{t: t, lens: MustLoad("V3Lens", raw), refuseAbove: 25_000_000}

	p, err := ReadTicks(t.Context(), node, node.lens, 50_000_000, 0, 7, nil, common.Address{1})
	require.NoError(t, err)
	assert.EqualValues(t, 100, p.Block)

	refused := 0
	for _, gas := range node.sent {
		if gas > node.refuseAbove {
			refused++
		}
	}
	assert.Equal(t, 1, refused, "gas sent: %v", node.sent)
	assert.Len(t, node.sent, 9, "one refused call and a call for each of the 8 words")
}
