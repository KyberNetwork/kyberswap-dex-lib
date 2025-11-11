package liquiditybookv20

import (
	"bytes"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/liquiditybookv20/abis"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
)

var (
	pairABI    abi.ABI
	factoryABI abi.ABI
	routerABI  abi.ABI
)

var (
	pairFilterer    = lo.Must(abis.NewLBPairFilterer(common.Address{}, nil))
	factoryFilterer = lo.Must(abis.NewLBFactoryFilterer(common.Address{}, nil))
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
		{
			&routerABI, routerABIJson,
		},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
