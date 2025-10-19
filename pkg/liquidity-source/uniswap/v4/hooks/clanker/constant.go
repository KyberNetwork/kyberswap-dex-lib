package clanker

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	Million              = big.NewInt(1_000_000)
	FeeDenominator       = Million // Uniswap 100% fee
	ProtocolFeeNumerator = big.NewInt(200_000)

	BpsDenominator        = uint256.NewInt(10_000)
	FeeControlDenominator = uint256.NewInt(10_000_000_000)
	maxUint24             = uint64(1<<24 - 1)

	ClankerAddressByChain = map[valueobject.ChainID]string{
		valueobject.ChainIDBase:        "0xE85A59c628F7d27878ACeB4bf3b35733630083a9",
		valueobject.ChainIDUnichain:    "0xE85A59c628F7d27878ACeB4bf3b35733630083a9",
		valueobject.ChainIDArbitrumOne: "0xEb9D2A726Edffc887a574dC7f46b3a3638E8E44f",
		valueobject.ChainIDEthereum:    "0x6C8599779B03B00AAaE63C6378830919Abb75473",
	}

	DynamicFeeHookAddresses = []common.Address{
		common.HexToAddress("0x34a45c6B61876d739400Bd71228CbcbD4F53E8cC"), // base
		common.HexToAddress("0x9A82BfCf5fd939CB7256f2d41479Bc0DC67968cC"), // base v2
		common.HexToAddress("0xd60D6B218116cFd801E28F78d011a203D2b068Cc"), // base v2 permission-ed
		common.HexToAddress("0x9b37A43422D7bBD4C8B231be11E50AD1acE828CC"), // unichain
		common.HexToAddress("0xFd213BE7883db36e1049dC42f5BD6A0ec66B68cC"), // arbitrum
	}

	StaticFeeHookAddresses = []common.Address{
		common.HexToAddress("0xDd5EeaFf7BD481AD55Db083062b13a3cdf0A68CC"), // base
		common.HexToAddress("0xBF5ACAB339D2970938Ff4A2753d6cbbb8AaaE8cC"), // base v2
		common.HexToAddress("0xb429d62f8f3bFFb98CdB9569533eA23bF0Ba28CC"), // base v2 permission-ed
		common.HexToAddress("0xBc6e5aBDa425309c2534Bc2bC92562F5419ce8Cc"), // unichain
		common.HexToAddress("0xf7aC669593d2D9D01026Fa5B756DD5B4f7aAa8Cc"), // arbitrum
		common.HexToAddress("0x6C24D0bCC264EF6A740754A11cA579b9d225e8Cc"), // ethereum
	}
)
