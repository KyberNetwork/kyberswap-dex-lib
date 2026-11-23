package ammalgam

import (
	"errors"
	"math/rand"
	"testing"

	"github.com/holiman/uint256"
)

func mathInt(s string) *uint256.Int { return uint256.MustFromDecimal(s) }

// These vectors are literal outputs from core-v1 v1.1.1. They cover the new
// N=40 quadratic curve and both strict quadratic-to-linear boundaries.
func TestV111QuadraticFeeVectors(t *testing.T) {
	tests := []struct{ name, input, current, reference, want string }{
		{"non-crossing quadratic boundary", "1000", "1000", "1000", "0xfa00000000000000000"},
		{"non-crossing first linear unit", "1001", "1000", "1000", "0xfa3fefa1e2d85c125a0"},
		{"crossing quadratic boundary", "1500", "500", "1000", "0xa6aaaaaaaaaaaaaaaaa"},
		{"crossing first linear unit", "1501", "500", "1000", "0xa6e38485080417e1425"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := calculateSwapFeeBipsQ64(mathInt(tc.input), mathInt(tc.current), mathInt(tc.reference))
			if err != nil {
				t.Fatal(err)
			}
			want := uint256.MustFromHex(tc.want)
			if !got.Eq(want) {
				t.Fatalf("fee = %s, want %s", got.Dec(), want.Dec())
			}
		})
	}
}

func TestV111ExactInUsesCeilFeeBeforeDepletion(t *testing.T) {
	amountOut, fee, err := quoteExactInWithFee(
		mathInt("4"), mathInt("96"), mathInt("1000"), mathInt("100"), mathInt("95"), mathInt("0"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !amountOut.Eq(mathInt("75")) {
		t.Fatalf("out = %s, want 75", amountOut.Dec())
	}
	if !fee.Eq(mathInt("1")) {
		t.Fatalf("effective fee = %s, want 1", fee.Dec())
	}
}

func TestV111ExactOutBoundaryVectorsAreMinimum(t *testing.T) {
	tests := []struct {
		name, target, reserve, reference, reserveOut, missingIn, missingOut, want string
	}{
		{"equal-reserve quadratic boundary", "375000000000000000000", "1000000000000000000000", "1000000000000000000000", "1000000000000000000000", "0", "0", "1000000000000000000000"},
		{"equal-reserve first linear output", "375000000000000000001", "1000000000000000000000", "1000000000000000000000", "1000000000000000000000", "0", "0", "1000000000000000000042"},
		{"crossing quadratic boundary", "687500000000000000000", "500000000000000000000", "1000000000000000000000", "1000000000000000000000", "0", "0", "1500000000000000000000"},
		{"crossing first linear output", "687500000000000000001", "500000000000000000000", "1000000000000000000000", "1000000000000000000000", "0", "0", "1500000000000000000083"},
		{"depleted", "10000000000000000000", "1000000000000000000000", "1000000000000000000000", "1000000000000000000000", "960000000000000000000", "951000000000000000000", "10298836671121632293"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			target := mathInt(tc.target)
			reserve, reference, reserveOut := mathInt(tc.reserve), mathInt(tc.reference), mathInt(tc.reserveOut)
			missingIn, missingOut := mathInt(tc.missingIn), mathInt(tc.missingOut)
			got, err := quoteExactOut(target, reserve, reference, reserveOut, missingIn, missingOut)
			if err != nil {
				t.Fatal(err)
			}
			if !got.Eq(mathInt(tc.want)) {
				t.Fatalf("input = %s, want %s", got.Dec(), tc.want)
			}
			atMinimum, err := quoteExactIn(&got, reserve, reference, reserveOut, missingIn, missingOut)
			if err != nil || atMinimum.Lt(target) {
				t.Fatalf("quote(input) = %s, target %s: %v", atMinimum.Dec(), target.Dec(), err)
			}
			var previousInput uint256.Int
			previousInput.SubUint64(&got, 1)
			previous, err := quoteExactIn(&previousInput, reserve, reference, reserveOut, missingIn, missingOut)
			if err != nil || !previous.Lt(target) {
				t.Fatalf("quote(input-1) = %s, target %s: %v", previous.Dec(), target.Dec(), err)
			}
		})
	}
}

func TestQuoteExactInTaggedAndDepletedThreshold(t *testing.T) {
	for _, tc := range []struct{ name, input, reserveIn, reserveOut, missingIn, missingOut, want string }{
		{"ordinary", "1000000000000000000", "1000000000000000000000", "1000000000000000000000", "0", "0", "998601797643075981"},
		{"input at threshold", "10", "100", "100", "95", "0", "8"},
		{"input above threshold", "10", "100", "100", "96", "0", "26"},
		{"output at threshold", "100", "100", "100", "0", "95", "1"},
		{"output above threshold", "100", "100", "100", "0", "96", "1"},
		{"zero", "0", "100", "100", "0", "0", "0"},
		{"one wei floors to zero", "1", "100", "100", "0", "0", "0"},
		{"two wei", "2", "100", "100", "0", "0", "0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input, ri, ro, mi, mo := mathInt(tc.input), mathInt(tc.reserveIn), mathInt(tc.reserveOut), mathInt(tc.missingIn), mathInt(tc.missingOut)
			before := [5]uint256.Int{*input, *ri, *ro, *mi, *mo}
			got, err := quoteExactIn(input, ri, ri, ro, mi, mo)
			if err != nil {
				t.Fatal(err)
			}
			if !got.Eq(mathInt(tc.want)) {
				t.Fatalf("out = %s, want %s", got.Dec(), tc.want)
			}
			if before != [5]uint256.Int{*input, *ri, *ro, *mi, *mo} {
				t.Fatal("mutated inputs")
			}
		})
	}
}

func TestExactOutputIsMinimumInput(t *testing.T) {
	for _, tc := range []struct {
		name, reserve, reference, reserveOut, missingIn, missingOut string
		targets                                                     []string
	}{
		{"minimum", "90000000000000000000", "100000000000000000000", "100000000000000000000", "0", "0", []string{"0", "1", "1000000000000000"}},
		{"quadratic and linear", "1000000000000000000000", "1000000000000000000000", "1000000000000000000000", "0", "0", []string{"1", "996006981039903216", "500000000000000000000", "900000000000000000000"}},
		{"crossing", "50000000000000000000", "100000000000000000000", "100000000000000000000", "0", "0", []string{"60000000000000000000", "90000000000000000000"}},
		{"depleted", "1000000000000000000000", "1000000000000000000000", "1000000000000000000000", "960000000000000000000", "951000000000000000000", []string{"1", "10000000000000000000", "48000000000000000000"}},
		{"tiny reserves", "100", "100", "100", "0", "96", []string{"1", "3"}},
	} {
		for _, target := range tc.targets {
			t.Run(tc.name+"/"+target, func(t *testing.T) {
				ri, ref, ro, mi, mo := mathInt(tc.reserve), mathInt(tc.reference), mathInt(tc.reserveOut), mathInt(tc.missingIn), mathInt(tc.missingOut)
				in, err := quoteExactOut(mathInt(target), ri, ref, ro, mi, mo)
				if err != nil {
					t.Fatal(err)
				}
				out, err := quoteExactIn(&in, ri, ref, ro, mi, mo)
				if err != nil || out.Lt(mathInt(target)) {
					t.Fatalf("quote(in) = %s, target %s: %v", out.Dec(), target, err)
				}
				if in.IsZero() {
					if target != "0" {
						t.Fatal("zero input")
					}
					return
				}
				var less uint256.Int
				less.SubUint64(&in, 1)
				previous, err := quoteExactIn(&less, ri, ref, ro, mi, mo)
				if err != nil || !previous.Lt(mathInt(target)) {
					t.Fatalf("quote(in-1) = %s, target %s: %v", previous.Dec(), target, err)
				}
			})
		}
	}
}

func TestTaggedExactOutputRoundTripAndDomainChecks(t *testing.T) {
	// Parameters are a historical depleted-pool fixture from core-v1's
	// ComputeExpectedInAmountFuzzTest.fixtureRoundtripCase(). That Solidity test
	// checks exact-output round trips and minimality rather than fixed outputs.
	ri := mathInt("182813970261590293")
	ref := mathInt("510079171498268351")
	ro := mathInt("8419899061741814363365")
	mi := mathInt("57293463935898495")
	mo := mathInt("444316374309153650")
	input := mathInt("18058708849222522100325")
	out, err := quoteExactIn(input, ri, ref, ro, mi, mo)
	if err != nil {
		t.Fatal(err)
	}
	got, err := quoteExactOut(&out, ri, ref, ro, mi, mo)
	if err != nil {
		t.Fatal(err)
	}
	quoted, err := quoteExactIn(&got, ri, ref, ro, mi, mo)
	if err != nil || !quoted.Eq(&out) {
		t.Fatalf("tagged round trip output = %s, want %s: %v", quoted.Dec(), out.Dec(), err)
	}
	if got.Sign() > 0 {
		var oneLess uint256.Int
		oneLess.SubUint64(&got, 1)
		previous, err := quoteExactIn(&oneLess, ri, ref, ro, mi, mo)
		if err != nil || !previous.Lt(&out) {
			t.Fatalf("tagged round trip was not minimal: previous output %s target %s: %v", previous.Dec(), out.Dec(), err)
		}
	}

	if _, err := quoteExactOut(mathInt("1"), ri, ref, ro, ri, mo); !errors.Is(err, errMissingGteActual) {
		t.Fatalf("exact-out must propagate invalid missing input: %v", err)
	}
}

func TestExactOutputRespectsUint112PostSwapReserveCap(t *testing.T) {
	maxReserve := uint256.MustFromHex("0xffffffffffffffffffffffffffff")
	reserveIn := new(uint256.Int).Sub(maxReserve, uint256.NewInt(1))
	if _, err := quoteExactOut(mathInt("2"), reserveIn, reserveIn, maxReserve, mathInt("0"), mathInt("0")); !errors.Is(err, errMathOverflow) {
		t.Fatalf("exact-out requiring input beyond uint112 reserve cap: %v", err)
	}
}

func TestExactOutputRandomizedMinimality(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 1_000; i++ {
		reserveIn := uint256.NewInt(uint64(rng.Int63n(1_000_000_000_000) + 100))
		reserveOut := uint256.NewInt(uint64(rng.Int63n(1_000_000_000_000) + 100))
		reference := uint256.NewInt(uint64(rng.Int63n(2_000_000_000_000) + 1))
		missingIn := uint256.NewInt(uint64(rng.Int63n(int64(reserveIn.Uint64() * 98 / 100))))
		missingOut := uint256.NewInt(uint64(rng.Int63n(int64(reserveOut.Uint64() * 98 / 100))))
		input := uint256.NewInt(uint64(rng.Int63n(2_000_000_000_000) + 1))

		target, err := quoteExactIn(input, reserveIn, reference, reserveOut, missingIn, missingOut)
		if err != nil || target.IsZero() {
			continue
		}
		got, err := quoteExactOut(&target, reserveIn, reference, reserveOut, missingIn, missingOut)
		if err != nil {
			t.Fatalf("case %d: exact-out failed: %v", i, err)
		}
		out, err := quoteExactIn(&got, reserveIn, reference, reserveOut, missingIn, missingOut)
		if err != nil || out.Lt(&target) {
			t.Fatalf("case %d: output %s below target %s: %v", i, out.Dec(), target.Dec(), err)
		}
		if !got.IsZero() {
			var previousInput uint256.Int
			previousInput.SubUint64(&got, 1)
			previous, err := quoteExactIn(&previousInput, reserveIn, reference, reserveOut, missingIn, missingOut)
			if err != nil || !previous.Lt(&target) {
				t.Fatalf("case %d: input %s is not minimal; previous output %s, target %s: %v",
					i, got.Dec(), previous.Dec(), target.Dec(), err)
			}
		}
	}
}

func TestMathErrorsMatchCheckedSolidity(t *testing.T) {
	zero, one, hundred := mathInt("0"), mathInt("1"), mathInt("100")
	if _, err := quoteExactIn(one, hundred, hundred, hundred, hundred, zero); !errors.Is(err, errMissingGteActual) {
		t.Fatalf("input missing == reserve: %v", err)
	}
	if _, err := quoteExactIn(one, hundred, hundred, hundred, mathInt("101"), zero); !errors.Is(err, errMathOverflow) {
		t.Fatalf("input missing above reserve underflows fee-adjusted balance: %v", err)
	}
	for _, missing := range []*uint256.Int{hundred, mathInt("101")} {
		if _, err := quoteExactIn(one, hundred, hundred, hundred, zero, missing); !errors.Is(err, errMissingGteActual) {
			t.Fatalf("output missing >= reserve: %v", err)
		}
		if _, err := quoteExactOut(one, hundred, hundred, hundred, zero, missing); !errors.Is(err, errMissingOutGteReserveOut) {
			t.Fatalf("exact-out unavailable: %v", err)
		}
	}
	if _, err := quoteExactOut(mathInt("4"), hundred, hundred, hundred, zero, mathInt("96")); !errors.Is(err, errSwapAmountOutExceedsAvailable) {
		t.Fatalf("available boundary: %v", err)
	}
	max := uint256.MustFromHex("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	if _, err := quoteExactIn(one, max, max, hundred, zero, zero); !errors.Is(err, errMathOverflow) {
		t.Fatalf("reserve addition overflow: %v", err)
	}
	// This product overflows although its quotient would fit: full-precision mulDiv is wrong here.
	large := uint256.MustFromHex("0x100000000000000000000000000000000000000000000000000")
	if _, err := calculateSwapFeeBipsQ64(large, large, large); !errors.Is(err, errMathOverflow) {
		t.Fatalf("checked fee multiplication: %v", err)
	}
	if _, err := quoteExactIn(one, large, large, large, zero, zero); !errors.Is(err, errMathOverflow) {
		t.Fatalf("checked balance multiplication: %v", err)
	}
	fee, err := calculateSwapFeeBipsQ64(one, zero, zero)
	if err != nil || !fee.Eq(uint256.MustFromHex("0x1f400000000000000000")) {
		t.Fatalf("zero reference follows the Solidity linear branch: fee=%s err=%v", fee.Dec(), err)
	}
}
