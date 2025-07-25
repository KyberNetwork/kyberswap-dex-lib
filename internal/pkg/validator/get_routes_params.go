package validator

import (
	"math/big"
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type getRoutesParamsValidator struct {
	chainID valueobject.ChainID
}

func NewGetRouteParamsValidator(chainID valueobject.ChainID) *getRoutesParamsValidator {
	return &getRoutesParamsValidator{
		chainID: chainID,
	}
}

func (v *getRoutesParamsValidator) ValidateBundled(params params.GetBundledRoutesParams) error {
	if len(params.TokensIn) != len(params.TokensOut) || len(params.TokensIn) != len(params.AmountsIn) {
		return NewValidationError("tokensIn", "should have same length with tokensOut and amountsIn")
	}
	for i, tokenIn := range params.TokensIn {
		if err := v.validateTokens(tokenIn, params.TokensOut[i]); err != nil {
			return err
		}

		if err := v.validateAmountIn(params.AmountsIn[i]); err != nil {
			return err
		}
	}

	if err := v.validateGasPrice(params.GasPrice); err != nil {
		return err
	}

	return nil
}

func (v *getRoutesParamsValidator) Validate(params params.GetRoutesParams) error {
	if err := v.validateTokens(params.TokenIn, params.TokenOut); err != nil {
		return err
	}

	if err := v.validateAmountIn(params.AmountIn); err != nil {
		return err
	}

	if err := v.validateChargeFeeBy(params.ChargeFeeBy, params.FeeAmount); err != nil {
		return err
	}

	if err := v.validateGasPrice(params.GasPrice); err != nil {
		return err
	}

	return nil
}

func (v *getRoutesParamsValidator) validateTokens(tokenIn, tokenOut string) error {
	// validate tokenIn
	if len(tokenIn) == 0 {
		return NewValidationError("tokenIn", "required")
	}

	if !IsEthereumAddress(tokenIn) {
		return NewValidationError("tokenIn", "invalid")
	}

	if strings.EqualFold(tokenIn, tokenOut) {
		return NewValidationError("tokenIn", "identical with tokenOut")
	}

	// validate tokenOut
	if len(tokenOut) == 0 {
		return NewValidationError("tokenOut", "required")
	}

	if !IsEthereumAddress(tokenOut) {
		return NewValidationError("tokenOut", "invalid")
	}

	if strings.EqualFold(valueobject.WrapNativeLower(tokenIn, v.chainID), tokenOut) ||
		strings.EqualFold(valueobject.WrapNativeLower(tokenOut, v.chainID), tokenIn) {
		return NewValidationError("tokens", "swapping between native and wrapped native is not allowed")
	}

	return nil
}

func (v *getRoutesParamsValidator) validateAmountIn(amountInParams string) error {
	amountIn, ok := new(big.Int).SetString(amountInParams, 10)
	if !ok {
		return NewValidationError("amountIn", "invalid")
	}

	if amountIn.Sign() <= 0 {
		return NewValidationError("amountIn", "invalid")
	}

	return nil
}

func (v *getRoutesParamsValidator) validateFeeReceiver(feeReceiver string) error {
	if len(feeReceiver) == 0 {
		return nil
	}

	if !IsEthereumAddress(feeReceiver) {
		return NewValidationError("feeReceiver", "invalid")
	}

	return nil
}

func (v *getRoutesParamsValidator) validateFeeAmount(feeAmount string) error {
	if len(feeAmount) == 0 {
		return nil
	}

	if _, ok := new(big.Int).SetString(feeAmount, 10); !ok {
		return NewValidationError("feeAmount", "invalid")
	}

	return nil
}

func (v *getRoutesParamsValidator) validateChargeFeeBy(chargeFeeBy valueobject.ChargeFeeBy, feeAmount string) error {
	if len(feeAmount) == 0 {
		return nil
	}

	for _, value := range valueobject.ChargeFeeByValues {
		if chargeFeeBy == value {
			return nil
		}
	}

	return NewValidationError("chargeFeeBy", "invalid")
}

func (v *getRoutesParamsValidator) validateGasPrice(gasPriceStr string) error {
	if len(gasPriceStr) == 0 {
		return nil
	}

	_, ok := new(big.Float).SetString(gasPriceStr)
	if !ok {
		return NewValidationError("gasPrice", "invalid")
	}

	return nil
}
