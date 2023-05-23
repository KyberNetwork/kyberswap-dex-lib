package nerve

import "math/big"

const (
	DexTypeNerve = "nerve"

	methodGetSwapStorage  = "swapStorage"
	methodGetTokenBalance = "getTokenBalance"
	methodGetTotalSupply  = "totalSupply"
	reserveZero           = "0"
)

var Zero = big.NewInt(0)
