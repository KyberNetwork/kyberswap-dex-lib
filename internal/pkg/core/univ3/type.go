package univ3

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// ExactInputSingleParams https://github.com/Uniswap/v3-periphery/blob/main/contracts/interfaces/ISwapRouter.sol#L10
type ExactInputSingleParams struct {
	TokenIn           common.Address
	TokenOut          common.Address
	Fee               *big.Int
	Recipient         common.Address
	Deadline          *big.Int
	AmountIn          *big.Int
	AmountOutMinimum  *big.Int
	SqrtPriceLimitX96 *big.Int
}
