package bignumber

import (
	"encoding/base64"
	"math/big"
)

// GobBigInt is a wrapper around big.Int that use Gob for json marshal/unmarshal
type GobBigInt big.Int

// almost all of our bigInt are for (u)int256, so should be only 33 bytes (32 for abs and 1 for version & sign)
const MaxBigIntGobLength = 64

func (f *GobBigInt) ToBig() *big.Int {
	return (*big.Int)(f)
}

func (f *GobBigInt) MarshalText() ([]byte, error) {
	bi := f.ToBig()
	b, err := bi.GobEncode()
	if err != nil {
		return nil, err
	}
	return []byte(base64.StdEncoding.EncodeToString(b)), nil
}

func (f *GobBigInt) UnmarshalText(input []byte) error {
	bi := f.ToBig()
	enc := base64.StdEncoding
	b64Len := len(input)
	bLen := enc.DecodedLen(b64Len)

	// if it's small enough then allocate temp store on stack to avoid GC
	if bLen < MaxBigIntGobLength {
		var b [MaxBigIntGobLength]byte
		n, err := base64.StdEncoding.Decode(b[:], input)
		if err != nil {
			return err
		}
		err = bi.GobDecode(b[:n])
		return err
	}

	// otherwise fall back
	b := make([]byte, bLen)
	n, err := base64.StdEncoding.Decode(b, input)
	if err != nil {
		return err
	}
	return bi.GobDecode(b[:n])
}
