package clanker

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

var (
	MILLION                = big.NewInt(1_000_000)
	FEE_DENOMINATOR        = MILLION // Uniswap 100% fee
	PROTOCOL_FEE_NUMERATOR = big.NewInt(200_000)

	BPS_DENOMINATOR         = uint256.NewInt(10_000)
	FEE_CONTROL_DENOMINATOR = uint256.NewInt(10_000_000_000)
	maxUint24               = uint64(1<<24 - 1)

	ClankerAddressByChain = map[valueobject.ChainID]string{
		valueobject.ChainIDBase:        "0xE85A59c628F7d27878ACeB4bf3b35733630083a9",
		valueobject.ChainIDUnichain:    "0xE85A59c628F7d27878ACeB4bf3b35733630083a9",
		valueobject.ChainIDArbitrumOne: "0xEb9D2A726Edffc887a574dC7f46b3a3638E8E44f",
	}

	DynamicFeeHookAddresses = []common.Address{
		common.HexToAddress("0x34a45c6B61876d739400Bd71228CbcbD4F53E8cC"), // base
		common.HexToAddress("0x9b37A43422D7bBD4C8B231be11E50AD1acE828CC"), // unichain
		common.HexToAddress("0xFd213BE7883db36e1049dC42f5BD6A0ec66B68cC"), // arbitrum
	}

	StaticFeeHookAddresses = []common.Address{
		common.HexToAddress("0xDd5EeaFf7BD481AD55Db083062b13a3cdf0A68CC"), // base
		common.HexToAddress("0xBc6e5aBDa425309c2534Bc2bC92562F5419ce8Cc"), // unichain
		common.HexToAddress("0xf7aC669593d2D9D01026Fa5B756DD5B4f7aAa8Cc"), // arbitrum
	}
)
