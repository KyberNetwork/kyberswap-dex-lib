package nuriv2

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/nuriv2/abis"
)

var (
	poolABI    abi.ABI
	factoryABI abi.ABI
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
		{&poolABI, poolJson},
		{&factoryABI, factoryJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
