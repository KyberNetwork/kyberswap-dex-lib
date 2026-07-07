package uniswapv1

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	ChainID                  valueobject.ChainID `json:"chainID"`
	MulticallContractAddress string              `json:"multicallContractAddress"`
	FactoryAddress           string              `json:"factoryAddress"`
	NewPoolLimit             int                 `json:"newPoolLimit"`
}
