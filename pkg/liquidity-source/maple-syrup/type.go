package maplesyrup

import "github.com/holiman/uint256"

type SwapType uint8

const (
	None SwapType = iota
	Deposit
	Redeem
	Both
)

type (
	Gas struct {
		Deposit uint64 `json:"d,omitempty"`
	}

	Extra struct {
		Gas          Gas            `json:"g"`
		SwapTypes    SwapType       `json:"sT,omitempty"`
		MaxDeposit   *uint256.Int   `json:"mD,omitempty"`
		DepositRates []*uint256.Int `json:"dR,omitempty"`
		Router       string         `json:"router"`
		Active       bool           `json:"active"`
		LiquidityCap *uint256.Int   `json:"liquidityCap"`
	}

	Meta struct {
		BlockNumber uint64 `json:"blockNumber"`
		Router      string `json:"router"`
	}
)
