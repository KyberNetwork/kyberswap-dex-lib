package aevm

import (
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

const (
	approveCalldataLength  = 4 + 32 + 32
	transferCalldataLength = 4 + 32 + 32
)

var (
	erc20ApproveSelector  = []byte{0x09, 0x5e, 0xa7, 0xb3}
	erc20TransferSelector = []byte{0xa9, 0x05, 0x9c, 0xbb}
)

func PackERC20ApproveCall(addr gethcommon.Address, amountIn *big.Int) ([]byte, error) {
	calldata := make([]byte, approveCalldataLength)
	copy(calldata, erc20ApproveSelector)
	copy(calldata[4:][12:], addr[:])
	a, _ := uint256.FromBig(amountIn)
	aBytes := a.Bytes32()
	copy(calldata[4:][32:], aBytes[:])
	return calldata, nil
}

func PackERC20TransferCall(to gethcommon.Address, amount *big.Int) ([]byte, error) {
	calldata := make([]byte, transferCalldataLength)
	copy(calldata, erc20TransferSelector)
	copy(calldata[4:][12:], to[:])
	a, _ := uint256.FromBig(amount)
	aBytes := a.Bytes32()
	copy(calldata[4:][32:], aBytes[:])
	return calldata, nil
}
