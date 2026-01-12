package shared

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type (
	Rounding int
	SwapKind int
)

var (
	BatchRouterMap = map[string]map[valueobject.ChainID]common.Address{
		"": {
			valueobject.ChainIDArbitrumOne:     common.HexToAddress("0xaD89051bEd8d96f045E8912aE1672c6C0bF8a85E"),
			valueobject.ChainIDAvalancheCChain: common.HexToAddress("0xc9b36096f5201ea332Db35d6D195774ea0D5988f"),
			valueobject.ChainIDBase:            common.HexToAddress("0x85a80afee867aDf27B50BdB7b76DA70f1E853062"),
			valueobject.ChainIDEthereum:        common.HexToAddress("0x136f1EFcC3f8f88516B9E94110D56FDBfB1778d1"),
			valueobject.ChainIDOptimism:        common.HexToAddress("0xaD89051bEd8d96f045E8912aE1672c6C0bF8a85E"),
			valueobject.ChainIDSonic:           common.HexToAddress("0x7761659F9e9834ad367e4d25E0306ba7A4968DAf"),
			valueobject.ChainIDHyperEVM:        common.HexToAddress("0x9dd5Db2d38b50bEF682cE532bCca5DfD203915E1"),
			valueobject.ChainIDPlasma:          common.HexToAddress("0x85a80afee867aDf27B50BdB7b76DA70f1E853062"),
		},
		"coinhane": {
			valueobject.ChainIDBSC: common.HexToAddress("0xBdf45255f34B9DD3d2F9aDacc7AeB482059f1C54"),
		},
	}
	VaultMap = map[string]common.Address{
		"":         common.HexToAddress("0xbA1333333333a1BA1108E8412f11850A5C319bA9"), // default
		"coinhane": common.HexToAddress("0xb61cb1E8EF4BB1b74bB858B8B60d82d79488F13D"),
	}

	AddrDummy = common.HexToAddress("0x1371783000000000000000000000000001371760")
)

const (
	RoundUp Rounding = iota
	RoundDown
)

const (
	ExactIn SwapKind = iota
	ExactOut
)

const (
	RelistInterval = 60 // relist every 60 times

	VaultMethodGetBufferAsset             = "getBufferAsset"
	VaultMethodGetHooksConfig             = "getHooksConfig"
	VaultMethodGetStaticSwapFeePercentage = "getStaticSwapFeePercentage"
	VaultMethodGetAggregateFeePercentages = "getAggregateFeePercentages"
	VaultMethodGetPoolData                = "getPoolData"

	VaultMethodIsVaultPaused        = "isVaultPaused"
	VaultMethodIsPoolPaused         = "isPoolPaused"
	VaultMethodIsPoolInRecoveryMode = "isPoolInRecoveryMode"

	ERC4626MethodConvertToAssets = "convertToAssets"
	ERC4626MethodConvertToShares = "convertToShares"
	ERC4626MethodMaxDeposit      = "maxDeposit"
	ERC4626MethodMaxRedeem       = "maxRedeem"
)
