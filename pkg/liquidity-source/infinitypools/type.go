package infinitypools

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Extra struct {
	Splits         *big.Int       `json:"splits"`
	FactoryAddress common.Address `json:"factoryAddress"`
	QuoterAddress  common.Address `json:"quoterAddress"`
}
