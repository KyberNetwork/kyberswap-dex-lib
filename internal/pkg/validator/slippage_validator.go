package validator

type slippageValidator struct {
	config SlippageValidatorConfig
}

func NewSlippageValidator(config SlippageValidatorConfig) *slippageValidator {
	return &slippageValidator{
		config: config,
	}
}

func (v *slippageValidator) Validate(slippageTolerance float64, ignoreCappedSlippage bool) error {
	// ignore check or slippage is valid
	if ignoreCappedSlippage ||
		(slippageTolerance >= v.config.SlippageToleranceGTE && slippageTolerance <= v.config.SlippageToleranceLTE) {
		return nil
	}

	return NewValidationError("slippageTolerance", "invalid")
}
