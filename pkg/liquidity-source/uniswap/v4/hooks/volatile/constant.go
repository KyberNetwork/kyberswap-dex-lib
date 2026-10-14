package volatile

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const DexType = valueobject.ExchangeUniswapV4Volatile

var (
	HookAddress           = common.HexToAddress("0x000022Ec4eBD6b12294396E852E4C3876b253Cc0")
	ImplementationAddress = common.HexToAddress("0x1ddbf7215be4f8a7623b9b1b0255c6e1e3f6c000")
	implementationSlot    = common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")
	proxyCodeHash         = common.HexToHash("0xa5659fa6440c9830926d098ec347a558447bfd8e483643e0d02398dc4cff1685")

	ErrPoolIsNotTracked      = errors.New("volatile: pool is not tracked")
	ErrUnsupportedDeployment = errors.New("volatile: unsupported deployment")
	ErrInvalidSwapFee        = errors.New("volatile: invalid swap fee")
)
