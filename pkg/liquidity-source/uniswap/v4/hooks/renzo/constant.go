package renzo

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	WAD           = big.NewInt(1e18)
	B1e12         = big.NewInt(1e12)
	q96           = new(big.Int).Lsh(bignumber.One, 96)
	q192          = new(big.Int).Lsh(bignumber.One, 192)
	HookAddresses = []common.Address{
		common.HexToAddress("0x09dea99d714a3a19378e3d80d1ad22ca46085080"),
	}
)
