package elfomofi

import (
	"math/big"
)

type Extra struct {
	Samples        [][][2]*big.Int `json:"samples"`
	FactoryAddress string          `json:"factoryAddress"`
}
