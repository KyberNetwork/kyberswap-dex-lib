package business

import (
	"fmt"
)

// GenDirectPairKey returns pair key from address of tokenIn and tokenOut
func GenDirectPairKey(tokenIn, tokenOut string) string {
	if tokenIn > tokenOut {
		return fmt.Sprintf("%s-%s", tokenIn, tokenOut)
	}
	return fmt.Sprintf("%s-%s", tokenOut, tokenIn)
}
