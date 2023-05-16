package types

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type EncodingData struct {
	EncodingMode EncodingMode

	TokenIn string

	// InputAmount total amount of TokenIn (before fee)
	InputAmount *big.Int

	TokenOut string

	// OutputAmount total amount of TokenOut (after fee)
	OutputAmount *big.Int

	// TotalAmountOut total amount out of TokenOut (before fee)
	TotalAmountOut *big.Int

	Route [][]EncodingSwap

	Recipient         string
	SlippageTolerance *big.Int
	Deadline          *big.Int

	ExtraFee valueobject.ExtraFee
	Flags    []EncodingFlag

	ClientData []byte

	// Allow user to swap without approving token beforehand
	Permit []byte
}
