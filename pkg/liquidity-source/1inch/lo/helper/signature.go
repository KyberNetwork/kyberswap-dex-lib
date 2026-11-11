package helper

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type Sig struct {
	R []byte // 32 bytes
	S []byte // 32 bytes
	V int
}

// compact representation of the `yParity` and `s` compacted into a single bytes32
func (s *Sig) yParityAndSBytes() []byte {
	yParityAndS := make([]byte, len(s.S))
	copy(yParityAndS, s.S)

	if s.yParity() == 1 {
		yParityAndS[0] |= 0x80
	}
	return yParityAndS
}

func (s *Sig) yParity() int {
	if s.V == 27 {
		return 0
	}
	return 1
}

// GetCompactedSignatureBytes returns the 32 bytes of R and 32 bytes of yParityAndS (or VS) concatenated
func (s *Sig) GetCompactedSignatureBytes() []byte {
	return append(s.R, s.yParityAndSBytes()...)
}

// LO1inchParseSignature https://github.com/ethers-io/ethers.js/blob/main/src.ts/crypto/signature.ts#L284
func LO1inchParseSignature(signature string) (*Sig, error) {
	sigBytes := common.FromHex(signature)
	if len(sigBytes) == 64 {
		r := sigBytes[:32]
		s := sigBytes[32:64]
		v := 27
		if s[0]&0x80 != 0 {
			v = 28
		}
		s[0] &= 0x7f
		return &Sig{R: r, S: s, V: v}, nil
	}

	if len(sigBytes) == 65 {
		r := sigBytes[0:32]
		s := sigBytes[32:64]
		if (s[0] & 0x80) != 0 {
			return nil, errors.New("non-canonical s")
		}
		v, err := getNormalizedV(sigBytes[64])
		if err != nil {
			return nil, err
		}
		return &Sig{R: r, S: s, V: v}, nil
	}

	return nil, errors.New("invalid raw signature length")
}

// https://github.com/ethers-io/ethers.js/blob/main/src.ts/crypto/signature.ts#L255
func getNormalizedV(v byte) (int, error) {
	switch v {
	case 0, 27:
		return 27, nil
	case 1, 28:
		return 28, nil
	}

	if v < 35 {
		return 0, fmt.Errorf("invalid signature v: %d", v)
	}

	return 28 - int(v&1), nil
}
