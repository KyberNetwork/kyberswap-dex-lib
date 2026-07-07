package abis

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
)

var (
	UniswapV3PoolABI    abi.ABI
	UniswapV3FactoryABI abi.ABI
)

var (
	UniswapV3PoolFilterer    *PoolFilterer
	UniswapV3FactoryFilterer *FactoryFilterer
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&UniswapV3PoolABI, uniswapV3PoolJson},
		{&UniswapV3FactoryABI, uniswapV3FactoryJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}

	UniswapV3PoolFilterer = lo.Must(NewPoolFilterer(common.Address{}, nil))
	UniswapV3FactoryFilterer = lo.Must(NewFactoryFilterer(common.Address{}, nil))
}
