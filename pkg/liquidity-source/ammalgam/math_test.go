package ammalgam

import (
	"errors"
	"testing"

	"github.com/holiman/uint256"
)

func mathInt(s string) *uint256.Int { return uint256.MustFromDecimal(s) }

// These vectors come from core-v1 v1.1.0 QuadraticSwapFeesTests.sol.
// Changing a fee region, its boundary comparison, or Q64 rounding must fail them.
func TestTaggedQuadraticFees(t *testing.T) {
	tests := []struct{ name, input, current, reference, want string }{
		{"zero", "0", "9000000000000000000", "9000000000000000000", "0x0"},
		{"100 bips", "90000000000000000", "9000000000000000000", "9000000000000000000", "0x140000000000000000"},
		{"150 bips", "135000000000000000", "9000000000000000000", "9000000000000000000", "0x1e0000000000000000"},
		{"quadratic boundary", "18000000000000000000", "9000000000000000000", "9000000000000000000", "0xfa00000000000000000"},
		{"linear", "24000000000000000000", "9000000000000000000", "9000000000000000000", "0x13880000000000000000"},
		{"already deviated", "90000000000000000", "9090000000000000000", "9000000000000000000", "0x3c0000000000000000"},
		{"already deviated 150", "135000000000000000", "9090000000000000000", "9000000000000000000", "0x460000000000000000"},
		{"deviated boundary", "17820000000000000000", "9090000000000000000", "9000000000000000000", "0xfa00000000000000000"},
		{"deviated linear", "23820000000000000000", "9090000000000000000", "9000000000000000000", "0x13880000000000000000"},
		{"toward reference", "10000000000000000000", "90000000000000000000", "100000000000000000000", "0x1999999999999999"},
		{"one wei crossing", "10000000000000000001", "90000000000000000000", "100000000000000000000", "0x1999999999999999"},
		{"crossing quadratic boundary", "250000000000000000000", "50000000000000000000", "100000000000000000000", "0xc800000000000000000"},
		{"crossing linear", "300000000000000000000", "50000000000000000000", "100000000000000000000", "0xfa00000000000000a6a"},
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

func TestQuoteExactInTaggedAndDepletedThreshold(t *testing.T) {
	for _, tc := range []struct{ name, input, reserveIn, reserveOut, missingIn, missingOut, want string }{
		{"ordinary", "1000000000000000000", "1000000000000000000000", "1000000000000000000000", "0", "0", "998801398361917754"},
		{"input at threshold", "10", "100", "100", "95", "0", "8"},
		{"input above threshold", "10", "100", "100", "96", "0", "26"},
		{"output at threshold", "100", "100", "100", "0", "95", "2"},
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
