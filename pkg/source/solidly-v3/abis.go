package solidlyv3

import (
	"bytes"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/solidly-v3/abis"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
)

var (
	solidlyV3PoolABI    abi.ABI
	solidlyV3FactoryABI abi.ABI
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
		{&solidlyV3PoolABI, solidlyV3PoolJson},
		{&solidlyV3FactoryABI, solidlyV3FactoryJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
