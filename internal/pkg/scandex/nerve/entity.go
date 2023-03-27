package nerve

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
	DefaultDepositFee  *big.Int
	DefaultWithdrawFee *big.Int
	Devaddr            common.Address
	LpToken            common.Address
}

type Balances []*big.Int
