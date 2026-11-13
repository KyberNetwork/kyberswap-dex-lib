package oneinchaqua

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

const testBlock = 4242

// fakeChain answers the three reads of the tracker from maps and can make single reads fail.
type fakeChain struct {
	mu        sync.Mutex
	raw       map[string][2]*big.Int // maker|app|hash|token -> balance, tokensCount
	balance   map[string]*big.Int    // token|maker
	allowance map[string]*big.Int    // token|maker
	fail      map[string]bool        // method|key -> revert
	empty     map[string]bool        // method|key -> success with no data
	reads     int
}

func newFakeChain() *fakeChain {
	return &fakeChain{raw: map[string][2]*big.Int{}, balance: map[string]*big.Int{}, allowance: map[string]*big.Int{},
		fail: map[string]bool{}, empty: map[string]bool{}}
}

func rawKey(maker, app common.Address, hash [32]byte, token common.Address) string {
	return addrHex(maker) + "|" + addrHex(app) + "|" + common.Hash(hash).Hex() + "|" + addrHex(token)
}

func (c *fakeChain) respond(target common.Address, data []byte) (bool, []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reads++
	switch target {
	case testAqua:
		args, err := aquaABI.Methods["rawBalances"].Inputs.Unpack(data[4:])
		if err != nil {
			return false, nil
		}
		key := rawKey(args[0].(common.Address), args[1].(common.Address), args[2].([32]byte), args[3].(common.Address))
		if c.fail["rawBalances|"+key] {
			return false, nil
		}
		v, ok := c.raw[key]
		if !ok {
			v = [2]*big.Int{new(big.Int), new(big.Int)}
		}
		out, _ := aquaABI.Methods["rawBalances"].Outputs.Pack(v[0], v[1])
		return true, out
	default:
		method, err := erc20ABI.MethodById(data)
		if err != nil {
			return false, nil
		}
		args, _ := method.Inputs.Unpack(data[4:])
		key := addrHex(target) + "|" + addrHex(args[0].(common.Address))
		if c.fail[method.Name+"|"+key] {
			return false, nil
		}
		if c.empty[method.Name+"|"+key] {
			return true, nil
		}
		m := c.balance
		if method.Name == "allowance" {
			m = c.allowance
		}
		v := m[key]
		if v == nil {
			v = new(big.Int)
		}
		out, _ := method.Outputs.Pack(v)
		return true, out
	}
}

func (c *fakeChain) setStrategy(o poolOpts, e entity.Pool, count int64, raw0, raw1 *uint256.Int) {
	var static StaticExtra
	_ = json.Unmarshal([]byte(e.StaticExtra), &static)
	hash := common.HexToHash(static.StrategyHash)
	maker, app := common.HexToAddress(static.Maker), common.HexToAddress(static.App)
	for k, raw := range []*uint256.Int{raw0, raw1} {
		token := common.HexToAddress(e.Tokens[k].Address)
		c.raw[rawKey(maker, app, hash, token)] = [2]*big.Int{raw.ToBig(), big.NewInt(count)}
	}
}

func (c *fakeChain) setWallet(maker, token string, balance, allowance *uint256.Int) {
	c.balance[strings.ToLower(token)+"|"+strings.ToLower(maker)] = balance.ToBig()
	c.allowance[strings.ToLower(token)+"|"+strings.ToLower(maker)] = allowance.ToBig()
}

func (c *fakeChain) tracker(t *testing.T, opts ...func(*Config)) (*PoolTracker, *testutil.MulticallStub) {
	t.Helper()
	stub := testutil.NewMulticallStub(t, c.respond).At(testBlock, 1_700_000_000)
	tr, err := NewPoolTracker(testConfig(opts...), stub.Client())
	require.NoError(t, err)
	return tr, stub
}

// listedPool is what the lister makes: static facts, zero state.
func listedPool(t *testing.T, id string, o poolOpts) entity.Pool {
	t.Helper()
	o.strategyId = id
	e := newEntity(t, o)
	e.Extra = "{}"
	e.Reserves = entity.PoolReserves{"0", "0"}
	e.BlockNumber = 0
	e.Timestamp = 0
	return e
}

func decodeExtra(t *testing.T, e entity.Pool) Extra { return mustUnmarshal[Extra](t, e.Extra) }

// An unchanged strategy read at a later block keeps the same Extra (the read blocks stay at the first read of
// the state), so a store that skips unchanged pools does not rewrite a pool whose state did not move.
func TestGetNewPoolStatesExtraStableAcrossBlocks(t *testing.T) {
	t.Parallel()
	chain := newFakeChain()
	e := listedPool(t, "1", poolOpts{})
	chain.setStrategy(poolOpts{}, e, 2, e18(1000), e18(500))
	chain.setWallet(testMaker.Hex(), testToken0, e18(900), e18(5000))
	chain.setWallet(testMaker.Hex(), testToken1, e18(200), unlimited)

	read := func(stored entity.Pool, block uint64) entity.Pool {
		stub := testutil.NewMulticallStub(t, chain.respond).At(block, 1_700_000_000)
		tr, err := NewPoolTracker(testConfig(), stub.Client())
		require.NoError(t, err)
		pools, errs := tr.GetNewPoolStates(context.Background(), []entity.Pool{stored})
		require.NoError(t, errs[0])
		return pools[0]
	}
	first := read(e, testBlock)
	require.Equal(t, uint64(testBlock), decodeExtra(t, first).WalletBlock, "the lister pool has no read block")

	later := read(first, testBlock+100)
	require.Equal(t, uint64(testBlock+100), later.BlockNumber)
	require.Equal(t, first.Extra, later.Extra)
	require.Equal(t, first.Reserves, later.Reserves)
	sim, err := NewPoolSimulator(later)
	require.NoError(t, err)
	_, err = quoteOut(sim, 0, e18(1), nil)
	require.NoError(t, err)

	chain.setWallet(testMaker.Hex(), testToken1, e18(150), unlimited)
	changed := read(later, testBlock+101)
	require.NotEqual(t, first.Extra, changed.Extra, "a wallet change is a state change")
	x := decodeExtra(t, changed)
	require.Equal(t, uint64(testBlock+101), x.RawBlock)
	require.Equal(t, uint64(testBlock+101), x.WalletBlock)

	// A stored state with mixed or missing read blocks is not kept: the new read stamps both blocks.
	mixed := later
	mx := decodeExtra(t, mixed)
	mx.RawBlock = 7
	b, err := json.Marshal(&mx)
	require.NoError(t, err)
	mixed.Extra = string(b)
	x = decodeExtra(t, read(mixed, testBlock+102))
	require.Equal(t, uint64(testBlock+102), x.RawBlock)
	require.Equal(t, uint64(testBlock+102), x.WalletBlock)
}

func TestGetNewPoolStatesLive(t *testing.T) {
	t.Parallel()
	chain := newFakeChain()
	e1, e2 := listedPool(t, "1", poolOpts{}), listedPool(t, "2", poolOpts{program: prog(iFlat(5_000_000), iSwap())})
	chain.setStrategy(poolOpts{}, e1, 2, e18(1000), e18(0))
	chain.setStrategy(poolOpts{}, e2, 2, e18(700), e18(300))
	chain.setWallet(testMaker.Hex(), testToken0, e18(900), e18(5000))
	chain.setWallet(testMaker.Hex(), testToken1, e18(200), unlimited)
	tr, stub := chain.tracker(t)

	pools, errs := tr.GetNewPoolStates(context.Background(), []entity.Pool{e1, e2})
	require.Equal(t, []error{nil, nil}, errs)

	// One multicall; the maker's wallet reads are shared by both pools: 4 raw + 2 pairs.
	targets := stub.Targets()
	require.Len(t, targets, 1)
	require.Len(t, targets[0], 4+2*2)

	for i, p := range pools {
		require.Equal(t, uint64(testBlock), p.BlockNumber)
		require.Greater(t, p.Timestamp, int64(1))
		x := decodeExtra(t, p)
		require.Equal(t, uint64(testBlock), x.RawBlock)
		require.Equal(t, uint64(testBlock), x.WalletBlock, "raw balances and wallet from one block")
		require.False(t, x.Docked || x.Inactive)
		require.Equal(t, e18(900).Dec(), x.WalletBalance0.Dec())
		require.Equal(t, e18(5000).Dec(), x.Allowance0.Dec())
		require.Equal(t, e18(200).Dec(), x.WalletBalance1.Dec())
		_ = i
	}
	// Reserves are the fillable amounts: min(raw, balance, allowance).
	require.Equal(t, entity.PoolReserves{e18(900).Dec(), "0"}, pools[0].Reserves)
	require.Equal(t, entity.PoolReserves{e18(700).Dec(), e18(200).Dec()}, pools[1].Reserves)

	// The simulator built from the tracked pool quotes, and from the lister's pool it does not.
	listed, err := NewPoolSimulator(e2)
	require.NoError(t, err)
	_, err = quoteOut(listed, 0, e18(1), nil)
	require.ErrorIs(t, err, ErrNoWalletSnapshot)
	sim, err := NewPoolSimulator(pools[1])
	require.NoError(t, err)
	res, err := quoteOut(sim, 0, e18(1), nil)
	require.NoError(t, err)
	require.True(t, res.IsValid())
	require.NotEmpty(t, sim.CalculateLimit())
}

func TestGetNewPoolStatesDedupe(t *testing.T) {
	t.Parallel()
	chain := newFakeChain()
	var pools []entity.Pool
	// Five strategies of one maker on the same pair, one more of another maker.
	for i := range 5 {
		e := listedPool(t, string(rune('a'+i)), poolOpts{program: prog(iFlat(uint32(i+1)), iSwap())})
		chain.setStrategy(poolOpts{}, e, 2, e18(10), e18(10))
		pools = append(pools, e)
	}
	other := listedPool(t, "z", poolOpts{maker: common.HexToAddress("0x1111111111111111111111111111111111111111")})
	chain.setStrategy(poolOpts{maker: common.HexToAddress("0x1111111111111111111111111111111111111111")}, other, 2, e18(10), e18(10))
	pools = append(pools, other)
	tr, stub := chain.tracker(t)
	_, errs := tr.GetNewPoolStates(context.Background(), pools)
	require.Equal(t, make([]error, 6), errs)
	targets := stub.Targets()
	require.Len(t, targets, 1)
	require.Len(t, targets[0], 6*2+2*2*2, "12 raw reads, 2 makers x 2 tokens x (balanceOf, allowance)")
}

func TestTerminalPools(t *testing.T) {
	t.Parallel()
	chain := newFakeChain()
	docked, inactive := listedPool(t, "d", poolOpts{}), listedPool(t, "i", poolOpts{program: prog(iFlat(2), iSwap())})
	chain.setStrategy(poolOpts{}, docked, tokensCountDocked, e18(0), e18(0))
	chain.setStrategy(poolOpts{}, inactive, 0, e18(0), e18(0))
	chain.setWallet(testMaker.Hex(), testToken0, e18(900), unlimited)
	chain.setWallet(testMaker.Hex(), testToken1, e18(900), unlimited)
	tr, stub := chain.tracker(t)

	pools, errs := tr.GetNewPoolStates(context.Background(), []entity.Pool{docked, inactive})
	require.Equal(t, make([]error, 2), errs)
	for i, p := range pools {
		require.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)
		if i == 0 {
			require.Equal(t, int64(1), p.Timestamp, "docked: never 0, which counts as always active")
		} else {
			require.Greater(t, p.Timestamp, int64(1_700_000_000), "inactive: the time it was first seen inactive")
		}
		require.Equal(t, uint64(testBlock), p.BlockNumber)
		sim, err := NewPoolSimulator(p)
		require.NoError(t, err)
		require.Nil(t, sim.CalculateLimit())
		_, err = quoteOut(sim, 0, e18(1), nil)
		require.ErrorIs(t, err, ErrStrategyTerminal)
	}
	require.True(t, decodeExtra(t, pools[0]).Docked)
	require.False(t, decodeExtra(t, pools[1]).Docked)
	require.True(t, decodeExtra(t, pools[1]).Inactive)

	// A docked pool is permanent: the next refresh returns it unchanged and reads nothing.
	before := len(stub.Targets())
	again, errs := tr.GetNewPoolStates(context.Background(), []entity.Pool{pools[0]})
	require.Equal(t, []error{nil}, errs)
	require.Equal(t, pools[0], again[0])
	require.Len(t, stub.Targets(), before)

	// An inactive pool is read again; once the strategy is shipped for real it goes live.
	chain.setStrategy(poolOpts{}, inactive, 2, e18(10), e18(10))
	revived, errs := tr.GetNewPoolStates(context.Background(), []entity.Pool{pools[1]})
	require.Equal(t, []error{nil}, errs)
	require.False(t, decodeExtra(t, revived[0]).Inactive)
	require.Greater(t, revived[0].Timestamp, int64(1))
}

// Wallet-limited and live, then docked while the maker tops up the wallet. At no step
// does the pool quote.
func TestDockedWhileDormantTracked(t *testing.T) {
	t.Parallel()
	chain := newFakeChain()
	e := listedPool(t, "dormant", poolOpts{})
	chain.setStrategy(poolOpts{}, e, 2, e18(500), e18(500))
	chain.setWallet(testMaker.Hex(), testToken0, e18(0), unlimited)
	chain.setWallet(testMaker.Hex(), testToken1, e18(0), unlimited)
	tr, _ := chain.tracker(t)

	step1, errs := tr.GetNewPoolStates(context.Background(), []entity.Pool{e})
	require.Equal(t, []error{nil}, errs)
	sim, err := NewPoolSimulator(step1[0])
	require.NoError(t, err)
	for in := range 2 {
		_, err = quoteOut(sim, in, e18(1), nil)
		require.ErrorIs(t, err, ErrInsufficientWallet)
	}
	require.Equal(t, entity.PoolReserves{"0", "0"}, step1[0].Reserves)

	// The strategy is docked and the maker tops up: the next read sees both in one block.
	chain.setStrategy(poolOpts{}, e, tokensCountDocked, e18(0), e18(0))
	chain.setWallet(testMaker.Hex(), testToken0, e18(900), unlimited)
	chain.setWallet(testMaker.Hex(), testToken1, e18(900), unlimited)
	step2, errs := tr.GetNewPoolStates(context.Background(), []entity.Pool{step1[0]})
	require.Equal(t, []error{nil}, errs)
	sim, err = NewPoolSimulator(step2[0])
	require.NoError(t, err)
	for in := range 2 {
		_, err = quoteOut(sim, in, e18(1), nil)
		require.ErrorIs(t, err, ErrStrategyTerminal)
	}
	require.Equal(t, entity.PoolReserves{"0", "0"}, step2[0].Reserves)
	require.Equal(t, int64(1), step2[0].Timestamp)
	require.Nil(t, sim.CalculateLimit())
}

// A sub-call of one pool that reverts, in a batch that
// succeeds, writes that pool fail-closed. Only a transport error keeps the stored state.
func TestSubCallRevertsFailClosed(t *testing.T) {
	t.Parallel()
	chain := newFakeChain()
	a, b := listedPool(t, "a", poolOpts{}), listedPool(t, "b", poolOpts{maker: common.HexToAddress("0x2222222222222222222222222222222222222222")})
	chain.setStrategy(poolOpts{}, a, 2, e18(100), e18(100))
	chain.setStrategy(poolOpts{maker: common.HexToAddress("0x2222222222222222222222222222222222222222")}, b, 2, e18(100), e18(100))
	other := "0x2222222222222222222222222222222222222222"
	chain.setWallet(testMaker.Hex(), testToken0, e18(50), unlimited)
	chain.setWallet(testMaker.Hex(), testToken1, e18(60), unlimited)
	chain.setWallet(other, testToken0, e18(70), unlimited)
	chain.setWallet(other, testToken1, e18(80), unlimited)
	tr, _ := chain.tracker(t)

	// Healthy baseline, so the stored pools hold a live, fillable state.
	base, errs := tr.GetNewPoolStates(context.Background(), []entity.Pool{a, b})
	require.Equal(t, make([]error, 2), errs)
	require.Equal(t, entity.PoolReserves{e18(50).Dec(), e18(60).Dec()}, base[0].Reserves)

	for name, tc := range map[string]struct {
		set     func()
		check   func(p entity.Pool)
		wantErr error
	}{
		"balanceOf reverts for token0": {
			set: func() {
				chain.fail["balanceOf|"+strings.ToLower(testToken0)+"|"+addrHex(testMaker)] = true
			},
			check: func(p entity.Pool) {
				require.Equal(t, entity.PoolReserves{"0", e18(60).Dec()}, p.Reserves, "token0 has no liquidity, token1 keeps its own")
				x := decodeExtra(t, p)
				require.True(t, x.WalletBalance0.IsZero() && x.Allowance0.IsZero(), "zero wallet limit for the broken token")
				require.Equal(t, e18(60).Dec(), x.WalletBalance1.Dec())
				require.Equal(t, uint64(testBlock), x.WalletBlock)
				sim, err := NewPoolSimulator(p)
				require.NoError(t, err)
				limit := sim.CalculateLimit()
				require.Equal(t, "0", limit[sim.limitKeys[0]].String())
				require.Equal(t, e18(60).Dec(), limit[sim.limitKeys[1]].String())
				_, err = quoteOut(sim, 1, e18(1), nil) // buying token0 out of an empty limit
				require.ErrorIs(t, err, ErrInsufficientWallet)
			},
		},
		"allowance returns no data": {
			set: func() {
				chain.empty["allowance|"+strings.ToLower(testToken1)+"|"+addrHex(testMaker)] = true
			},
			check: func(p entity.Pool) {
				require.Equal(t, entity.PoolReserves{e18(50).Dec(), "0"}, p.Reserves)
			},
		},
		"rawBalances reverts": {
			set: func() {
				chain.fail["rawBalances|"+rawKey(testMaker, testRouter, common.HexToHash(decodeStatic(t, a).StrategyHash), common.HexToAddress(testToken1))] = true
			},
			wantErr: ErrRawRead,
			check: func(p entity.Pool) {
				require.Equal(t, base[0], p, "the pool keeps its previous state, it is not marked terminal")
				require.False(t, decodeExtra(t, p).Inactive)
			},
		},
	} {
		chain.mu.Lock()
		chain.fail, chain.empty = map[string]bool{}, map[string]bool{}
		chain.mu.Unlock()
		tc.set()
		pools, errs := tr.GetNewPoolStates(context.Background(), base)
		require.ErrorIs(t, errs[0], tc.wantErr, name)
		require.NoError(t, errs[1], name)
		tc.check(pools[0])
		// The other maker's pool is unaffected.
		require.Equal(t, entity.PoolReserves{e18(70).Dec(), e18(80).Dec()}, pools[1].Reserves, name)
		require.Equal(t, uint64(testBlock), pools[1].BlockNumber)
	}
}

func decodeStatic(t *testing.T, e entity.Pool) StaticExtra {
	return mustUnmarshal[StaticExtra](t, e.StaticExtra)
}

func TestTransportErrorKeepsStoredState(t *testing.T) {
	t.Parallel()
	chain := newFakeChain()
	e := listedPool(t, "1", poolOpts{})
	chain.setStrategy(poolOpts{}, e, 2, e18(100), e18(100))
	chain.setWallet(testMaker.Hex(), testToken0, e18(50), unlimited)
	chain.setWallet(testMaker.Hex(), testToken1, e18(60), unlimited)
	tr, stub := chain.tracker(t)
	pools, errs := tr.GetNewPoolStates(context.Background(), []entity.Pool{e})
	require.Equal(t, []error{nil}, errs)

	stub.Close()
	again, errs := tr.GetNewPoolStates(context.Background(), []entity.Pool{pools[0]})
	require.Error(t, errs[0])
	require.Equal(t, pools[0], again[0], "the stored pool stands")
}

func TestGetNewPoolStatesInputErrors(t *testing.T) {
	t.Parallel()
	chain := newFakeChain()
	good := listedPool(t, "1", poolOpts{})
	chain.setStrategy(poolOpts{}, good, 2, e18(1), e18(1))
	tr, _ := chain.tracker(t)

	badStatic := good
	badStatic.StaticExtra = "{"
	badExtra := good
	badExtra.Extra = "{"
	oneToken := good
	oneToken.Tokens = good.Tokens[:1]
	badHash := listedPool(t, "2", poolOpts{})
	var s StaticExtra
	require.NoError(t, json.Unmarshal([]byte(badHash.StaticExtra), &s))
	s.StrategyHash = "0x12"
	b, _ := json.Marshal(s)
	badHash.StaticExtra = string(b)
	badMaker := listedPool(t, "3", poolOpts{})
	require.NoError(t, json.Unmarshal([]byte(badMaker.StaticExtra), &s))
	s.Maker = "nope"
	b, _ = json.Marshal(s)
	badMaker.StaticExtra = string(b)

	pools, errs := tr.GetNewPoolStates(context.Background(), []entity.Pool{badStatic, badExtra, oneToken, badHash, badMaker, good})
	for i := range 5 {
		require.Error(t, errs[i], i)
	}
	require.NoError(t, errs[5])
	require.Equal(t, uint64(testBlock), pools[5].BlockNumber)

	// An empty batch and a batch of only docked pools read nothing.
	out, errs := tr.GetNewPoolStates(context.Background(), nil)
	require.Empty(t, out)
	require.Empty(t, errs)

	_, err := NewPoolTracker(testConfig(func(c *Config) { c.AquaAddress = "nope" }), nil)
	require.Error(t, err)
}

// GetNewPoolState is the same code on one pool.
func TestGetNewPoolStateMatchesBatch(t *testing.T) {
	t.Parallel()
	chain := newFakeChain()
	e := listedPool(t, "1", poolOpts{})
	chain.setStrategy(poolOpts{}, e, 2, e18(100), e18(100))
	chain.setWallet(testMaker.Hex(), testToken0, e18(50), unlimited)
	chain.setWallet(testMaker.Hex(), testToken1, e18(60), unlimited)
	tr, _ := chain.tracker(t)
	single, err := tr.GetNewPoolState(context.Background(), e, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	batch, errs := tr.GetNewPoolStates(context.Background(), []entity.Pool{e})
	require.NoError(t, errs[0])
	single.Timestamp, batch[0].Timestamp = 0, 0
	require.Equal(t, batch[0], single)

	_, err = tr.GetNewPoolState(context.Background(), entity.Pool{}, pool.GetNewPoolStateParams{})
	require.Error(t, err)
}

// Two refreshes of one stored pool at once give the same state (run with -race).
func TestConcurrentRefresh(t *testing.T) {
	t.Parallel()
	chain := newFakeChain()
	e := listedPool(t, "1", poolOpts{})
	chain.setStrategy(poolOpts{}, e, 2, e18(100), e18(100))
	chain.setWallet(testMaker.Hex(), testToken0, e18(50), unlimited)
	chain.setWallet(testMaker.Hex(), testToken1, e18(60), unlimited)
	tr, _ := chain.tracker(t)
	var wg sync.WaitGroup
	results := make([]entity.Pool, 8)
	for i := range results {
		wg.Go(func() {
			p, err := tr.GetNewPoolState(context.Background(), e, pool.GetNewPoolStateParams{})
			require.NoError(t, err)
			p.Timestamp = 0
			results[i] = p
		})
	}
	wg.Wait()
	for _, r := range results[1:] {
		require.Equal(t, results[0], r)
	}
}

func TestTrackerMalformedValues(t *testing.T) {
	t.Parallel()
	e := listedPool(t, "1", poolOpts{})
	tp, docked, err := newTrackedPool(0, e)
	require.NoError(t, err)
	require.False(t, docked)
	all := make([]bool, 6)
	for i := range all {
		all[i] = true
	}
	tp.rawAt = [2]int{0, 3}
	w0, w1 := &walletRead{balanceAt: 1, balance: big.NewInt(1), allowance: big.NewInt(1)}, &walletRead{balanceAt: 4, balance: big.NewInt(1), allowance: big.NewInt(1)}
	tp.wallet = [2]*walletRead{w0, w1}
	big248 := new(big.Int).Lsh(big.NewInt(1), 248)
	for name, raw := range map[string][2]rawBalancesOutput{
		"balance above uint248": {{Balance: big248, TokensCount: big.NewInt(2)}, {Balance: big.NewInt(1), TokensCount: big.NewInt(2)}},
		"negative balance":      {{Balance: big.NewInt(-1), TokensCount: big.NewInt(2)}, {Balance: big.NewInt(1), TokensCount: big.NewInt(2)}},
		"nil output":            {{}, {Balance: big.NewInt(1), TokensCount: big.NewInt(2)}},
	} {
		tp.raw = raw
		p, err := tp.build(all, 9, 99, 0)
		require.ErrorIs(t, err, ErrRawRead, name)
		require.Equal(t, e, p, name+": the stored pool stands")
	}
	// A huge tokensCount is not a count: inactive.
	tp.raw = [2]rawBalancesOutput{{Balance: big.NewInt(1), TokensCount: new(big.Int).Lsh(big.NewInt(1), 70)}, {Balance: big.NewInt(1), TokensCount: big.NewInt(2)}}
	p, err := tp.build(all, 9, 99, 0)
	require.NoError(t, err)
	require.True(t, decodeExtra(t, p).Inactive)
	// Docked on one token wins over a zero count on the other.
	tp.raw = [2]rawBalancesOutput{{Balance: big.NewInt(0), TokensCount: big.NewInt(tokensCountDocked)}, {Balance: big.NewInt(0), TokensCount: big.NewInt(0)}}
	p, err = tp.build(all, 9, 99, 0)
	require.NoError(t, err)
	require.True(t, decodeExtra(t, p).Docked)

	// Wallet values that are not uint256 fail closed for that token.
	tp.raw = [2]rawBalancesOutput{{Balance: big.NewInt(5), TokensCount: big.NewInt(2)}, {Balance: big.NewInt(5), TokensCount: big.NewInt(2)}}
	w0.balance = new(big.Int).Lsh(big.NewInt(1), 256)
	w1.allowance = big.NewInt(-1)
	p, err = tp.build(all, 9, 99, 0)
	require.NoError(t, err)
	require.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)
	w0.balance, w1.allowance = big.NewInt(3), big.NewInt(4)
	p, err = tp.build(all, 9, 99, 0)
	require.NoError(t, err)
	require.Equal(t, entity.PoolReserves{"1", "1"}, p.Reserves, "min(raw, balance, allowance) per token")
	require.Equal(t, int64(99), p.Timestamp)
	_, _, good := walletLimit(&walletRead{balanceAt: 1, balance: big.NewInt(1), allowance: big.NewInt(1)}, []bool{true, false, true})
	require.False(t, good)
}

func TestSimulatorFactory(t *testing.T) {
	t.Parallel()
	e := newEntity(t, poolOpts{})
	sim, err := pool.Factory(DexType)(pool.FactoryParams{EntityPool: e})
	require.NoError(t, err)
	require.IsType(t, &PoolSimulator{}, sim)
	_, err = pool.Factory(DexType)(pool.FactoryParams{EntityPool: entity.Pool{}})
	require.Error(t, err)
}

// Tracker rules for the terminal states: Aqua is refreshed by interval only, so the rules for a node that is
// a block behind, for a dock that a reorg removes and for a raw read that fails are in the refresh itself.

func withConfirmations(n uint64) func(*Config) { return func(c *Config) { c.Confirmations = &n } }

// refresh runs one refresh of a single pool at the block.
func refresh(t *testing.T, tr *PoolTracker, stub *testutil.MulticallStub, block uint64, p entity.Pool) (entity.Pool, error) {
	t.Helper()
	stub.At(block, 1_700_000_000+block)
	out, errs := tr.GetNewPoolStates(context.Background(), []entity.Pool{p})
	return out[0], errs[0]
}

func mustMarshal(t testing.TB, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

func liveChain(t *testing.T) (*fakeChain, entity.Pool) {
	t.Helper()
	chain := newFakeChain()
	e := listedPool(t, "r", poolOpts{})
	chain.setStrategy(poolOpts{}, e, 2, e18(100), e18(100))
	chain.setWallet(testMaker.Hex(), testToken0, e18(50), unlimited)
	chain.setWallet(testMaker.Hex(), testToken1, e18(60), unlimited)
	return chain, e
}

// indexerActive models the indexer's activity rule for an exchange with an inactivity duration: a pool is
// active while its timestamp is within the duration of now. A pool that is not active is archived and removed
// from the refresh set, and nothing brings it back.
const inactiveDuration = 24 * time.Hour

func indexerActive(p entity.Pool, now time.Time) bool {
	return now.Unix()-p.Timestamp <= int64(inactiveDuration.Seconds())
}

func quotes(t *testing.T, p entity.Pool) error {
	t.Helper()
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)
	_, err = quoteOut(sim, 0, e18(1), nil)
	return err
}

// A node that is a block behind says tokensCount 0 for a strategy that is shipped: the pool is inactive with
// zero reserves and no limit keys until the next refresh, which makes it live again.
func TestLaggingNodeZeroRecovers(t *testing.T) {
	t.Parallel()
	chain, e := liveChain(t)
	tr, stub := chain.tracker(t)
	live, err := refresh(t, tr, stub, 100, e)
	require.NoError(t, err)
	require.NoError(t, quotes(t, live))

	chain.setStrategy(poolOpts{}, e, 0, e18(0), e18(0)) // the lagging node
	before := time.Now().Unix()
	lag, err := refresh(t, tr, stub, 101, live)
	require.NoError(t, err)
	x := decodeExtra(t, lag)
	require.True(t, x.Inactive)
	require.False(t, x.Docked)
	require.Equal(t, entity.PoolReserves{"0", "0"}, lag.Reserves)
	require.GreaterOrEqual(t, lag.Timestamp, before, "a live pool turning inactive starts the clock now, not at 1")
	sim, err := NewPoolSimulator(lag)
	require.NoError(t, err)
	require.Nil(t, sim.CalculateLimit())
	require.ErrorIs(t, quotes(t, lag), ErrStrategyTerminal)

	chain.setStrategy(poolOpts{}, e, 2, e18(100), e18(100)) // the node caught up
	back, err := refresh(t, tr, stub, 102, lag)
	require.NoError(t, err)
	x = decodeExtra(t, back)
	require.False(t, x.Inactive || x.Docked)
	require.NoError(t, quotes(t, back))
	require.GreaterOrEqual(t, back.Timestamp, lag.Timestamp)
	require.Equal(t, entity.PoolReserves{e18(50).Dec(), e18(60).Dec()}, back.Reserves)
}

// An inactive pool keeps the timestamp of the first inactive read while it stays inactive, so the indexer
// archives it after its inactivity duration; a pool that was live or is new starts the clock at now.
func TestInactiveTimestampCarriesForward(t *testing.T) {
	t.Parallel()
	for _, count := range []int64{0, 1, 3, 0xfe} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			t.Parallel()
			chain, e := liveChain(t)
			chain.setStrategy(poolOpts{}, e, count, e18(0), e18(0))
			tr, stub := chain.tracker(t)

			// new from the lister: Extra {} and Timestamp 0
			before := time.Now().Unix()
			first, err := refresh(t, tr, stub, 100, e)
			require.NoError(t, err)
			require.True(t, decodeExtra(t, first).Inactive)
			require.GreaterOrEqual(t, first.Timestamp, before)

			// still inactive later: the timestamp stays
			first.Timestamp = 1_000_000 // pretend it started long ago
			again, err := refresh(t, tr, stub, 101, first)
			require.NoError(t, err)
			require.Equal(t, int64(1_000_000), again.Timestamp)
			again, err = refresh(t, tr, stub, 102, again)
			require.NoError(t, err)
			require.Equal(t, int64(1_000_000), again.Timestamp)
			require.True(t, decodeExtra(t, again).Inactive)

			// a stored inactive pool with timestamp 1 (never a start time) or 0 gets now
			for _, stale := range []int64{0, 1} {
				first.Timestamp = stale
				p, err := refresh(t, tr, stub, 103, first)
				require.NoError(t, err)
				require.GreaterOrEqual(t, p.Timestamp, before, "stale timestamp %d", stale)
			}

			// it goes live, then inactive again: the clock restarts at now
			chain.setStrategy(poolOpts{}, e, 2, e18(100), e18(100))
			live, err := refresh(t, tr, stub, 104, again)
			require.NoError(t, err)
			require.GreaterOrEqual(t, live.Timestamp, before)
			chain.setStrategy(poolOpts{}, e, count, e18(0), e18(0))
			live.Timestamp = 5_000_000
			again2, err := refresh(t, tr, stub, 105, live)
			require.NoError(t, err)
			require.True(t, decodeExtra(t, again2).Inactive)
			require.GreaterOrEqual(t, again2.Timestamp, before, "the previous state was live: now, not the stored timestamp")
			require.NotEqual(t, int64(5_000_000), again2.Timestamp)
		})
	}
}

// A docked pool is read every refresh until it was read docked `confirmations` blocks after the first docked
// read; then it is final and later refreshes read nothing.
func TestDockedFinalAfterConfirmations(t *testing.T) {
	t.Parallel()
	chain, e := liveChain(t)
	tr, stub := chain.tracker(t, withConfirmations(3))
	chain.setStrategy(poolOpts{}, e, tokensCountDocked, e18(0), e18(0))

	p, err := refresh(t, tr, stub, 100, e)
	require.NoError(t, err)
	check := func(p entity.Pool, atBlock uint64, final bool) {
		x := decodeExtra(t, p)
		require.True(t, x.Docked)
		require.Equal(t, atBlock, x.DockedAtBlock)
		require.Equal(t, final, x.DockedFinal)
		require.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)
		if final {
			require.Equal(t, int64(1), p.Timestamp, "a final dock may be archived")
			require.False(t, indexerActive(p, time.Now()))
		} else {
			require.Greater(t, p.Timestamp, int64(1_700_000_000), "not final: a current timestamp")
			require.True(t, indexerActive(p, time.Now()), "not final: active, so it stays in the refresh set")
		}
		require.ErrorIs(t, quotes(t, p), ErrStrategyTerminal)
		sim, err := NewPoolSimulator(p)
		require.NoError(t, err)
		require.Nil(t, sim.CalculateLimit())
	}
	check(p, 100, false)

	// 101 and 102 are fewer than 3 blocks after 100: read again, the docked-at block stays.
	for _, block := range []uint64{101, 102} {
		reads := len(stub.Targets())
		p, err = refresh(t, tr, stub, block, p)
		require.NoError(t, err)
		require.Len(t, stub.Targets(), reads+1, "block %d: read again", block)
		check(p, 100, false)
	}
	// 103 is 3 blocks after 100: final.
	p, err = refresh(t, tr, stub, 103, p)
	require.NoError(t, err)
	check(p, 100, true)

	// Final: nothing is read, whatever the chain says now (a final dock is permanent).
	chain.setStrategy(poolOpts{}, e, 2, e18(100), e18(100))
	reads := len(stub.Targets())
	again, err := refresh(t, tr, stub, 500, p)
	require.NoError(t, err)
	require.Equal(t, p, again)
	require.Len(t, stub.Targets(), reads)
}

// The head is the block of the same batch's multicall: a final dock in a mixed batch costs nothing, a
// docked pool that is not final rides in the batch of the live pools.
func TestDockedMixedBatch(t *testing.T) {
	t.Parallel()
	chain := newFakeChain()
	live, docking, final := listedPool(t, "live", poolOpts{}), listedPool(t, "docking", poolOpts{program: prog(iFlat(1), iSwap())}), listedPool(t, "final", poolOpts{program: prog(iFlat(2), iSwap())})
	chain.setStrategy(poolOpts{}, live, 2, e18(100), e18(100))
	chain.setStrategy(poolOpts{}, docking, tokensCountDocked, e18(0), e18(0))
	chain.setStrategy(poolOpts{}, final, tokensCountDocked, e18(0), e18(0))
	chain.setWallet(testMaker.Hex(), testToken0, e18(50), unlimited)
	chain.setWallet(testMaker.Hex(), testToken1, e18(60), unlimited)
	tr, stub := chain.tracker(t, withConfirmations(12))
	stub.At(100, 1_700_000_100)
	first, errs := tr.GetNewPoolStates(context.Background(), []entity.Pool{live, docking, final})
	require.Equal(t, make([]error, 3), errs)
	require.False(t, decodeExtra(t, first[1]).DockedFinal)
	require.True(t, indexerActive(first[1], time.Now()), "a dock that is not final stays active")
	// 12 blocks later: the dock is final; the third pool is final already (as if stored by a deep read)
	x := decodeExtra(t, first[2])
	x.DockedFinal = true
	first[2].Extra = mustMarshal(t, x)
	stub.At(112, 1_700_000_112)
	reads := len(stub.Targets())
	second, errs := tr.GetNewPoolStates(context.Background(), first)
	require.Equal(t, make([]error, 3), errs)
	require.Len(t, stub.Targets(), reads+1, "one multicall")
	targets := stub.Targets()[reads]
	// live and docking: 2 raw reads each, one wallet pair per maker token (shared) = 4 + 4
	require.Len(t, targets, 4+2*2, "the final pool is not read")
	require.True(t, decodeExtra(t, second[1]).DockedFinal)
	require.Equal(t, int64(1), second[1].Timestamp, "the dock became final: archiving is allowed from here")
	require.False(t, indexerActive(second[1], time.Now()))
	require.Equal(t, uint64(100), decodeExtra(t, second[1]).DockedAtBlock)
	require.Equal(t, uint64(112), second[0].BlockNumber)
	require.Equal(t, first[2], second[2], "the final pool stands unchanged")
}

// A docked read that a reorg removes: the pool goes back to live; a later dock is a new dock.
func TestDockedReorgedBackToLive(t *testing.T) {
	t.Parallel()
	chain, e := liveChain(t)
	tr, stub := chain.tracker(t, withConfirmations(12))
	chain.setStrategy(poolOpts{}, e, tokensCountDocked, e18(0), e18(0))
	reorgStart := time.Now().Unix()
	p, err := refresh(t, tr, stub, 100, e)
	require.NoError(t, err)
	require.True(t, decodeExtra(t, p).Docked)
	require.False(t, decodeExtra(t, p).DockedFinal)
	require.GreaterOrEqual(t, p.Timestamp, reorgStart, "a dock that is not final has a current timestamp")
	require.True(t, indexerActive(p, time.Now()), "so the indexer does not archive it before the reorg window is over")

	// the dock is reorged away: shipped and funded again
	chain.setStrategy(poolOpts{}, e, 2, e18(100), e18(100))
	p, err = refresh(t, tr, stub, 103, p)
	require.NoError(t, err)
	x := decodeExtra(t, p)
	require.False(t, x.Docked || x.DockedFinal || x.Inactive)
	require.Zero(t, x.DockedAtBlock)
	require.GreaterOrEqual(t, p.Timestamp, reorgStart, "live with a fresh timestamp")
	require.True(t, indexerActive(p, time.Now()))
	require.NoError(t, quotes(t, p))
	require.Equal(t, entity.PoolReserves{e18(50).Dec(), e18(60).Dec()}, p.Reserves)

	// docked for real later: the clock of the dock starts at the new read
	chain.setStrategy(poolOpts{}, e, tokensCountDocked, e18(0), e18(0))
	p, err = refresh(t, tr, stub, 120, p)
	require.NoError(t, err)
	require.Equal(t, uint64(120), decodeExtra(t, p).DockedAtBlock)
	// and one confirmations later it is final
	p, err = refresh(t, tr, stub, 132, p)
	require.NoError(t, err)
	require.True(t, decodeExtra(t, p).DockedFinal)

	// docked, then both the dock and the ship reorged out: inactive (a clock from now), not docked
	chain2, e2 := liveChain(t)
	tr2, stub2 := chain2.tracker(t, withConfirmations(12))
	chain2.setStrategy(poolOpts{}, e2, tokensCountDocked, e18(0), e18(0))
	d, err := refresh(t, tr2, stub2, 100, e2)
	require.NoError(t, err)
	chain2.setStrategy(poolOpts{}, e2, 0, e18(0), e18(0))
	before := time.Now().Unix()
	i, err := refresh(t, tr2, stub2, 101, d)
	require.NoError(t, err)
	x = decodeExtra(t, i)
	require.True(t, x.Inactive)
	require.False(t, x.Docked || x.DockedFinal)
	require.Zero(t, x.DockedAtBlock)
	require.GreaterOrEqual(t, i.Timestamp, before)
}

// A read from a node that is behind the first docked read keeps the earliest block.
func TestDockedAtBlockNeverMovesForward(t *testing.T) {
	t.Parallel()
	chain, e := liveChain(t)
	tr, stub := chain.tracker(t, withConfirmations(5))
	chain.setStrategy(poolOpts{}, e, tokensCountDocked, e18(0), e18(0))
	p, err := refresh(t, tr, stub, 100, e)
	require.NoError(t, err)
	for _, block := range []uint64{102, 98, 101, 102} {
		p, err = refresh(t, tr, stub, block, p)
		require.NoError(t, err)
		x := decodeExtra(t, p)
		if block == 98 {
			require.Equal(t, uint64(98), x.DockedAtBlock, "an earlier read of the dock moves it back")
		}
		require.False(t, x.DockedFinal, "block %d", block)
	}
	p, err = refresh(t, tr, stub, 103, p) // 5 blocks after 98
	require.NoError(t, err)
	require.True(t, decodeExtra(t, p).DockedFinal)

	// a stored pool from before this field (docked, no docked-at block, not final) is read again
	old := decodeExtra(t, e)
	old.Docked = true
	e.Extra = mustMarshal(t, old)
	q, err := refresh(t, tr, stub, 200, e)
	require.NoError(t, err)
	require.Equal(t, uint64(200), decodeExtra(t, q).DockedAtBlock)
	require.False(t, decodeExtra(t, q).DockedFinal)
}

// A rawBalances sub-call that fails or decodes badly returns an error for that pool and keeps its stored
// state, as for a failed batch; the other pools of the batch refresh.
func TestRawReadFailureKeepsOldState(t *testing.T) {
	t.Parallel()
	other := common.HexToAddress("0x2222222222222222222222222222222222222222")
	for name, state := range map[string]func(chain *fakeChain, e entity.Pool) entity.Pool{
		"live": func(chain *fakeChain, e entity.Pool) entity.Pool {
			tr, stub := chain.tracker(t)
			p, err := refresh(t, tr, stub, 90, e)
			require.NoError(t, err)
			return p
		},
		"new from the lister": func(chain *fakeChain, e entity.Pool) entity.Pool { return e },
		"docked, not final": func(chain *fakeChain, e entity.Pool) entity.Pool {
			chain.setStrategy(poolOpts{}, e, tokensCountDocked, e18(0), e18(0))
			tr, stub := chain.tracker(t, withConfirmations(12))
			p, err := refresh(t, tr, stub, 90, e)
			require.NoError(t, err)
			require.False(t, decodeExtra(t, p).DockedFinal)
			chain.setStrategy(poolOpts{}, e, 2, e18(100), e18(100))
			return p
		},
		"inactive": func(chain *fakeChain, e entity.Pool) entity.Pool {
			chain.setStrategy(poolOpts{}, e, 0, e18(0), e18(0))
			tr, stub := chain.tracker(t)
			p, err := refresh(t, tr, stub, 90, e)
			require.NoError(t, err)
			chain.setStrategy(poolOpts{}, e, 2, e18(100), e18(100))
			return p
		},
	} {
		for _, token := range []string{testToken0, testToken1} {
			chain := newFakeChain()
			e := listedPool(t, "x", poolOpts{})
			b := listedPool(t, "b", poolOpts{maker: other})
			chain.setStrategy(poolOpts{}, e, 2, e18(100), e18(100))
			chain.setStrategy(poolOpts{maker: other}, b, 2, e18(10), e18(10))
			for _, m := range []string{testMaker.Hex(), other.Hex()} {
				chain.setWallet(m, testToken0, e18(50), unlimited)
				chain.setWallet(m, testToken1, e18(60), unlimited)
			}
			stored := state(chain, e)
			chain.fail["rawBalances|"+rawKey(testMaker, testRouter, common.HexToHash(decodeStatic(t, e).StrategyHash), common.HexToAddress(token))] = true
			tr, stub := chain.tracker(t, withConfirmations(12))
			stub.At(200, 1_700_000_200)
			pools, errs := tr.GetNewPoolStates(context.Background(), []entity.Pool{stored, b})
			require.ErrorIs(t, errs[0], ErrRawRead, name)
			require.Equal(t, stored, pools[0], name+": the stored pool stands")
			require.NoError(t, errs[1], name)
			require.Equal(t, uint64(200), pools[1].BlockNumber, name)
			require.Equal(t, entity.PoolReserves{e18(10).Dec(), e18(10).Dec()}, pools[1].Reserves)

			// the single-pool path (inactive re-queue, manual refresh) behaves the same
			one, err := tr.GetNewPoolState(context.Background(), stored, pool.GetNewPoolStateParams{})
			require.ErrorIs(t, err, ErrRawRead, name)
			require.Equal(t, stored, one, name)
		}
	}

	// Both reads of a pool bad, a bad decode, and an empty answer.
	chain, e := liveChain(t)
	tr, stub := chain.tracker(t)
	first, err := refresh(t, tr, stub, 100, e)
	require.NoError(t, err)
	hash := common.HexToHash(decodeStatic(t, e).StrategyHash)
	chain.fail["rawBalances|"+rawKey(testMaker, testRouter, hash, common.HexToAddress(testToken0))] = true
	chain.fail["rawBalances|"+rawKey(testMaker, testRouter, hash, common.HexToAddress(testToken1))] = true
	kept, err := refresh(t, tr, stub, 101, first)
	require.ErrorIs(t, err, ErrRawRead)
	require.Equal(t, first, kept)
	require.NoError(t, quotes(t, kept), "the old state still quotes: the failure is not terminal")
}

// The indexer archives a pool that is not active and drops it from the refresh set for good, and Aqua has no
// dependency events to bring it back. So a pool whose dock can still be reorged out must stay active, and
// only a final dock may be archived. This runs the refresh loop with the indexer's rule: a pool that is not
// active after a refresh is archived and never refreshed again.
func TestNotFinalDockIsNeverArchived(t *testing.T) {
	t.Parallel()
	for _, confirmations := range []uint64{0, 3, 12} {
		t.Run(fmt.Sprint(confirmations), func(t *testing.T) {
			t.Parallel()
			chain, e := liveChain(t)
			tr, stub := chain.tracker(t, withConfirmations(confirmations))
			chain.setStrategy(poolOpts{}, e, tokensCountDocked, e18(0), e18(0))
			start := time.Now().Unix()
			p, archived := e, false
			var archivedAt uint64
			for block := uint64(100); block <= 100+confirmations+2 && !archived; block++ {
				var err error
				p, err = refresh(t, tr, stub, block, p)
				require.NoError(t, err)
				x := decodeExtra(t, p)
				require.True(t, x.Docked)
				if indexerActive(p, time.Now()) {
					require.False(t, x.DockedFinal, "block %d: an active pool is not final", block)
					require.GreaterOrEqual(t, p.Timestamp, start)
					continue
				}
				archived, archivedAt = true, block
				require.True(t, x.DockedFinal, "block %d: only a final dock is archived", block)
				require.Equal(t, int64(1), p.Timestamp)
			}
			require.True(t, archived, "the dock becomes final and is archived")
			require.Equal(t, 100+confirmations, archivedAt, "archived at the first final read, confirmations blocks after the first docked read")
		})
	}
	// The case the rule protects: a dock removed by a reorg before it is final, in a pool that was never archived.
	chain, e := liveChain(t)
	tr, stub := chain.tracker(t, withConfirmations(12))
	chain.setStrategy(poolOpts{}, e, tokensCountDocked, e18(0), e18(0))
	p, err := refresh(t, tr, stub, 100, e)
	require.NoError(t, err)
	require.True(t, indexerActive(p, time.Now()))
	chain.setStrategy(poolOpts{}, e, 2, e18(100), e18(100))
	p, err = refresh(t, tr, stub, 105, p)
	require.NoError(t, err)
	require.True(t, indexerActive(p, time.Now()))
	require.NoError(t, quotes(t, p), "the live strategy quotes again")
}
