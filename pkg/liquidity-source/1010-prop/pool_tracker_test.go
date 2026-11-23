package prop

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

const (
	testRouterAddr      = "0x4a8E34cfE4f643132e8de0E9752054a9aC862816"
	testRouterAddrLower = "0x4a8e34cfe4f643132e8de0e9752054a9ac862816"
	testOtherRouterAddr = "0x1111111111111111111111111111111111111111"
	testTokenA          = "0x101000000000000000000000000000000000000a"
	testTokenB          = "0x101000000000000000000000000000000000000b"
	testTokenC          = "0x101000000000000000000000000000000000000c"
	testTokenMissing    = "0x10100000000000000000000000000000000000ff"
)

func testPoolTokens(a, b string) []*entity.PoolToken {
	return []*entity.PoolToken{
		{Address: a, Swappable: true},
		{Address: b, Swappable: true},
	}
}

func makeSnapshot(block int64, balA, balB, balC int64) *reservesSnapshot {
	return &reservesSnapshot{
		balanceByAddr: map[string]*big.Int{
			testTokenA: big.NewInt(balA),
			testTokenB: big.NewInt(balB),
			testTokenC: big.NewInt(balC),
		},
		blockNumber: big.NewInt(block),
	}
}

func TestConfig_ReservesCacheTTLMs_BackwardsCompatible(t *testing.T) {
	t.Parallel()

	var cfg Config
	require.NoError(t, json.Unmarshal([]byte(`{"dexID":"1010-prop","chainId":56,"routerAddress":"`+testRouterAddr+`","buffer":9999}`), &cfg))
	assert.Equal(t, int64(0), cfg.ReservesCacheTTLMs)

	tracker := NewPoolTracker(&cfg, nil)
	assert.Equal(t, 200*time.Millisecond, tracker.reservesCacheTTL())

	require.NoError(t, json.Unmarshal([]byte(`{"reservesCacheTTLMs":500}`), &cfg))
	assert.Equal(t, 500*time.Millisecond, tracker.reservesCacheTTL())

	require.NoError(t, json.Unmarshal([]byte(`{"reservesCacheTTLMs":-1}`), &cfg))
	assert.True(t, tracker.reservesCacheTTL() < 0)
}

func TestPoolTracker_FetchBalances_ConcurrentSingleflightAndCacheHit(t *testing.T) {
	t.Parallel()

	tracker := NewPoolTracker(&Config{ReservesCacheTTLMs: 200}, nil)
	now := time.Unix(1000, 0)
	tracker.now = func() time.Time { return now }

	var calls atomic.Int32
	gate := make(chan struct{})
	tracker.fetchReserves = func(ctx context.Context, routerAddr string) (*reservesSnapshot, error) {
		calls.Add(1)
		<-gate
		return makeSnapshot(12345, 100, 200, 300), nil
	}

	const n = 20
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make([][]*big.Int, n)
	blocks := make([]*big.Int, n)
	errs := make([]error, n)

	for i := range n {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			router := testRouterAddr
			if idx%2 == 1 {
				router = testRouterAddrLower
			}
			results[idx], blocks[idx], errs[idx] = tracker.fetchBalances(
				context.Background(),
				router,
				testPoolTokens(testTokenA, testTokenB),
			)
		}(i)
	}

	close(start)
	// Allow all goroutines to enter fetchBalances / singleflight before releasing.
	require.Eventually(t, func() bool {
		return calls.Load() == 1
	}, time.Second, time.Millisecond)
	close(gate)
	wg.Wait()

	assert.Equal(t, int32(1), calls.Load())
	for i := range n {
		require.NoError(t, errs[i])
		assert.Equal(t, big.NewInt(12345), blocks[i])
		assert.Equal(t, []*big.Int{big.NewInt(100), big.NewInt(200)}, results[i])
	}

	// Sequential call within TTL hits the cache without triggering another fetch.
	now = now.Add(199 * time.Millisecond)
	bals, block, err := tracker.fetchBalances(
		context.Background(),
		testRouterAddrLower,
		testPoolTokens(testTokenB, testTokenC),
	)
	require.NoError(t, err)
	assert.Equal(t, int32(1), calls.Load())
	assert.Equal(t, big.NewInt(12345), block)
	assert.Equal(t, []*big.Int{big.NewInt(200), big.NewInt(300)}, bals)
}

func TestPoolTracker_FetchBalances_TTLExpiryRefetches(t *testing.T) {
	t.Parallel()

	tracker := NewPoolTracker(&Config{ReservesCacheTTLMs: 200}, nil)
	var mu sync.Mutex
	now := time.Unix(2000, 0)
	tracker.now = func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	}

	var calls atomic.Int32
	tracker.fetchReserves = func(ctx context.Context, routerAddr string) (*reservesSnapshot, error) {
		c := calls.Add(1)
		return makeSnapshot(int64(100+c), int64(1000*c), int64(2000*c), 3000), nil
	}

	bals, block, err := tracker.fetchBalances(context.Background(), testRouterAddr, testPoolTokens(testTokenA, testTokenB))
	require.NoError(t, err)
	assert.Equal(t, int32(1), calls.Load())
	assert.Equal(t, big.NewInt(101), block)
	assert.Equal(t, []*big.Int{big.NewInt(1000), big.NewInt(2000)}, bals)

	// Advance past TTL (>= 200ms) -> must refetch.
	mu.Lock()
	now = now.Add(200 * time.Millisecond)
	mu.Unlock()

	bals, block, err = tracker.fetchBalances(context.Background(), testRouterAddr, testPoolTokens(testTokenA, testTokenB))
	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load())
	assert.Equal(t, big.NewInt(102), block)
	assert.Equal(t, []*big.Int{big.NewInt(2000), big.NewInt(4000)}, bals)
}

func TestPoolTracker_FetchBalances_CancelledCallerDoesNotFailOtherWaiters(t *testing.T) {
	t.Parallel()

	tracker := NewPoolTracker(&Config{ReservesCacheTTLMs: 200}, nil)

	var calls atomic.Int32
	fetchStarted := make(chan struct{})
	allowFinish := make(chan struct{})
	var fetchCtxErr error

	tracker.fetchReserves = func(ctx context.Context, routerAddr string) (*reservesSnapshot, error) {
		calls.Add(1)
		close(fetchStarted)
		<-allowFinish
		fetchCtxErr = ctx.Err()
		return makeSnapshot(555, 700, 800, 900), nil
	}

	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()

	type result struct {
		bals  []*big.Int
		block *big.Int
		err   error
	}
	res1Ch := make(chan result, 1)
	res2Ch := make(chan result, 1)

	// Leader starts the shared fetch.
	go func() {
		bals, block, err := tracker.fetchBalances(ctx1, testRouterAddr, testPoolTokens(testTokenA, testTokenB))
		res1Ch <- result{bals, block, err}
	}()

	<-fetchStarted

	// Second waiter joins the same singleflight call.
	go func() {
		bals, block, err := tracker.fetchBalances(context.Background(), testRouterAddr, testPoolTokens(testTokenA, testTokenC))
		res2Ch <- result{bals, block, err}
	}()

	// Cancel leader's context; leader should exit early with context.Canceled.
	cancel1()
	res1 := <-res1Ch
	require.ErrorIs(t, res1.err, context.Canceled)

	// Release shared fetch; second waiter must succeed.
	close(allowFinish)
	res2 := <-res2Ch
	require.NoError(t, res2.err)
	require.NoError(t, fetchCtxErr)
	assert.Equal(t, int32(1), calls.Load())
	assert.Equal(t, big.NewInt(555), res2.block)
	assert.Equal(t, []*big.Int{big.NewInt(700), big.NewInt(900)}, res2.bals)
}

func TestPoolTracker_FetchBalances_ErrorNotCached(t *testing.T) {
	t.Parallel()

	tracker := NewPoolTracker(&Config{ReservesCacheTTLMs: 200}, nil)
	errRPC := errors.New("rpc boom")

	var calls atomic.Int32
	tracker.fetchReserves = func(ctx context.Context, routerAddr string) (*reservesSnapshot, error) {
		if calls.Add(1) == 1 {
			return nil, errRPC
		}
		return makeSnapshot(777, 10, 20, 30), nil
	}

	_, _, err := tracker.fetchBalances(context.Background(), testRouterAddr, testPoolTokens(testTokenA, testTokenB))
	require.ErrorIs(t, err, errRPC)
	assert.Equal(t, int32(1), calls.Load())

	// Next call must retry immediately because the error was not cached.
	bals, block, err := tracker.fetchBalances(context.Background(), testRouterAddr, testPoolTokens(testTokenA, testTokenB))
	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load())
	assert.Equal(t, big.NewInt(777), block)
	assert.Equal(t, []*big.Int{big.NewInt(10), big.NewInt(20)}, bals)
}

func TestPoolTracker_FetchBalances_ReturnedBalancesAreCopies(t *testing.T) {
	t.Parallel()

	tracker := NewPoolTracker(&Config{ReservesCacheTTLMs: 200}, nil)
	tracker.fetchReserves = func(ctx context.Context, routerAddr string) (*reservesSnapshot, error) {
		return makeSnapshot(888, 111, 222, 333), nil
	}

	bals1, block1, err := tracker.fetchBalances(context.Background(), testRouterAddr, testPoolTokens(testTokenA, testTokenB))
	require.NoError(t, err)
	require.Equal(t, []*big.Int{big.NewInt(111), big.NewInt(222)}, bals1)
	require.Equal(t, big.NewInt(888), block1)

	// Mutate returned values aggressively.
	bals1[0].SetInt64(999999)
	bals1[1].Add(bals1[1], big.NewInt(500000))
	block1.SetInt64(0)

	// Cached values must remain untouched.
	bals2, block2, err := tracker.fetchBalances(context.Background(), testRouterAddr, testPoolTokens(testTokenA, testTokenB))
	require.NoError(t, err)
	assert.Equal(t, []*big.Int{big.NewInt(111), big.NewInt(222)}, bals2)
	assert.Equal(t, big.NewInt(888), block2)
}

func TestPoolTracker_FetchBalances_NegativeTTLDisablesCaching(t *testing.T) {
	t.Parallel()

	tracker := NewPoolTracker(&Config{ReservesCacheTTLMs: -1}, nil)
	var calls atomic.Int32
	tracker.fetchReserves = func(ctx context.Context, routerAddr string) (*reservesSnapshot, error) {
		c := calls.Add(1)
		return makeSnapshot(int64(200+c), int64(c*10), int64(c*20), 30), nil
	}

	for i := int32(1); i <= 3; i++ {
		bals, block, err := tracker.fetchBalances(context.Background(), testRouterAddr, testPoolTokens(testTokenA, testTokenB))
		require.NoError(t, err)
		assert.Equal(t, i, calls.Load())
		assert.Equal(t, big.NewInt(int64(200+i)), block)
		assert.Equal(t, []*big.Int{big.NewInt(int64(i * 10)), big.NewInt(int64(i * 20))}, bals)
	}
}

func TestPoolTracker_FetchBalances_DifferentRoutersAndMissingToken(t *testing.T) {
	t.Parallel()

	tracker := NewPoolTracker(&Config{ReservesCacheTTLMs: 200}, nil)
	var calls atomic.Int32
	tracker.fetchReserves = func(ctx context.Context, routerAddr string) (*reservesSnapshot, error) {
		calls.Add(1)
		if routerAddr == testOtherRouterAddr {
			return makeSnapshot(999, 50, 60, 70), nil
		}
		return makeSnapshot(888, 10, 20, 30), nil
	}

	bals, _, err := tracker.fetchBalances(context.Background(), testRouterAddr, testPoolTokens(testTokenA, testTokenB))
	require.NoError(t, err)
	assert.Equal(t, []*big.Int{big.NewInt(10), big.NewInt(20)}, bals)
	assert.Equal(t, int32(1), calls.Load())

	// Missing token returns ErrInsufficientLiquidity, while valid snapshot stays cached.
	_, _, err = tracker.fetchBalances(context.Background(), testRouterAddr, testPoolTokens(testTokenA, testTokenMissing))
	require.ErrorIs(t, err, ErrInsufficientLiquidity)
	assert.Equal(t, int32(1), calls.Load())

	// Different router triggers its own fetch.
	otherBals, otherBlock, err := tracker.fetchBalances(context.Background(), testOtherRouterAddr, testPoolTokens(testTokenA, testTokenB))
	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load())
	assert.Equal(t, big.NewInt(999), otherBlock)
	assert.Equal(t, []*big.Int{big.NewInt(50), big.NewInt(60)}, otherBals)
}
