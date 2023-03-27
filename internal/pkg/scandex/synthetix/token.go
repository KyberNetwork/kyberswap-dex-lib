package synthetix

import "github.com/ethereum/go-ethereum/common"

type Token struct {
	Address  common.Address `json:"address"`
	Decimals uint8          `json:"decimals"`
	Symbol   string         `json:"symbol"`
}
