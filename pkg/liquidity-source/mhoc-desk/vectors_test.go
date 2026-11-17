package mhocdesk

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// testdata/vectors.json.gz is produced by the pool's own bytecode on a local anvil chain (see testdata/README.md):
// every expected amount and error below is what PropAMM.quoteDetailed returned, or what a mined swap paid.

type vecSnapshot struct {
	BlockNumber          uint64 `json:"blockNumber"`
	ReadTimestamp        uint64 `json:"readTimestamp"`
	StateWord            string `json:"stateWord"`
	FloorBase            string `json:"floorBase"`
	FloorQuote           string `json:"floorQuote"`
	MaxNotional          string `json:"maxNotional"`
	MaxNotionalPerUpdate string `json:"maxNotionalPerUpdate"`
	FilledThisUpdate     string `json:"filledThisUpdate"`
	MaxNotionalPerWindow string `json:"maxNotionalPerWindow"`
	FilledThisWindow     string `json:"filledThisWindow"`
	CardSeq              uint32 `json:"cardSeq"`
	CardFilled           string `json:"cardFilled"`
	FillBlock            uint32 `json:"fillBlock"`
	BlockFilled          string `json:"blockFilled"`
	WindowID             uint32 `json:"windowId"`
	WindowFilled         string `json:"windowFilled"`
	BalanceBase          string `json:"balanceBase"`
	BalanceQuote         string `json:"balanceQuote"`
}

type vecQuote struct {
	TokenIn   string  `json:"tokenIn"`
	AmountIn  string  `json:"amountIn"`
	AmountOut string  `json:"amountOut"`
	Error     *string `json:"error"`
}

// vecStep is one mined transaction: a swap (with its fill or revert) or a price update (with the stored word).
type vecStep struct {
	vecQuote
	Type        string `json:"type"`
	BlockNumber uint64 `json:"blockNumber"`
	Timestamp   uint64 `json:"timestamp"`
	StateWord   string `json:"stateWord"`
}

type vecScenario struct {
	Name             string      `json:"name"`
	Snapshot         vecSnapshot `json:"snapshot"`
	QuoteBlockNumber uint64      `json:"quoteBlockNumber"`
	QuoteTimestamp   uint64      `json:"quoteTimestamp"`
	NextBlockNumber  uint64      `json:"nextBlockNumber"`
	NextTimestamp    uint64      `json:"nextTimestamp"`
	Quotes           []vecQuote  `json:"quotes"`
	NextQuotes       []vecQuote  `json:"nextQuotes"`
	Sequence         []vecStep   `json:"sequence"`
}

type vecPool struct {
	BaseDecimals  uint8         `json:"baseDecimals"`
	QuoteDecimals uint8         `json:"quoteDecimals"`
	PriceScale    string        `json:"priceScale"`
	KScale        uint32        `json:"kScale"`
	Scenarios     []vecScenario `json:"scenarios"`
}

type vecFile struct {
	Meta struct {
		ContractSha256 string `json:"contractSha256"`
	} `json:"meta"`
	Pools []vecPool `json:"pools"`
}

const (
	tBase  = "0x00000000000000000000000000000000000000b1"
	tQuote = "0x00000000000000000000000000000000000000b2"
)

func loadVectors(t *testing.T) *vecFile {
	t.Helper()
	path := os.Getenv("MHOCDESK_VECTORS")
	if path == "" {
		path = "testdata/vectors.json.gz"
	}
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	if strings.HasSuffix(path, ".gz") {
		zr, err := gzip.NewReader(bytes.NewReader(raw))
		require.NoError(t, err)
		raw, err = io.ReadAll(zr)
		require.NoError(t, err)
	}
	var v vecFile
	require.NoError(t, json.Unmarshal(raw, &v))
	require.NotEmpty(t, v.Pools)
	return &v
}

func u(s string) *uint256.Int { return uint256.MustFromDecimal(s) }

func snapshotExtra(s vecSnapshot) Extra {
	return Extra{
		StateWord: u(s.StateWord), FloorBase: u(s.FloorBase), FloorQuote: u(s.FloorQuote),
		MaxNotional: u(s.MaxNotional).Uint64(), MaxPerUpdate: u(s.MaxNotionalPerUpdate).Uint64(),
		MaxPerWindow: uint32(u(s.MaxNotionalPerWindow).Uint64()),
		Counters: Counters{
			CardSeq: s.CardSeq, CardFilled: u(s.CardFilled), FillBlock: s.FillBlock, BlockFilled: u(s.BlockFilled),
			WindowID: s.WindowID, WindowFilled: u(s.WindowFilled),
		},
		ReadTimestamp: s.ReadTimestamp,
	}
}

func entityFromSnapshot(t *testing.T, p vecPool, s vecSnapshot) entity.Pool {
	t.Helper()
	extra, err := json.Marshal(snapshotExtra(s))
	require.NoError(t, err)
	static, err := json.Marshal(StaticExtra{BaseDecimals: p.BaseDecimals, QuoteDecimals: p.QuoteDecimals,
		PriceScale: p.PriceScale, KScale: p.KScale})
	require.NoError(t, err)
	return entity.Pool{
		Address: "0x00000000000000000000000000000000000000a1", Exchange: DexType, Type: DexType,
		Timestamp: int64(s.ReadTimestamp), BlockNumber: s.BlockNumber,
		Reserves: entity.PoolReserves{s.BalanceBase, s.BalanceQuote},
		Tokens: []*entity.PoolToken{{Address: tBase, Decimals: p.BaseDecimals, Swappable: true},
			{Address: tQuote, Decimals: p.QuoteDecimals, Swappable: true}},
		Extra:       string(extra),
		StaticExtra: string(static),
	}
}

func errName(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimPrefix(err.Error(), "mhoc-desk: ")
}

func want(e *string) string {
	if e == nil {
		return ""
	}
	return *e
}

// checkQuotes compares every quote of a set with the simulator evaluated in block (bn, ts).
func checkQuotes(t *testing.T, sim *PoolSimulator, label string, quotes []vecQuote, bn, ts uint64) int {
	for _, q := range quotes {
		name := fmt.Sprintf("%s %s %s", label, q.TokenIn, q.AmountIn)
		var gotOut, gotErr string
		if q.TokenIn == "other" {
			_, err := Quote(QuoteInput{D: sim.d, QuoteUnit: sim.quoteUnit, KScale: sim.kScale, Extra: &sim.extra, Unknown: true,
				AmountIn: u(q.AmountIn), BalanceOut: new(uint256.Int), BlockNumber: bn, Now: ts})
			gotErr = errName(err)
		} else {
			res, err := sim.calcAt(params(q), bn, ts)
			gotErr = errName(err)
			if err == nil {
				gotOut = res.TokenAmountOut.Amount.String()
			}
		}
		require.Equal(t, want(q.Error), gotErr, name)
		if q.Error == nil {
			require.Equal(t, q.AmountOut, gotOut, name)
		}
	}
	return len(quotes)
}

// TestVectors_Quotes: from the snapshot alone, every quoteDetailed result (amount or revert reason) is reproduced,
// in the snapshot's own block (where the per-block counter counts) and in the next, empty block (where it has
// restarted: what an aggregator prices on the end-of-block state).
func TestVectors_Quotes(t *testing.T) {
	t.Parallel()
	v := loadVectors(t)
	n := 0
	for _, p := range v.Pools {
		for _, sc := range p.Scenarios {
			sim, err := NewPoolSimulator(entityFromSnapshot(t, p, sc.Snapshot))
			require.NoError(t, err)
			label := fmt.Sprintf("%d/%d %s", p.BaseDecimals, p.QuoteDecimals, sc.Name)
			n += checkQuotes(t, sim, label+" same block", sc.Quotes, sc.QuoteBlockNumber, sc.QuoteTimestamp)
			n += checkQuotes(t, sim, label+" next block", sc.NextQuotes, sc.NextBlockNumber, sc.NextTimestamp)
		}
	}
	t.Logf("%d contract quotes reproduced exactly (contract sha256 %s)", n, v.Meta.ContractSha256)
}

// TestVectors_NextBlockDefault: CalcAmountOut's own clock (the next block after the tracked one, at the indexing
// snapshot time) is the next-block quote set.
func TestVectors_NextBlockDefault(t *testing.T) {
	t.Parallel()
	v := loadVectors(t)
	n := 0
	for _, p := range v.Pools {
		for _, sc := range p.Scenarios {
			sim, err := NewPoolSimulator(entityFromSnapshot(t, p, sc.Snapshot))
			require.NoError(t, err)
			sim.quoteAt = int64(sc.NextTimestamp)
			require.Equal(t, sc.NextBlockNumber, sim.nextBlock())
			for _, q := range sc.NextQuotes {
				if q.TokenIn == "other" {
					continue
				}
				res, err := sim.CalcAmountOut(params(q))
				require.Equal(t, want(q.Error), errName(err), sc.Name, q.AmountIn)
				if err == nil {
					require.Equal(t, q.AmountOut, res.TokenAmountOut.Amount.String(), sc.Name, q.AmountIn)
				}
				n++
			}
		}
	}
	t.Logf("%d next-block quotes reproduced through CalcAmountOut", n)
}

// TestVectors_Sequences: starting from the snapshot, applying UpdateBalance after each simulated fill and the stored
// state word after each price update, every later on-chain swap's fill (or revert) is predicted exactly: second and
// later swaps on a card (slippage on the cumulative fill), several swaps in one block, a newer card in the same
// block as earlier fills (the per-block counter carries over to it), budget exhaustion, floors and a per-minute
// counter that restarts in the next minute.
func TestVectors_Sequences(t *testing.T) {
	t.Parallel()
	v := loadVectors(t)
	filled, updates := 0, 0
	for _, p := range v.Pools {
		for _, sc := range p.Scenarios {
			sim, err := NewPoolSimulator(entityFromSnapshot(t, p, sc.Snapshot))
			require.NoError(t, err)
			cur := sim.CloneState().(*PoolSimulator)
			for i, st := range sc.Sequence {
				name := fmt.Sprintf("%d/%d %s step %d", p.BaseDecimals, p.QuoteDecimals, sc.Name, i)
				if st.Type == "update" {
					cur.extra.StateWord = u(st.StateWord)
					updates++
					continue
				}
				res, err := cur.calcAt(params(st.vecQuote), st.BlockNumber, st.Timestamp)
				require.Equal(t, want(st.Error), errName(err), name)
				if err != nil {
					continue
				}
				require.Equal(t, st.AmountOut, res.TokenAmountOut.Amount.String(), name)
				cur.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn: params(st.vecQuote).TokenAmountIn, TokenAmountOut: *res.TokenAmountOut,
					Fee: *res.Fee, SwapInfo: res.SwapInfo,
				})
				filled++
			}
			// the clone was isolated: the original still holds the snapshot
			require.Equal(t, sc.Snapshot.BalanceBase, sim.Info.Reserves[0].String())
			require.Equal(t, sc.Snapshot.CardFilled, sim.extra.Counters.CardFilled.Dec())
			require.Equal(t, sc.Snapshot.StateWord, sim.extra.StateWord.Dec())
		}
	}
	t.Logf("%d on-chain fills predicted exactly (%d price updates replayed)", filled, updates)
}

func params(q vecQuote) pool.CalcAmountOutParams {
	in, out := tBase, tQuote
	if q.TokenIn == "quote" {
		in, out = tQuote, tBase
	}
	return pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: in, Amount: bignumber.NewBig(q.AmountIn)},
		TokenOut:      out,
	}
}

func TestCloneState(t *testing.T) {
	t.Parallel()
	v := loadVectors(t)
	for _, sc := range v.Pools[0].Scenarios {
		sim, err := NewPoolSimulator(entityFromSnapshot(t, v.Pools[0], sc.Snapshot))
		require.NoError(t, err)
		sim.quoteAt = int64(sc.NextTimestamp)
		for _, q := range sc.NextQuotes {
			if q.Error == nil && q.TokenIn != "other" {
				testutil.TestCloneState(t, sim, params(q), nil)
				return
			}
		}
	}
	t.Fatal("no fillable quote")
}

// card packs a price card in the contract's layout (card.go).
func card(price, dBid, dAsk, kBid, kAsk, qMax, seq, validUntil, validUntilBlock uint64) *uint256.Int {
	w := new(uint256.Int)
	for _, f := range [][2]uint64{{validUntilBlock, 216}, {price, 152}, {dBid, 136}, {dAsk, 120}, {kBid, 104},
		{kAsk, 88}, {qMax, 64}, {seq, 32}, {validUntil, 0}} {
		w.Or(w, new(uint256.Int).Lsh(uint256.NewInt(f[0]), uint(f[1])))
	}
	return w
}

// readmeInput is the worked example of the contract README (test/ReadmeExample.t.sol): WETH/USDC with PRICE_SCALE
// 1e8 and K_SCALE 4, p = 3000 USDC/WETH, 1 bp half-spreads, k = 0.1 bp per $10k on both sides (field 250). The card
// is valid until timestamp 1000 and block 20; the swap runs at timestamp 999 in block 10.
func readmeInput() QuoteInput {
	ex := Extra{StateWord: card(3000_000000*1e8, 100, 100, 250, 250, 500_000, 1, 1_000, 20),
		FloorBase: new(uint256.Int), FloorQuote: new(uint256.Int),
		MaxNotional: 1 << 40, MaxPerUpdate: 1 << 40, MaxPerWindow: 1<<32 - 1,
		Counters: Counters{CardFilled: new(uint256.Int), BlockFilled: new(uint256.Int), WindowFilled: new(uint256.Int)}}
	return QuoteInput{D: new(uint256.Int).Mul(pow10(18), uint256.NewInt(1e8)), QuoteUnit: pow10(6), KScale: 4,
		Extra: &ex, BalanceOut: new(uint256.Int).Lsh(uint256.NewInt(1), 200), BlockNumber: 10, Now: 999}
}

// TestReadmeExample: the first swap on a fresh card is the single-swap formula: sell 10 WETH -> 29,996.10 USDC;
// pay 100,000 USDC -> 33.326667999733386656 WETH.
func TestReadmeExample(t *testing.T) {
	t.Parallel()
	in := readmeInput()
	in.SellBase, in.AmountIn = true, new(uint256.Int).Mul(uint256.NewInt(10), pow10(18))
	r, err := Quote(in)
	require.NoError(t, err)
	require.Equal(t, "29996100000", r.AmountOut.Dec())
	in.SellBase, in.AmountIn = false, uint256.NewInt(100_000_000000)
	r, err = Quote(in)
	require.NoError(t, err)
	require.Equal(t, "33326667999733386656", r.AmountOut.Dec())
}

// TestSplitNeverBeatsOneSwap: the pool prices a swap on the cumulative fill F, so ten buys of $10k on one card pay out
// no more than one buy of $100k (the difference is rounding, in the pool's favour), in one block or across blocks.
func TestSplitNeverBeatsOneSwap(t *testing.T) {
	t.Parallel()
	for _, sameBlock := range []bool{true, false} {
		in := readmeInput()
		in.SellBase, in.AmountIn = false, uint256.NewInt(100_000_000000)
		one, err := Quote(in)
		require.NoError(t, err)
		total := new(uint256.Int)
		for i := 0; i < 10; i++ {
			in.AmountIn = uint256.NewInt(10_000_000000)
			r, err := Quote(in)
			require.NoError(t, err)
			total.Add(total, r.AmountOut)
			in.Extra.Counters = r.Counters
			if !sameBlock {
				in.BlockNumber++
			}
		}
		require.True(t, total.Cmp(one.AmountOut) <= 0, "split %s > one %s", total, one.AmountOut)
		require.True(t, new(uint256.Int).Sub(one.AmountOut, total).Uint64() < 100, "split %s one %s", total, one.AmountOut)
	}
}

// TestWindowZeroFailsClosed: a per-minute cap of 0 means no trading.
func TestWindowZeroFailsClosed(t *testing.T) {
	t.Parallel()
	in := readmeInput()
	in.Extra.MaxPerWindow = 0
	in.SellBase, in.AmountIn = true, pow10(18)
	_, err := Quote(in)
	require.ErrorIs(t, err, ErrExceedsWindowCap)
}

// TestSideSlippage: a sell is priced with kBid and a buy with kAsk, each times K_SCALE.
func TestSideSlippage(t *testing.T) {
	t.Parallel()
	sell := func(kBid, kAsk, kScale uint64) string {
		in := readmeInput()
		in.Extra.StateWord, in.KScale = card(3000_000000*1e8, 100, 100, kBid, kAsk, 500_000, 1, 1_000, 20), kScale
		in.SellBase, in.AmountIn = true, new(uint256.Int).Mul(uint256.NewInt(10), pow10(18))
		r, err := Quote(in)
		require.NoError(t, err)
		return r.AmountOut.Dec()
	}
	buy := func(kBid, kAsk, kScale uint64) string {
		in := readmeInput()
		in.Extra.StateWord, in.KScale = card(3000_000000*1e8, 100, 100, kBid, kAsk, 500_000, 1, 1_000, 20), kScale
		in.SellBase, in.AmountIn = false, uint256.NewInt(100_000_000000)
		r, err := Quote(in)
		require.NoError(t, err)
		return r.AmountOut.Dec()
	}
	require.Equal(t, "29996100000", sell(250, 9999, 4), "a sell ignores kAsk")
	require.Equal(t, "33326667999733386656", buy(9999, 250, 4), "a buy ignores kBid")
	require.Equal(t, sell(250, 0, 4), sell(1000, 0, 1), "field * K_SCALE is the coefficient")
	require.Equal(t, buy(0, 250, 4), buy(0, 1000, 1))
	require.NotEqual(t, sell(250, 250, 4), sell(251, 250, 4))
	require.NotEqual(t, buy(250, 250, 4), buy(250, 251, 4))
}

// TestBlockExpiry: a card trades only while block.timestamp <= validUntil and block.number <= validUntilBlock (the
// whole block number against the 32-bit field).
func TestBlockExpiry(t *testing.T) {
	t.Parallel()
	in := readmeInput() // valid until timestamp 1000 and block 20
	in.SellBase, in.AmountIn = true, pow10(18)
	for _, c := range []struct {
		block, now uint64
		ok         bool
	}{{20, 1000, true}, {21, 999, false}, {20, 1001, false}, {1<<32 + 20, 999, false}, {0, 0, true}} {
		in.BlockNumber, in.Now = c.block, c.now
		_, err := Quote(in)
		if c.ok {
			require.NoError(t, err, "block %d time %d", c.block, c.now)
		} else {
			require.ErrorIs(t, err, ErrStalePrice, "block %d time %d", c.block, c.now)
		}
	}
}

// TestDarkPool: a pool before its owner's setup (born paused; every cap 0): no liquidity.
func TestDarkPool(t *testing.T) {
	t.Parallel()
	in := readmeInput()
	in.Extra = &Extra{StateWord: new(uint256.Int).Lsh(uint256.NewInt(1), flagPausedBit), FloorBase: new(uint256.Int),
		FloorQuote: new(uint256.Int),
		Counters:   Counters{CardFilled: new(uint256.Int), BlockFilled: new(uint256.Int), WindowFilled: new(uint256.Int)}}
	in.SellBase, in.AmountIn = true, pow10(18)
	_, err := Quote(in)
	require.ErrorIs(t, err, ErrPoolPaused)
}
