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
	portalABI        abi.ABI
	routerABI        abi.ABI
	swapPoolABI      abi.ABI
	curveABI         abi.ABI
	oracleABI        abi.ABI
	pythAdapterV2ABI abi.ABI

	oracleFilterer   = lo.Must(abis.NewNablaOracleFilterer(common.Address{}, nil))
	swapPoolFilterer = lo.Must(abis.NewNablaSwapPoolFilterer(common.Address{}, nil))
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&portalABI, portalBytes},
		{&routerABI, routerBytes},
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
