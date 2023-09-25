package validator

import (
	"math/big"
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
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

	if err := v.validateSources(params.ExcludedSources); err != nil {
		return err
	}

	if err := v.validateSources(params.IncludedSources); err != nil {
		return err
	}
	return nil
}

func (v *getRoutesParamsValidator) validateTokenIn(tokenIn, tokenOut string) error {
	if len(tokenIn) == 0 {
		return NewValidationError("tokenIn", "required")
	}

	if !IsEthereumAddress(tokenIn) {
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

	if !IsEthereumAddress(tokenOut) {
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

func (v *getRoutesParamsValidator) validateSources(sources string) error {
	dexes := utils.TransformSliceParams(sources)
	for _, dex := range dexes {
		if !valueobject.IsAnExchange(valueobject.Exchange(dex)) {
			return NewValidationError("AvailableSources", "invalid")
		}
	}
	return nil
}
