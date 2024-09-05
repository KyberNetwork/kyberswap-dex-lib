package maverickv2

import "math/big"

type State struct {
	ReserveA      *big.Int `json:"reserveA"`
	ReserveB      *big.Int `json:"reserveB"`
	LastTimestamp int64    `json:"lastTimestamp"`
}
