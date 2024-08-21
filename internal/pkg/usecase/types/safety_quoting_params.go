package types

import (
	"math/big"
)

type (
	SafetyQuoteCategory string

	SafetyQuotingParams struct {
		PoolType             string
		TokenIn              string
		TokenOut             string
		ApplyDeductionFactor bool
		ClientId             string
	}
)

const (
	Default        SafetyQuoteCategory = "Default"
	StrictlyStable SafetyQuoteCategory = "StrictlyStable"
	Stable         SafetyQuoteCategory = "Stable"
	Correlated     SafetyQuoteCategory = "Correlated"
	LowSlippage    SafetyQuoteCategory = "LowSlippaged"
	NormalSlippage SafetyQuoteCategory = "NormalSlippage"
	HighSlippage   SafetyQuoteCategory = "HighSlippage"
)

var (
	// SafetyQuoteMappingDefault defines the default safety quote factors for each category
	SafetyQuoteMappingDefault = map[SafetyQuoteCategory]float64{
		Default:        0,
		StrictlyStable: 0,
		Stable:         0.5,
		Correlated:     1.5,
		LowSlippage:    3,
		NormalSlippage: 10,
		HighSlippage:   50,
	}

	// BasisPoint is one hundredth of 1 percentage point
	// https://en.wikipedia.org/wiki/Basis_point
	BasisPointMulByTen = big.NewInt(10 * 10_000)
)
