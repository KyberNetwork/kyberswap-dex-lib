package msgpencode

import "math/big"

const (
	negSign = 255
)

func EncodeInt(x *big.Int) []byte {
	if x == nil {
		return nil
	}

	b := make([]byte, 1 /* sign */ +(len(x.Bits())*8) /* words */)
	x.FillBytes(b[1:])
	if x.Sign() < 0 {
		b[0] = negSign
	} else {
		b[0] = 0
	}
	return b
}

func DecodeInt(b []byte) *big.Int {
	if b == nil {
		return nil
	}

	z := new(big.Int)
	z.SetBytes(b[1:])
	if b[0] == negSign {
		z.Neg(z)
	}
	return z
}
