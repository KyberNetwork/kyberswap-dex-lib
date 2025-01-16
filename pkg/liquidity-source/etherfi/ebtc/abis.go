package etherfiebtc

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	tellerABI     abi.ABI
	accountantABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&tellerABI, tellerABIData},
		{&accountantABI, accountantABIData},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
