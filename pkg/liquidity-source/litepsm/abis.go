package litepsm

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	erc20ABI     abi.ABI
	litePSMABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&erc20ABI, erc20ABIBytes},
		{&litePSMABI, litePSMBytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
