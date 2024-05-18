package univ3

import (
	"math"
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

const (
	routerExactInputSingleCalldataLength = 4 + // selector
		32 + // tokenIn
		32 + // tokenOut
		32 + // fee
		32 + // recipient
		32 + // deadline
		32 + // amountIn
		32 + // amountInMinimum
		32 // sqrtPriceLimitX96
)

var (
	routerExactInputSingleSelector = []byte{0x41, 0x4b, 0xf3, 0x89}
	deadlineBytes                  = uint256.NewInt(math.MaxUint64).Bytes32()
)

// PackRouterExactInputSingleCalldata pack SwapRouter.exactInputSingle calldata
func PackRouterExactInputSingleCalldata(amountIn, fee *big.Int, tokenIn, tokenOut, wallet gethcommon.Address) ([]byte, error) {
	calldata := make([]byte, routerExactInputSingleCalldataLength)
	copy(calldata, routerExactInputSingleSelector)
	copy(calldata[4:][12:], tokenIn[:])
	copy(calldata[4:][32:][12:], tokenOut[:])
	feeU256, _ := uint256.FromBig(fee)
	feeBytes := feeU256.Bytes32()
	copy(calldata[4:][32:][32:], feeBytes[:])
	copy(calldata[4:][32:][32:][32:][12:], wallet[:])
	copy(calldata[4:][32:][32:][32:][32:], deadlineBytes[:])
	a, _ := uint256.FromBig(amountIn)
	aBytes := a.Bytes32()
	copy(calldata[4:][32:][32:][32:][32:][32:], aBytes[:])
	return calldata, nil
}
