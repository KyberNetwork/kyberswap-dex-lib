package volatile

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const DexType = valueobject.ExchangeUniswapV4Volatile

// HookGas is the mean gas of the beforeSwap + afterSwap calls into the hook over 44 first-in-block
// swaps (blocks 26192646..26193390) on the ETH/USDC pool, excluding txs with an access list
// (they skip cold-access costs; the executor sends none). Tenderly sim of our own swap: 23761 + 1438.
const HookGas int64 = 24820

var (
	HookAddress = common.HexToAddress("0x000022Ec4eBD6b12294396E852E4C3876b253Cc0")

	ErrPoolIsNotTracked = errors.New("volatile: pool is not tracked")
	ErrInvalidFeeState  = errors.New("volatile: invalid fee state")
	ErrInvalidSqrtPrice = errors.New("volatile: invalid sqrt price")
)
