package base

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
)

const (
	bufferGas int64 = 120534
)

var BalancerV3BatchRouter = map[valueobject.ChainID]common.Address{
	valueobject.ChainIDArbitrumOne:     common.HexToAddress("0xaD89051bEd8d96f045E8912aE1672c6C0bF8a85E"),
	valueobject.ChainIDAvalancheCChain: common.HexToAddress("0xc9b36096f5201ea332Db35d6D195774ea0D5988f"),
	valueobject.ChainIDBase:            common.HexToAddress("0x85a80afee867aDf27B50BdB7b76DA70f1E853062"),
	valueobject.ChainIDEthereum:        common.HexToAddress("0x136f1EFcC3f8f88516B9E94110D56FDBfB1778d1"),
	valueobject.ChainIDOptimism:        common.HexToAddress("0xaD89051bEd8d96f045E8912aE1672c6C0bF8a85E"),
	valueobject.ChainIDSonic:           common.HexToAddress("0x7761659F9e9834ad367e4d25E0306ba7A4968DAf"),
	valueobject.ChainIDHyperEVM:        common.HexToAddress("0x9dd5Db2d38b50bEF682cE532bCca5DfD203915E1"),
}
