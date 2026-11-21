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

// ChangingFeeMaxFee is ChangingFeeHook's MAX_CHARGED_FEE_E6: the most it charges a swap, 20,000 pips (2%) a side,
// applied to the fee beforeSwap returns and getFee quotes. Its stored fee state can be larger (the library's own clamp
// is 999,998, Uniswap's ceiling); the fee charged cannot.
const ChangingFeeMaxFee uint32 = 20_000

var (
	// HookAddress is Uniswap Labs' VolatilePairHook (a UUPS proxy).
	HookAddress = common.HexToAddress("0x000022Ec4eBD6b12294396E852E4C3876b253Cc0")

	// ChangingFeeHookAddresses are the three immutable ChangingFeeHook contracts (Polygon, Ethereum, Base): a
	// VolatilePairHook fork with the same fee math, the same feeConfig/feeState views, no proxy, no roles, no config
	// update and a 2% fee ceiling. Sources verified on Polygonscan, Etherscan and Sourcify.
	ChangingFeeHookAddresses = []common.Address{
		common.HexToAddress("0x3A9671f5ac0eE185Cc168EC0fC66e334A24d6080"), // Polygon (137)
		common.HexToAddress("0x89ea496c2d7a8834B85249984DC0279eDdaeA080"), // Ethereum (1)
		common.HexToAddress("0x5EAB65856C5fE9D8A06c40B412390660F5d3e080"), // Base (8453)
	}

	ErrPoolIsNotTracked = errors.New("volatile: pool is not tracked")
	ErrInvalidFeeState  = errors.New("volatile: invalid fee state")
	ErrInvalidSqrtPrice = errors.New("volatile: invalid sqrt price")
)
