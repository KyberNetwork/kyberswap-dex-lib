package entity

import (
	"math/big"
)

type BlockHeader struct {
	Number    *big.Int `json:"number"`
	Hash      string   `json:"hash"`
	Timestamp uint64   `json:"timestamp"`
}
