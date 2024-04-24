package msgpencode

import (
	"math/big"
	"testing"
)

var intEncodingTests = []string{
	"0",
	"1",
	"2",
	"10",
	"1000",
	"1234567890",
	"298472983472983471903246121093472394872319615612417471234712061",
}

func TestIntMsgpEncoding(t *testing.T) {
	for _, test := range intEncodingTests {
		for _, sign := range []string{"", "+", "-"} {
			x := sign + test
			tx, _ := new(big.Int).SetString(x, 10)
			b := EncodeInt(tx)
			rx := DecodeInt(b)
			if rx.Cmp(tx) != 0 {
				t.Errorf("messagepack encoding of %s failed: got %s want %s", tx, rx, tx)
			}
		}
	}
}
