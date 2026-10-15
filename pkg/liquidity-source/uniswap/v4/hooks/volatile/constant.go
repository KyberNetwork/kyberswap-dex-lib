package volatile

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const DexType = valueobject.ExchangeUniswapV4Volatile

var (
	HookAddress = common.HexToAddress("0x000022Ec4eBD6b12294396E852E4C3876b253Cc0")

	ErrPoolIsNotTracked = errors.New("volatile: pool is not tracked")
	ErrInvalidSwapFee   = errors.New("volatile: invalid swap fee")
)
