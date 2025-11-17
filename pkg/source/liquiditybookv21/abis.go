package liquiditybookv21

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/liquiditybookv21/abis"
)

var (
	pairABI    abi.ABI
	factoryABI abi.ABI
)

var (
	pairFilterer    *abis.LBPairFilterer
	factoryFilterer *abis.LBFactoryFilterer
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&pairABI, pairABIJson,
		},
		{
			&factoryABI, factoryABIJson,
		},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}

	pairFilterer = lo.Must(abis.NewLBPairFilterer(common.Address{}, nil))
	factoryFilterer = lo.Must(abis.NewLBFactoryFilterer(common.Address{}, nil))
}
