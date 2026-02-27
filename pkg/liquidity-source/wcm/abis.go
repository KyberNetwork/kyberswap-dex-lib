package wcm

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	compositeExchangeABI abi.ABI
	spotOrderBookABI     abi.ABI
	erc20ABI             abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&compositeExchangeABI, compositeExchangeJson},
		{&spotOrderBookABI, spotOrderBookJson},
		{&erc20ABI, []byte(`[{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"payable":false,"type":"function"}]`)},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
