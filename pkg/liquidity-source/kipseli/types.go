package kipseli

import (
	"errors"
	"math/big"
)

type Extra struct {
	Samples          [][][2]*big.Int              `json:"samples"`
	MaxIn            []*big.Int                   `json:"maxIn,omitempty"`
	SO               map[string]map[string]string `json:"so,omitempty"`
	LastUpdatedBlock uint64                       `json:"lub,omitempty"`
}

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
)
