package validator

import (
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/api/params"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/valueobject"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
)

type getRoutesParamsValidator struct{}

func NewGetRouteParamsValidator() *getRoutesParamsValidator {
	return &getRoutesParamsValidator{}
}

func (v *getRoutesParamsValidator) Validate(params params.GetRoutesParams) error {
	if err := v.validateTokenIn(params.TokenIn, params.TokenOut); err != nil {
		return err
	}

	if err := v.validateTokenOut(params.TokenOut); err != nil {
		return err
	}

	if err := v.validateAmountIn(params.AmountIn); err != nil {
		return err
	}

	if err := v.validateFeeReceiver(params.FeeReceiver); err != nil {
		return err
	}

	if err := v.validateFeeAmount(params.FeeAmount); err != nil {
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

func (v *getRoutesParamsValidator) validateTokenIn(tokenIn, tokenOut string) error {
	if len(tokenIn) == 0 {
		return NewValidationError("tokenIn", "required")
	}

	if !isEthereumAddress(tokenIn) {
		return NewValidationError("tokenIn", "invalid")
	}

	if strings.EqualFold(tokenIn, tokenOut) {
		return NewValidationError("tokenIn", "identical with tokenOut")
	}

	return nil
}

func (v *getRoutesParamsValidator) validateTokenOut(tokenOut string) error {
	if len(tokenOut) == 0 {
		return NewValidationError("tokenOut", "required")
	}

	if !isEthereumAddress(tokenOut) {
		return NewValidationError("tokenOut", "invalid")
	}

	return nil
}

func (v *getRoutesParamsValidator) validateAmountIn(amountInParams string) error {
	amountIn, ok := new(big.Int).SetString(amountInParams, 10)
	if !ok {
		return NewValidationError("amountIn", "invalid")
	}

	if amountIn.Cmp(constant.Zero) <= 0 {
		return NewValidationError("amountIn", "invalid")
	}

	return nil
}

func (v *getRoutesParamsValidator) validateFeeReceiver(feeReceiver string) error {
	if len(feeReceiver) == 0 {
		return nil
	}

	if !isEthereumAddress(feeReceiver) {
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

func (v *getRoutesParamsValidator) validateChargeFeeBy(chargeFeeBy string, feeAmount string) error {
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
