package lglclob

import (
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testdata/lens_base.hex is what Lens reverted with for base's WETH/USDC pool
// 0x6a6951db3bb99f76a8ccb9f8a8399b613666298b at block 52279003, measuring, and
// lens_base_book.hex the same without measuring.
func loadBaseState(t *testing.T) []byte {
	t.Helper()
	return loadLensAnswer(t, "testdata/lens_base.hex")
}

func loadLensAnswer(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	data, err := hexutil.Decode(strings.TrimSpace(string(raw)))
	require.NoError(t, err)
	return data
}

func levelsOf(levels ...[2]uint64) OrderBookLevels {
	out := OrderBookLevels{
		ArrayPrices: make([]*uint256.Int, len(levels)),
		ArrayShares: make([]*uint256.Int, len(levels)),
	}
	for i, l := range levels {
		out.ArrayPrices[i], out.ArrayShares[i] = uint256.NewInt(l[0]), uint256.NewInt(l[1])
	}
	return out
}

func TestDecodeState_LiveBase(t *testing.T) {
	t.Parallel()

	state, err := decodeState(loadBaseState(t))
	require.NoError(t, err)
	assert.Equal(t, uint64(52_279_003), state.block)
	assert.True(t, state.hasMaker)
	assert.Equal(t, [4]quote{
		{gas: 229_336, levels: 1},
		{gas: 284_272, levels: 2},
		{gas: 222_006, levels: 1},
		{gas: 276_965, levels: 2},
	}, state.quotes)
	assert.Equal(t, OrderBook{
		Bids: levelsOf([2]uint64{260790, 7669}, [2]uint64{260785, 19172}, [2]uint64{260778, 115040},
			[2]uint64{260767, 230090}, [2]uint64{237299, 40440}),
		Asks: levelsOf([2]uint64{260750, 7670}, [2]uint64{260756, 19175}, [2]uint64{260763, 115046},
			[2]uint64{260774, 205016}, [2]uint64{284242, 22779}),
	}, state.book)
	// The pool has a quoter, which quotes its best bid at 260739, but a sell
	// there fills the book's 260790, so base is not configured as QuoterFills.
	assert.True(t, state.hasQuoter)
	assert.Equal(t, uint64(260739), state.quoterBook().Bids.ArrayPrices[0].Uint64())

	assert.Equal(t, &MakerQuoteGas{
		Bids: &LevelGas{First: 229_336, Next: 54_936},
		Asks: &LevelGas{First: 222_006, Next: 54_959},
	}, state.makerQuoteGas())
}

// Without measuring, Lens reads the same book in place and quotes nothing.
func TestDecodeState_BookOnly(t *testing.T) {
	t.Parallel()

	measured, err := decodeState(loadBaseState(t))
	require.NoError(t, err)
	state, err := decodeState(loadLensAnswer(t, "testdata/lens_base_book.hex"))
	require.NoError(t, err)
	assert.Equal(t, measured.block, state.block)
	assert.Equal(t, measured.book, state.book)
	assert.False(t, state.hasMaker)
	assert.Equal(t, [4]quote{}, state.quotes)
}

// testdata/lens_monad_book.hex is what Lens reverted with for monad's cbBTC/WMON
// pool 0x012e06b56eee881603e47748931dc38efabf54ef at block 125910244, without
// measuring. Its book shows the market maker's bids from 5885320 and a resting
// bid at 1, but the pool's quoter posts the market maker's bids 0.3% lower, from
// 5867660, which is what a sell fills on chain.
func TestDecodeState_QuoterBook(t *testing.T) {
	t.Parallel()

	state, err := decodeState(loadLensAnswer(t, "testdata/lens_monad_book.hex"))
	require.NoError(t, err)
	assert.Equal(t, uint64(125_910_244), state.block)
	assert.True(t, state.hasQuoter)
	assert.Equal(t, uint64(5885320), state.book.Bids.ArrayPrices[0].Uint64(), "the book as the helper shows it")
	assert.Equal(t, OrderBook{
		Bids: levelsOf([2]uint64{5867660, 2177771}, [2]uint64{5867430, 2177856}, [2]uint64{5867310, 6533703},
			[2]uint64{5866960, 9801139}, [2]uint64{5814850, 1098774}, [2]uint64{1, 100000000}),
		Asks: levelsOf([2]uint64{5914230, 3104029}, [2]uint64{5914480, 3104029}, [2]uint64{5914590, 9312089},
			[2]uint64{5914950, 13968134}, [2]uint64{5967460, 1552014}),
	}, state.quoterBook())

	assert.Equal(t, OrderBook{}, (&lensState{}).quoterBook(), "a pool without a quoter keeps its book")
}

func TestFillable(t *testing.T) {
	t.Parallel()

	// The market maker shows 100 at 10 and 50 at 9, and a resting order 7 at 9.
	shown := levelsOf([2]uint64{10, 100}, [2]uint64{9, 57}, [2]uint64{5, 20})
	maker := levelsOf([2]uint64{10, 100}, [2]uint64{9, 50})
	quoter := levelsOf([2]uint64{9, 110}, [2]uint64{8, 55})

	assert.Equal(t, levelsOf([2]uint64{9, 7}, [2]uint64{9, 110}, [2]uint64{8, 55}, [2]uint64{5, 20}),
		fillable(shown, &maker, &quoter, true), "the quoter's levels take the market maker's, resting orders stay")
	assert.Equal(t, levelsOf([2]uint64{9, 7}, [2]uint64{5, 20}), fillable(shown, &maker, nil, true),
		"a side the quoter could not quote has none of the market maker's levels")
	assert.Equal(t, shown, fillable(shown, nil, &quoter, true),
		"a side the market maker could not quote is kept as shown")

	asks := levelsOf([2]uint64{10, 100}, [2]uint64{12, 20})
	askMaker := levelsOf([2]uint64{10, 100})
	askQuoter := levelsOf([2]uint64{11, 100})
	assert.Equal(t, levelsOf([2]uint64{11, 100}, [2]uint64{12, 20}), fillable(asks, &askMaker, &askQuoter, false),
		"asks stay lowest first")
	assert.Equal(t, uint64(100), shown.ArrayShares[0].Uint64(), "the shown book is not changed")
}

// Anything but Lens's own answer must fail, rather than read as a pool, and a
// revert without data is a book Lens could not read.
func TestDecodeState_Rejects(t *testing.T) {
	t.Parallel()

	_, err := decodeState(nil)
	assert.ErrorIs(t, err, ErrBookUnread)

	foreign := loadBaseState(t)
	foreign[0] ^= 0xff
	_, err = decodeState(foreign)
	assert.ErrorIs(t, err, ErrUnexpectedLensRevert)

	data := loadBaseState(t)
	for name, broken := range map[string][]byte{
		"a level cut short": data[:len(data)-1],
		"a byte left over":  append(append([]byte{}, data...), 0),
		"the header alone":  data[:lensHeader],
	} {
		_, err := decodeState(broken)
		assert.ErrorIs(t, err, ErrUnexpectedLensRevert, name)
	}

	// An empty book is followed by whether the pool has a quoter, and a quoter
	// by its four quotes.
	emptyBook := slices.Concat([]byte(lensTag), make([]byte, lensHeader-len(lensTag)+2))
	_, err = decodeState(slices.Concat(emptyBook, []byte{0}))
	assert.NoError(t, err, "a pool without a quoter")
	_, err = decodeState(slices.Concat(emptyBook, []byte{1, quoteFailed, quoteFailed, quoteFailed, quoteFailed}))
	assert.NoError(t, err, "a quoter whose quotes all failed")
	for name, broken := range map[string][]byte{
		"no quoter byte":       emptyBook,
		"an unknown byte":      slices.Concat(emptyBook, []byte{2}),
		"a quoter cut short":   slices.Concat(emptyBook, []byte{1, quoteFailed, quoteFailed, quoteFailed}),
		"a quote left over":    slices.Concat(emptyBook, []byte{1, 0, 0, 0, 0, 0}),
		"a quote level halved": slices.Concat(emptyBook, []byte{1, 1, 0, 0, 0}, make([]byte, lensLevel/2)),
	} {
		_, err := decodeState(broken)
		assert.ErrorIs(t, err, ErrUnexpectedLensRevert, name)
	}
}

func TestLensState_MakerQuoteGas(t *testing.T) {
	t.Parallel()

	assert.Nil(t, (&lensState{}).makerQuoteGas(), "a pool without a market maker")
	assert.Equal(t, &MakerQuoteGas{}, (&lensState{hasMaker: true}).makerQuoteGas(),
		"a market maker without orders to quote")
}

func TestMeasureDue(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_800_000_000, 0)
	at := func(ago time.Duration) int64 { return now.Add(-ago).Unix() }
	measured := &MakerQuoteGas{Bids: &LevelGas{First: 1, Next: 1}, Asks: &LevelGas{First: 1, Next: 1}}
	asksOnly := &MakerQuoteGas{Asks: &LevelGas{First: 1, Next: 1}}
	bids := OrderBook{Bids: levelsOf([2]uint64{1, 1})}

	assert.True(t, measureDue(Extra{}, now), "never measured")
	assert.False(t, measureDue(Extra{MakerQuoteGas: measured, MeasuredAt: at(time.Minute)}, now))
	assert.True(t, measureDue(Extra{MakerQuoteGas: measured, MeasuredAt: at(makerQuoteRefresh)}, now))
	assert.False(t, measureDue(Extra{MeasuredAt: at(time.Minute)}, now), "a pool without a market maker")
	assert.False(t, measureDue(Extra{MakerQuoteGas: asksOnly, MeasuredAt: at(time.Minute)}, now),
		"a side without orders waits for them")
	assert.False(t, measureDue(Extra{OrderBook: bids, MakerQuoteGas: asksOnly, MeasuredAt: at(time.Second)}, now),
		"orders on an unquoted side wait makerQuoteRetry")
	assert.True(t, measureDue(Extra{OrderBook: bids, MakerQuoteGas: asksOnly, MeasuredAt: at(makerQuoteRetry)}, now),
		"orders on an unquoted side are measured then")
}

func TestFitLevelGas(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		quotes []quote
		want   *LevelGas
	}{
		{
			name: "two quotes give the gas each level more adds",
			quotes: []quote{
				{gas: 150_000, levels: 3},
				{gas: 100_000, levels: 1},
			},
			want: &LevelGas{First: 100_000, Next: 25_000},
		},
		{
			name: "quotes past the first level are extended back to it",
			quotes: []quote{
				{gas: 200_000, levels: 2},
				{gas: 260_000, levels: 3},
			},
			want: &LevelGas{First: 140_000, Next: 60_000},
		},
		{
			name: "a single quote takes unmeasuredMakerQuote's for each level more",
			quotes: []quote{
				{gas: 200_000, levels: 2},
			},
			want: &LevelGas{First: 200_000 - unmeasuredMakerQuote.Next, Next: unmeasuredMakerQuote.Next},
		},
		{
			name: "failed and empty quotes are left out",
			quotes: []quote{
				{gas: 0, levels: 0},
				{gas: 15_657, levels: 0},
				{gas: 100_000, levels: 1},
			},
			want: &LevelGas{First: 100_000, Next: unmeasuredMakerQuote.Next},
		},
		{
			name:   "a side without quotes is nil",
			quotes: nil,
			want:   nil,
		},
		{
			name: "a side whose quotes all failed is nil",
			quotes: []quote{
				{gas: 0, levels: 0},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, fitLevelGas(tt.quotes))
		})
	}
}

func TestExtra_JSON(t *testing.T) {
	t.Parallel()

	extra := Extra{
		Bids: OrderBookLevels{
			ArrayPrices: []*uint256.Int{uint256.NewInt(270069)},
			ArrayShares: []*uint256.Int{uint256.NewInt(7405)},
		},
		MakerQuoteGas: &MakerQuoteGas{
			Bids: &LevelGas{First: 231_726, Next: 54_950},
		},
		MeasuredAt: 1_800_000_000,
	}
	data, err := json.Marshal(extra)
	require.NoError(t, err)
	assert.JSONEq(t,
		`{"b":{"p":["270069"],"s":["7405"]},"a":{"p":null,"s":null},"g":{"b":{"f":231726,"n":54950}},"t":1800000000}`,
		string(data))

	var got Extra
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, extra, got)

	// An order book saved before the market maker's quote was measured reads as unmeasured.
	var book Extra
	require.NoError(t, json.Unmarshal([]byte(`{"b":{"p":["1"],"s":["2"]},"a":{"p":[],"s":[]}}`), &book))
	assert.Nil(t, book.MakerQuoteGas)
	assert.Zero(t, book.MeasuredAt)
	assert.True(t, measureDue(book, time.Now()))
	assert.Len(t, book.Bids.ArrayPrices, 1)
}
