package dpp

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	abis "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/dpp/abi"
)

var (
	factoryABI abi.ABI
)

var (
	factoryFilterer *abis.DPPFactoryFilterer
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&factoryABI, factoryABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}

	factoryFilterer = lo.Must(abis.NewDPPFactoryFilterer(common.Address{}, nil))
}
