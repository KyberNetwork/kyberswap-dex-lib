package ilyrislb

import (
	"bytes"
	_ "embed"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	//go:embed abi/BinPoolLens.json
	lensABIData []byte
	//go:embed abi/BinPool.json
	poolABIData []byte
	//go:embed abi/BinFactory.json
	factoryABIData []byte
	//go:embed abi/ContinuousMarketGuard.json
	guardABIData []byte

	lensABI, poolABI, factoryABI, guardABI abi.ABI
)

func init() {
	for _, b := range []struct {
		abi  *abi.ABI
		data []byte
	}{{&lensABI, lensABIData}, {&poolABI, poolABIData}, {&factoryABI, factoryABIData}, {&guardABI, guardABIData}} {
		parsed, err := abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
		*b.abi = parsed
	}
}
