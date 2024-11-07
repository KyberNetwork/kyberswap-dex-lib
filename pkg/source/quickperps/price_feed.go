package quickperps

import (
	"math/big"
)

type PriceFeed struct {
	Price     *big.Int `json:"price"`
	Timestamp uint32   `json:"timestamp"`
}

const (
	priceFeedMethodRead = "read"
)
