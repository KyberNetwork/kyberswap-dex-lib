package gsm4626

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	gsm4626ABI       abi.ABI
	priceStrategyABI abi.ABI
	feeStrategyABI   abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&gsm4626ABI, gsm4626Bytes},
		{&priceStrategyABI, priceStrategyBytes},
		{&feeStrategyABI, feeStrategyBytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
