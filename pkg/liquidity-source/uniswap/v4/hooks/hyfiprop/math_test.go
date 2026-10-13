package hyfiprop

import (
	"os"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

type fixtureBook struct {
	Name string    `json:"name"`
	Bid  [3]string `json:"bid"`
	Ask  [3]string `json:"ask"`
}

type fixture struct {
	Block     uint64 `json:"block"`
	Timestamp uint64 `json:"timestamp"`
	Cfg       struct {
		TickWidth    string `json:"tickWidth"`
		BaseLiqUnit  string `json:"baseLiqUnit"`
		FeePerSecond uint32 `json:"feePerSecond"`
		BaseIsC0     bool   `json:"baseIsCurrency0"`
	} `json:"cfg"`
	Books []fixtureBook `json:"books"`
	Cases []struct {
		Book       string  `json:"book"`
		ZeroForOne bool    `json:"zeroForOne"`
		In         string  `json:"in"`
		Out        *string `json:"out"` // nil: quoteDirect reverted
		Fee        *string `json:"fee"`
	} `json:"cases"`
}

func u(s string) uint256.Int { return *uint256.MustFromDecimal(s) }

func hexSide(t testing.TB, w [3]string) (s Side) {
	t.Helper()
	for i, p := range []*uint256.Int{&s.Slot0, &s.WordA, &s.WordB} {
		require.NoError(t, p.SetFromHex(w[i]))
	}
	return s
}

func loadFixture(t testing.TB) *fixture {
	t.Helper()
	raw, err := os.ReadFile("testdata/quote_direct.json")
	require.NoError(t, err)
	var f fixture
	require.NoError(t, json.Unmarshal(raw, &f))
	return &f
}

func (f *fixture) cfg() PairConfig {
	return PairConfig{TickWidth: u(f.Cfg.TickWidth), BaseLiqUnit: u(f.Cfg.BaseLiqUnit),
		FeePerSecond: f.Cfg.FeePerSecond, BaseIsC0: f.Cfg.BaseIsC0}
}

// Every case is an eth_call of HyFi.quoteDirect on Base (block in the fixture) against the live book or
// a book injected with state overrides: multi-tick walks, a pre-set walk pointer, the 100% fee cap and
// sizes past the book's depth (revert).
func TestPriceExactIn_MatchesQuoteDirect(t *testing.T) {
	f := loadFixture(t)
	cfg := f.cfg()
	books := map[string]*[2]Side{}
	for _, b := range f.Books {
		books[b.Name] = &[2]Side{hexSide(t, b.Bid), hexSide(t, b.Ask)}
	}
	var reverts int
	for _, c := range f.Cases {
		in := u(c.In)
		res, err := priceExactIn(books[c.Book], &cfg, c.ZeroForOne, &in, f.Timestamp)
		if c.Out == nil {
			reverts++
			require.ErrorIs(t, err, ErrInsufficientLiquidity, "%s zfo=%v in=%s", c.Book, c.ZeroForOne, c.In)
			continue
		}
		require.NoError(t, err, "%s zfo=%v in=%s", c.Book, c.ZeroForOne, c.In)
		require.Equal(t, *c.Out, res.amountOut.Dec(), "out: %s zfo=%v in=%s", c.Book, c.ZeroForOne, c.In)
		require.Equal(t, *c.Fee, res.fee.Dec(), "fee: %s zfo=%v in=%s", c.Book, c.ZeroForOne, c.In)
	}
	require.NotZero(t, reverts)
}
