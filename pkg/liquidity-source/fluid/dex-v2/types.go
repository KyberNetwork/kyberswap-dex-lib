package dexv2

import "github.com/ethereum/go-ethereum/common"

type Metadata struct {
	LastCreatedAtTimestamp int      `json:"lastCreatedAtTimestamp"`
	LastPoolIds            []string `json:"lastPoolIds"`
}

type SubgraphPool struct {
	ID                 string `json:"id"`
	DexId              string `json:"dexId"`
	DexType            int    `json:"dexType"`
	Token0             string `json:"token0"`
	Token1             string `json:"token1"`
	Fee                int    `json:"fee"`
	TickSpacing        int    `json:"tickSpacing"`
	Controller         string `json:"controller"`
	CreatedAtTimestamp string `json:"createdAtTimestamp"`
}

type Extra struct {
	DexType     int            `json:"dexType"`
	Fee         int            `json:"fee"`
	TickSpacing int            `json:"tickSpacing"`
	Controller  common.Address `json:"controller,omitempty"`
	IsNative    [2]bool        `json:"isNative"`
}
