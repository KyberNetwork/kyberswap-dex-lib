package clanker

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var (
	HookAddresses = []common.Address{
		common.HexToAddress("0xa0b0d2d00fd544d8e0887f1a3cedd6e24baf10cc"),
	}

	BPS_DENOMINATOR         = big.NewInt(10000)
	FEE_CONTROL_DENOMINATOR = big.NewInt(10_000_000_000)
	PROTOCOL_FEE_NUMERATOR  = big.NewInt(200_000)                                              // 20% of the imposed LP fee
	FEE_DENOMINATOR         = big.NewInt(1_000_000)                                            // Uniswap 100% fee
	maxUint24               = new(big.Int).Sub(new(big.Int).Lsh(common.Big1, 24), common.Big1) // 2^24 - 1

)
