package ilyris

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

func testEntity(t *testing.T, ex Extra) entity.Pool {
	t.Helper()
	return entity.Pool{
		Address:  "0x90D0950065C567B9324A08A9AaE8a28890fBab16", // checksummed on purpose
		Exchange: DexType,
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: "0x0Bd7D308f8E1639FAb988df18A8011f41EAcAD73", Decimals: 18, Swappable: true},
			{Address: "0x5fc5360D0400a0Fd4f2af552ADD042D716F1d168", Decimals: 6, Swappable: true},
		},
		StaticExtra: mustJSON(t, StaticExtra{BinStepBps: 10, DecimalsX: 18, DecimalsY: 6}),
		Extra:       mustJSON(t, ex),
		BlockNumber: 43307616,
	}
}

func liveLikeExtra() Extra {
	return Extra{
		ActiveID:     7796,
		TotalFeeRate: 3_000_000,
		Bins: []BinJSON{
			{ID: 7795, ReserveX: "0", ReserveY: "500000000"},
			{ID: 7796, ReserveX: "1000000000000000000", ReserveY: "500000000"},
			{ID: 7797, ReserveX: "1000000000000000000", ReserveY: "0"},
		},
	}
}

func TestNewPoolSimulatorFromEntity(t *testing.T) {
	s, err := NewPoolSimulator(testEntity(t, liveLikeExtra()))
	if err != nil {
		t.Fatalf("construct: %v", err)
	}
	// Addresses must be folded to lower case at construction, because their GetTokenIndex
	// compares with == and our manifest is checksummed.
	for _, tok := range s.Info.Tokens {
		for _, r := range tok {
			if r >= 'A' && r <= 'Z' {
				t.Fatalf("token address not lowercased: %s", tok)
			}
		}
	}
	if s.Info.Address != "0x90d0950065c567b9324a08a9aae8a28890fbab16" {
		t.Fatalf("pool address not lowercased: %s", s.Info.Address)
	}
	// Info.Reserves is the SUM of bin reserves -- that is what their coarse ranking reads.
	wantX, _ := new(big.Int).SetString("2000000000000000000", 10)
	if s.Info.Reserves[0].Cmp(wantX) != 0 {
		t.Fatalf("X reserve sum wrong: %s", s.Info.Reserves[0])
	}
	if s.Info.Reserves[1].Cmp(big.NewInt(1_000_000_000)) != 0 {
		t.Fatalf("Y reserve sum wrong: %s", s.Info.Reserves[1])
	}
	var _ pool.IPoolSimulator = s
}

// Reserves are uint128 on chain. JSON numbers are float64, which loses precision above 2^53 --
// so they travel as decimal strings. This asserts a value that WOULD be corrupted by a float
// survives the round trip, because the corruption is silent and produces a wrong quote rather
// than an error.
func TestLargeReservesSurviveTheWire(t *testing.T) {
	huge := "340282366920938463463374607431768211455" // max uint128
	ex := liveLikeExtra()
	ex.Bins[1].ReserveX = huge
	s, err := NewPoolSimulator(testEntity(t, ex))
	if err != nil {
		t.Fatalf("construct: %v", err)
	}
	want, _ := new(big.Int).SetString(huge, 10)
	found := false
	for _, b := range s.bins {
		if b.ReserveX.Cmp(want) == 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("max-uint128 reserve did not survive decoding")
	}
}

// A pool with nothing in it must REFUSE to build. A simulator over an empty book quotes zero,
// and a zero is routed as a genuine offer of nothing rather than as "skip this pool".
func TestEmptyBookIsRefused(t *testing.T) {
	ex := liveLikeExtra()
	for i := range ex.Bins {
		ex.Bins[i].ReserveX, ex.Bins[i].ReserveY = "0", "0"
	}
	if _, err := NewPoolSimulator(testEntity(t, ex)); err != ErrEmptyBook {
		t.Fatalf("expected ErrEmptyBook, got %v", err)
	}
}

// A zero bin step makes price(id) = 1 at every id -- every bin the same price, which is not a
// pool. Refuse rather than quote from it.
func TestZeroBinStepIsRefused(t *testing.T) {
	ep := testEntity(t, liveLikeExtra())
	ep.StaticExtra = mustJSON(t, StaticExtra{BinStepBps: 0, DecimalsX: 18, DecimalsY: 6})
	if _, err := NewPoolSimulator(ep); err != ErrMalformedExtra {
		t.Fatalf("expected ErrMalformedExtra, got %v", err)
	}
}

// Bins must end up ascending regardless of payload order: the traversal walks outward from the
// active bin assuming that, and an out-of-order book would quote wrong WITHOUT erroring.
func TestBinsAreSortedRegardlessOfPayloadOrder(t *testing.T) {
	ex := liveLikeExtra()
	ex.Bins[0], ex.Bins[2] = ex.Bins[2], ex.Bins[0]
	s, err := NewPoolSimulator(testEntity(t, ex))
	if err != nil {
		t.Fatalf("construct: %v", err)
	}
	for i := 1; i < len(s.bins); i++ {
		if s.bins[i-1].ID >= s.bins[i].ID {
			t.Fatalf("bins not ascending at %d: %d then %d", i, s.bins[i-1].ID, s.bins[i].ID)
		}
	}
}

func TestGuardStateSurvivesConstruction(t *testing.T) {
	ex := liveLikeExtra()
	ex.GuardSwapsPaused = true
	s, err := NewPoolSimulator(testEntity(t, ex))
	if err != nil {
		t.Fatalf("construct: %v", err)
	}
	if _, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: s.Info.Tokens[1], Amount: big.NewInt(1_000_000)},
		TokenOut:      s.Info.Tokens[0],
	}); err != ErrSwapsPaused {
		t.Fatalf("guard state lost in construction, got %v", err)
	}
}

func TestFactoryDecodesPoolCreated(t *testing.T) {
	f := NewPoolFactory(DexType)
	topic := common.HexToHash(poolCreatedTopic)
	if !f.IsEventSupported(topic) {
		t.Fatal("own topic not recognised")
	}
	if f.IsEventSupported(common.HexToHash("0xdead")) {
		t.Fatal("unrelated topic accepted")
	}

	ev := types.Log{
		Address: common.HexToAddress("0x4A943A11a6fFBF8D204Df4d5A080Ca741697ca33"),
		Topics: []common.Hash{
			topic,
			common.HexToHash("0x0000000000000000000000000bd7d308f8e1639fab988df18a8011f41eacad73"),
			common.HexToHash("0x0000000000000000000000005fc5360d0400a0fd4f2af552add042d716f1d168"),
			common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000000a"),
		},
		BlockNumber: 43307616,
	}
	p, err := f.DecodePoolCreated(ev)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if p.Tokens[0].Address != "0x0bd7d308f8e1639fab988df18a8011f41eacad73" {
		t.Fatalf("tokenX wrong: %s", p.Tokens[0].Address)
	}
	if p.Tokens[1].Address != "0x5fc5360d0400a0fd4f2af552add042d716f1d168" {
		t.Fatalf("tokenY wrong: %s", p.Tokens[1].Address)
	}
}

// A truncated log must error rather than decode garbage into a plausible-looking pool.
func TestFactoryRejectsShortLog(t *testing.T) {
	f := NewPoolFactory(DexType)
	ev := types.Log{Topics: []common.Hash{common.HexToHash(poolCreatedTopic)}}
	if _, err := f.DecodePoolCreated(ev); err == nil {
		t.Fatal("expected an error for a log missing its indexed fields")
	}
}

func TestNilClientConstructorsDoNotPanic(t *testing.T) {
	// factory_test constructs trackers/listers with a nil ethrpc client.
	tr := NewPoolTrackerFromConfig(nil, nil)
	if tr == nil {
		t.Fatal("nil-config tracker constructor returned nil")
	}
	u := newPoolsListUpdaterFromConfig(nil, nil)
	if u == nil {
		t.Fatal("nil-config lister constructor returned nil")
	}
	f := newPoolFactoryFromConfig(nil)
	if f == nil {
		t.Fatal("nil-config factory constructor returned nil")
	}
}
