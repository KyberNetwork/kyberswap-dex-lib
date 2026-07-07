package tricryptong

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	curveTricryptoNGABI abi.ABI
)

func init() {
	var build = []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&curveTricryptoNGABI, curveTricryptoNGABIBytes},
	}

	var err error
	for _, b := range build {
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
