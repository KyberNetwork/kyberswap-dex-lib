package validator

import (
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type buildRouteParamsValidator struct {
	nowFunc func() time.Time
}

func NewBuildRouteParamsValidator(nowFunc func() time.Time) *buildRouteParamsValidator {
	return &buildRouteParamsValidator{
		nowFunc: nowFunc,
	}
}

func (v *buildRouteParamsValidator) Validate(params params.BuildRouteParams) error {
	if err := v.validateRoute(params.RouteSummary); err != nil {
		return err
	}

	if err := v.validateTokenIn(params.RouteSummary.TokenIn, params.RouteSummary.TokenOut); err != nil {
		return err
	}

	if err := v.validateTokenOut(params.RouteSummary.TokenOut); err != nil {
		return err
	}

	if err := v.validateSlippageTolerance(params.SlippageTolerance); err != nil {
		return err
	}

	if err := v.validateChargeFeeBy(params.RouteSummary.ExtraFee.ChargeFeeBy, params.RouteSummary.ExtraFee.FeeAmount); err != nil {
		return err
	}

	if err := v.validateFeeReceiver(params.RouteSummary.ExtraFee.FeeReceiver); err != nil {
		return err
	}

	if err := v.validateFeeAmount(params.RouteSummary.ExtraFee.FeeAmount); err != nil {
		return err
	}

	if err := v.validateDeadline(params.Deadline); err != nil {
		return err
	}

	if err := v.validateRecipient(params.Recipient); err != nil {
		return err
	}

	if err := v.validatePermit(params.Permit); err != nil {
		return err
	}

	return nil
}

func (v *buildRouteParamsValidator) validateRoute(route params.RouteSummary) error {
	if len(route.Route) == 0 {
		return NewValidationError("route.route", "empty route")
	}

	return nil
}

func (v *buildRouteParamsValidator) validateTokenIn(tokenIn, tokenOut string) error {
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

func (v *buildRouteParamsValidator) validateTokenOut(tokenOut string) error {
	if len(tokenOut) == 0 {
		return NewValidationError("tokenOut", "required")
	}

	if !isEthereumAddress(tokenOut) {
		return NewValidationError("tokenOut", "invalid")
	}

	return nil
}

func (v *buildRouteParamsValidator) validateSlippageTolerance(slippageTolerance int64) error {
	if slippageTolerance < 0 || slippageTolerance > constant.MaximumSlippage {
		return NewValidationError("slippageTolerance", "invalid")
	}

	return nil
}

func (v *buildRouteParamsValidator) validateChargeFeeBy(chargeFeeBy string, feeAmount string) error {
	if len(feeAmount) == 0 || feeAmount == "0" {
		return nil
	}

	for _, value := range valueobject.ChargeFeeByValues {
		if chargeFeeBy == value {
			return nil
		}
	}

	return NewValidationError("chargeFeeBy", "invalid")
}

func (v *buildRouteParamsValidator) validateFeeReceiver(feeReceiver string) error {
	if len(feeReceiver) == 0 {
		return nil
	}

	if !isEthereumAddress(feeReceiver) {
		return NewValidationError("feeReceiver", "invalid")
	}

	return nil
}

func (v *buildRouteParamsValidator) validateFeeAmount(feeAmount string) error {
	if len(feeAmount) == 0 {
		return nil
	}

	if _, ok := new(big.Int).SetString(feeAmount, 10); !ok {
		return NewValidationError("feeAmount", "invalid")
	}

	return nil
}

func (v *buildRouteParamsValidator) validateDeadline(deadline int64) error {
	if deadline == 0 {
		return nil
	}

	if deadline < v.nowFunc().Unix() {
		return NewValidationError("deadline", "in the past")
	}

	return nil
}

func (v *buildRouteParamsValidator) validateRecipient(to string) error {
	if len(to) == 0 {
		return NewValidationError("recipient", "required")
	}

	if !isEthereumAddress(to) {
		return NewValidationError("recipient", "invalid")
	}

	return nil
}

func (v *buildRouteParamsValidator) validatePermit(permit string) error {
	// Return early when permit is empty
	if len(permit) == 0 || permit == constant.EmptyHex {
		return nil
	}

	permitBytes := common.FromHex(permit)

	// The permit can only be empty or 32 * 7 bytes
	// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/974c6c248fd536292c3a9eac7306c62f8bace4da/contracts/dependency/Permitable.sol#L34
	if len(permitBytes) != 0 && len(permitBytes) != constant.PermitBytesLength {
		return NewValidationError("permit", "invalid")
	}

	return nil
}
