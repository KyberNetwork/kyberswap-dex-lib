package carbon

import (
	"bytes"
	_ "embed"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	abis "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/carbon/abi"
)

var (
	controllerABI abi.ABI

	controllerFilterer = lo.Must(abis.NewControllerFilterer(common.Address{}, nil))
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&controllerABI, controllerBytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
