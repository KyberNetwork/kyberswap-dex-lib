package msgpencode

import (
	"math/big"
	"testing"
)

var floatVals = []string{
	"0",
	"1",
	"0.1",
	"2.71828",
	"1234567890",
	"3.14e1234",
	"3.14e-1234",
	"0.738957395793475734757349579759957975985497e100",
	"0.73895739579347546656564656573475734957975995797598589749859834759476745986795497e100",
	"inf",
	"Inf",
}

func TestFloatMsgpEncoding(t *testing.T) {
	for _, test := range floatVals {
		for _, sign := range []string{"", "+", "-"} {
			for _, prec := range []uint{0, 1, 2, 10, 53, 64, 100, 1000} {
				for _, mode := range []big.RoundingMode{big.ToNearestEven, big.ToNearestAway, big.ToZero, big.AwayFromZero, big.ToNegativeInf, big.ToPositiveInf} {
					x := sign + test

					tx := new(big.Float)
					_, _, err := tx.SetPrec(prec).SetMode(mode).Parse(x, 0)
					if err != nil {
						t.Errorf("parsing of %s (%dbits, %v) failed (invalid test case): %v", x, prec, mode, err)
						continue
					}

					// If tx was set to prec == 0, tx.Parse(x, 0) assumes precision 64. Correct it.
					if prec == 0 {
						tx.SetPrec(0)
					}

					buf := EncodeFloat(tx)

					rx := DecodeFloat(buf)

					if rx.Cmp(tx) != 0 {
						t.Errorf("transmission of %s failed: got %s want %s", x, rx.String(), tx.String())
						continue
					}

					if rx.Prec() != prec {
						t.Errorf("transmission of %s's prec failed: got %d want %d", x, rx.Prec(), prec)
					}

					if rx.Mode() != mode {
						t.Errorf("transmission of %s's mode failed: got %s want %s", x, rx.Mode(), mode)
					}

					if rx.Acc() != tx.Acc() {
						t.Errorf("transmission of %s's accuracy failed: got %s want %s", x, rx.Acc(), tx.Acc())
					}
				}
			}
		}
	}
}
