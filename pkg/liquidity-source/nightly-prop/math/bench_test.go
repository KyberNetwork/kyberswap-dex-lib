package math

// Benchmarks of the pool math, from the primitives up to the walks. The router calls a walk
// thousands of times per route, so the allocations matter as much as the time.

import (
	"bytes"
	"testing"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
)

// widePool is shaped like the live Arbitrum WETH/USDC pool: 40 levels of 1 bp each, from -20 to 19,
// around ETH at $2,000. Each level below the mid holds 1,000 USDC, each level from the mid up holds
// 0.5 ETH, the position is at the lower edge of level 0, and both oracle spreads are the live 283
// millionths.
func widePool(b testing.TB) (Pool, OraclePrice) {
	levels := make([]Level, 40)
	for i := range levels {
		levelID := i - 20
		levels[i].Config = config(SignedSpread(levelID*100), SignedSpread(levelID*100+100), 0, 0)
		if levelID < 0 {
			levels[i].Liquidity.QuoteAmount = *usdc(b, 1_000)
		} else {
			levels[i].Liquidity.BaseAmount = *scaled(b, 5, 17)
		}
	}
	return Pool{
		BaseDecimals:  18,
		QuoteDecimals: 6,
		MaxPriceAgeMs: 30_000,
		LowestLevelID: -20,
		Levels:        levels,
		Position:      Position{LevelID: 0, Fraction: Lower},
	}, OraclePrice{MidPrice: fixtureMid, BidSpread: 283, AskSpread: 283, TimestampMs: timeMs}
}

// buyAcross returns the quote that drains crossed levels of the wide pool from level 0 up and buys
// half of the next level's base.
func buyAcross(b testing.TB, p Pool, oracle OraclePrice, crossed int) *uint256.Int {
	m := mid(b, oracle.MidPrice, p.BaseDecimals, p.QuoteDecimals)
	total := new(uint256.Int)
	for i := 0; i <= crossed; i++ {
		level := p.Levels[20+i]
		ask := askRange(b, level.Config, m, oracle.AskSpread)
		drain, err := BuyWithExactQuote(&level.Liquidity.BaseAmount, Lower, maxUint256, ask)
		if err != nil {
			b.Fatal(err)
		}
		cost := &drain.QuoteAmount
		if i == crossed {
			cost = new(uint256.Int).Rsh(cost, 1)
		}
		total.Add(total, cost)
	}
	return total
}

// sellAcross returns the base that refills crossed levels of the wide pool from level -1 down and
// sells half of the next level's buy-back.
func sellAcross(b testing.TB, p Pool, oracle OraclePrice, crossed int) *uint256.Int {
	m := mid(b, oracle.MidPrice, p.BaseDecimals, p.QuoteDecimals)
	total := new(uint256.Int)
	for i := 0; i <= crossed; i++ {
		level := p.Levels[19-i]
		bid := bidRange(b, level.Config, m, oracle.BidSpread)
		refill, err := SellExactBase(&level.Liquidity.QuoteAmount, Upper, maxUint256, bid)
		if err != nil {
			b.Fatal(err)
		}
		base := &refill.BaseAmount
		if i == crossed {
			base = new(uint256.Int).Rsh(base, 1)
		}
		total.Add(total, base)
	}
	return total
}

func BenchmarkMulDiv(b *testing.B) {
	// A quote amount from a base amount at a native ETH/USDC price: the product fits 256 bits.
	fits := [3]*uint256.Int{scaled(b, 7, 18), scaled(b, 2_000, 36), scaled(b, 1, 48)}
	// A product of about 2^456 that needs the 512-bit path, and a quotient of about 2^255 that fits.
	wide := [3]*uint256.Int{maxUint256, new(uint256.Int).Lsh(one, 200), new(uint256.Int).Lsh(one, 201)}
	for _, c := range []struct {
		name     string
		operands [3]*uint256.Int
		rounding Rounding
	}{
		{"fits_256_floor", fits, Floor},
		{"fits_256_ceil", fits, Ceil},
		{"needs_512_floor", wide, Floor},
		{"needs_512_ceil", wide, Ceil},
	} {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := MulDiv(c.operands[0], c.operands[1], c.operands[2], c.rounding); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkSqrt(b *testing.B) {
	// The square of a quote value of about 2^100 native units, as buyWithExactQuote takes.
	square := new(uint256.Int).Lsh(one, 200)
	square.AddUint64(square, 12_345)
	b.ReportAllocs()
	for b.Loop() {
		Sqrt(square, Ceil)
	}
}

func BenchmarkFraction_MoveUp(b *testing.B) {
	start := fraction(b, new(uint256.Int).Div(maxRaw, u(3)))
	amount, depth := scaled(b, 3, 18), scaled(b, 10, 18)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := start.MoveUp(amount, depth, Floor); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPriceFromOracle(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := PriceFromOracle(fixtureMid, 18, 6); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAskRange(b *testing.B) {
	m := mid(b, fixtureMid, 18, 6)
	c := config(-1_000, 1_000, 200, 100)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := AskRange(c, m, 283); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPriceRange_PriceAt(b *testing.B) {
	r := askRange(b, config(0, 1_000, 0, 0), mid(b, fixtureMid, 18, 6), 0)
	f := fraction(b, new(uint256.Int).Div(maxRaw, u(3)))
	b.ReportAllocs()
	for b.Loop() {
		if _, err := r.PriceAt(f, Ceil); err != nil {
			b.Fatal(err)
		}
	}
}

// This level holds 10 ETH between 2,000 and 2,002 USDC per ETH.
func BenchmarkLevel_BuyWithExactQuote(b *testing.B) {
	ask := askRange(b, config(0, 1_000, 0, 0), mid(b, fixtureMid, 18, 6), 0)
	depth := scaled(b, 10, 18)
	start := fraction(b, new(uint256.Int).Div(maxRaw, u(3)))
	for _, c := range []struct {
		name   string
		amount *uint256.Int
	}{
		{"inside", u(1_234_567_891)},
		{"drain", usdc(b, 30_000)},
	} {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := BuyWithExactQuote(depth, start, c.amount, ask); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkLevel_SellExactBase: one level from 1,998 to 2,000 with 19,990 USDC.
func BenchmarkLevel_SellExactBase(b *testing.B) {
	bid := bidRange(b, config(-1_000, 0, 0, 0), mid(b, fixtureMid, 18, 6), 0)
	depth := usdc(b, 19_990)
	start := fraction(b, new(uint256.Int).Mul(new(uint256.Int).Div(maxRaw, u(3)), u(2)))
	for _, c := range []struct {
		name   string
		amount *uint256.Int
	}{
		{"inside", dec(b, "1234567890123456789")},
		{"drain", scaled(b, 20, 18)},
	} {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := SellExactBase(depth, start, c.amount, bid); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkPool_BuyWithExactQuote: walks that stay in one level, cross one level, or cross many.
func BenchmarkPool_BuyWithExactQuote(b *testing.B) {
	wide, wideOracle := widePool(b)
	f := funded(b, levelConfigs())
	for _, c := range []struct {
		name   string
		pool   Pool
		oracle OraclePrice
		amount *uint256.Int
	}{
		{"one_level", f.pool, f.oracle, usdc(b, 7_000)},
		{"cross_1_level", f.pool, f.oracle, usdc(b, 25_000)},
		{"wide_cross_5_levels", wide, wideOracle, buyAcross(b, wide, wideOracle, 5)},
		{"wide_cross_19_levels", wide, wideOracle, buyAcross(b, wide, wideOracle, 19)},
	} {
		b.Run(c.name, func(b *testing.B) {
			p := c.pool
			b.ReportAllocs()
			for b.Loop() {
				if _, err := p.BuyWithExactQuote(c.amount, c.oracle, fixtureBlockTime); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkPool_SellExactBase: as BenchmarkPool_BuyWithExactQuote, downward. The wide pool's walk
// first passes level 0, which holds no quote.
func BenchmarkPool_SellExactBase(b *testing.B) {
	wide, wideOracle := widePool(b)
	f := funded(b, levelConfigs())
	crossing := funded(b, levelConfigs())
	crossing.seedLevel(0, ether(b, 5), u(10_002_500_000))
	crossing.seedPosition(0, fraction(b, new(uint256.Int).Rsh(maxRaw, 1)))
	for _, c := range []struct {
		name   string
		pool   Pool
		oracle OraclePrice
		amount *uint256.Int
	}{
		{"one_level", f.pool, f.oracle, ether(b, 5)},
		{"cross_1_level", crossing.pool, crossing.oracle, ether(b, 6)},
		{"wide_cross_5_levels", wide, wideOracle, sellAcross(b, wide, wideOracle, 5)},
		{"wide_cross_19_levels", wide, wideOracle, sellAcross(b, wide, wideOracle, 19)},
	} {
		b.Run(c.name, func(b *testing.B) {
			p := c.pool
			b.ReportAllocs()
			for b.Loop() {
				if _, err := p.SellExactBase(c.amount, c.oracle, fixtureBlockTime); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkPool_Apply: a walk across 19 levels of the wide pool, written into a copy of the pool.
// Apply writes absolute balances, so applying the same swap again leaves the same state.
func BenchmarkPool_Apply(b *testing.B) {
	wide, oracle := widePool(b)
	swap, err := wide.BuyWithExactQuote(buyAcross(b, wide, oracle, 19), oracle, fixtureBlockTime)
	if err != nil {
		b.Fatal(err)
	}
	p := wide
	p.Levels = append([]Level(nil), wide.Levels...)
	b.ReportAllocs()
	for b.Loop() {
		if err := p.Apply(swap); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPool_Decode: the 40-level wide pool from JSON, as NewPoolSimulator reads a record, and
// from msgpack with the options of dex-lib's pkg/msgpack, as a snapshot of simulators is read. The
// test cannot import pkg/msgpack, which imports nightly-prop and so this package.
func BenchmarkPool_Decode(b *testing.B) {
	wide, _ := widePool(b)
	encodedJSON, err := json.Marshal(&wide)
	if err != nil {
		b.Fatal(err)
	}
	var buffer bytes.Buffer
	encoder := msgpack.NewEncoder(&buffer)
	encoder.IncludeUnexported(true)
	encoder.SetForceAsArray(true)
	if err := encoder.Encode(&wide); err != nil {
		b.Fatal(err)
	}
	encodedMsgpack := buffer.Bytes()
	b.Run("json", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var p Pool
			if err := json.Unmarshal(encodedJSON, &p); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("msgpack", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			decoder := msgpack.NewDecoder(bytes.NewReader(encodedMsgpack))
			decoder.IncludeUnexported(true)
			decoder.SetForceAsArray(true)
			var p Pool
			if err := decoder.Decode(&p); err != nil {
				b.Fatal(err)
			}
		}
	})
}
