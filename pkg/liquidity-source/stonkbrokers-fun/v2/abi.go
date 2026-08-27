package stonkbrokersfunv2

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// PadABI / lensABI are parsed directly from Blockscout's verified-source ABI
// for the reference pad (0xFCd61B25BbF3AbD6cf0070D6328E351cc30EEC9f) and the
// shared lens (0x25b5Df581f4b2Ed450203f375ad8A28b17F115B3) -- ground truth
// pulled from the explorer, not the PDF's advertised
// stonkbrokers.io/integration/ download. All 8 V2 pads share this ABI (same
// verified contract name + compiler settings).
//
// aggregatorV3ABI / twapPoolABI back the buy-side oracle reads (see
// embed.go).
var (
	PadABI          abi.ABI
	lensABI         abi.ABI
	aggregatorV3ABI abi.ABI
	twapPoolABI     abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&PadABI, padBytes},
		{&lensABI, lensBytes},
		{&aggregatorV3ABI, aggregatorV3Bytes},
		{&twapPoolABI, twapPoolBytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
