package sfrxeth

import (
	"errors"
)

const (
	DexType = "sfrxeth"
)

const (
	minterMethodSubmitPaused = "submitPaused"

	SfrxETHMethodTotalAssets = "totalAssets"
	SfrxETHMethodTotalSupply = "totalSupply"
)

var (
	defaultGas = Gas{
		SubmitAndDeposit: 90000,
	}
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrSubmitPaused = errors.New("submit is paused")
)
