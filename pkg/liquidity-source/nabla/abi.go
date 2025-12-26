package nabla

import (
	"bytes"
	_ "embed"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	abis "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/nabla/abi"
)

var (
	RouterABI        abi.ABI
	portalABI        abi.ABI
	swapPoolABI      abi.ABI
	curveABI         abi.ABI
	oracleABI        abi.ABI
	pythAdapterV2ABI abi.ABI

	swapPoolFilterer = lo.Must(abis.NewNablaSwapPoolFilterer(common.Address{}, nil))
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&RouterABI, routerBytes},
		{&portalABI, portalBytes},
		{&swapPoolABI, swapPoolBytes},
		{&curveABI, curveBytes},
		{&oracleABI, oracleBytes},
		{&pythAdapterV2ABI, pythAdapterV2Bytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
