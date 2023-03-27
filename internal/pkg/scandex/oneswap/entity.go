package oneswap

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type SwapStorage struct {
	InitialA           *big.Int
	FutureA            *big.Int
	InitialATime       *big.Int
	FutureATime        *big.Int
	SwapFee            *big.Int
	AdminFee           *big.Int
	DefaultWithdrawFee *big.Int
	LpToken            common.Address
}

type Balances []*big.Int
