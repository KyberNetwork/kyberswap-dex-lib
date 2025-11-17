package pancakev3

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/v3/abis"
)

var (
	pancakeV3PoolABI    abi.ABI
	pancakeV3FactoryABI abi.ABI
)

var (
	poolFilterer    = lo.Must(abis.NewPoolFilterer(common.Address{}, nil))
	factoryFilterer = lo.Must(abis.NewFactoryFilterer(common.Address{}, nil))
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&pancakeV3PoolABI, pancakeV3PoolJson},
		{&pancakeV3FactoryABI, pancakeV3FactoryJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
