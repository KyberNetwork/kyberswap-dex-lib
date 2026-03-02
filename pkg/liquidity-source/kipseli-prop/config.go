package kipseliprop

import (
	"github.com/ethereum/go-ethereum/common"
)

type Config struct {
	DexID         string         `json:"dexID"`
	ChainID       int            `json:"chainId"`
	LensAddress   string         `json:"lensAddress"`
	RouterAddress string         `json:"routerAddress"`
	Verifier      common.Address `json:"verifier"`
	Quoter        common.Hash    `json:"quoter"`
	Buffer        int64          `json:"buffer"`
}
