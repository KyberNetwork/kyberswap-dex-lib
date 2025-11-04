package deli

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

var (
	HookAddresses = []common.Address{
		common.HexToAddress("0xC384B99A6e5cD1a800B2d83aB71BaB7bD712b0cc"), // DeliHook
	}
	ConstantProductAddresses = []common.Address{
		common.HexToAddress("0x00C9DA9AbC5303219ead3Cf0307b5A8A7644BaC8"), // DeliHookConstantProduct
	}
)

const wBLT = "0x4e74d4db6c0726ccded4656d0bce448876bb4c7a"

var (
	FeeDenom  = big.NewInt(1e6)
	UFeeDenom = uint256.NewInt(1e6)
)
