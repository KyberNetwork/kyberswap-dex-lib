package validator

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type getRouteEncodeParamsValidator struct {
	config GetRouteEncodeParamsConfig
}

func NewGetRouteEncodeParamsValidator(
	config GetRouteEncodeParamsConfig,
) *getRouteEncodeParamsValidator {
	return &getRouteEncodeParamsValidator{
		config: config,
	}
}

func (r *getRouteEncodeParamsValidator) Validate(params params.GetRouteEncodeParams) error {
	if err := r.validateTokens(params.TokenIn, params.TokenOut); err != nil {
		return err
	}

	if err := r.validateTokenIn(params.TokenIn); err != nil {
		return err
	}

	if err := r.validateTokenOut(params.TokenOut); err != nil {
		return err
	}

	if err := r.validateAmountIn(params.AmountIn); err != nil {
		return err
	}

	if err := r.validateFeeReceiver(params.FeeReceiver); err != nil {
		return err
	}

	if err := r.validateFeeAmount(params.FeeAmount); err != nil {
		return err
	}

	if err := r.validateChargeFeeBy(params.ChargeFeeBy, params.FeeAmount); err != nil {
		return err
	}

	if err := r.validatePermit(params.Permit); err != nil {
		return err
	}

	if err := r.validateTo(params.To); err != nil {
		return err
	}

	if err := r.validateSlippageTolerance(params.SlippageTolerance); err != nil {
		return err
	}

	if err := r.validateGasPrice(params.GasPrice); err != nil {
		return err
	}

	return nil
}

func (r *getRouteEncodeParamsValidator) validateAmountIn(amountInParams string) error {
	amountInBi, ok := new(big.Int).SetString(amountInParams, 10)
	if !ok || amountInBi.Cmp(constant.Zero) <= 0 {
		return NewValidationError("amountIn", "invalid")
	}

	return nil
}

func (r *getRouteEncodeParamsValidator) validateTokens(tokenIn, tokenOut string) error {
	if strings.EqualFold(tokenIn, tokenOut) {
		return NewValidationError("tokenIn-out", "identical")
	}

	return nil
}

func (r *getRouteEncodeParamsValidator) validateTokenIn(tokenIn string) error {
	if len(tokenIn) == 0 {
		return NewValidationError("tokenIn", "required")
	}

	if !isEthereumAddress(tokenIn) {
		return NewValidationError("tokenIn", "invalid")
	}

	return nil
}

func (r *getRouteEncodeParamsValidator) validateTokenOut(tokenOut string) error {
	if len(tokenOut) == 0 {
		return NewValidationError("tokenOut", "required")
	}

	if !isEthereumAddress(tokenOut) {
		return NewValidationError("tokenOut", "invalid")
	}

	return nil
}

func (r *getRouteEncodeParamsValidator) validateTo(to string) error {
	if len(to) == 0 {
		return NewValidationError("to", "required")
	}

	if !isEthereumAddress(to) {
		return NewValidationError("to", "invalid")
	}
	return nil
}

func (r *getRouteEncodeParamsValidator) validateFeeReceiver(feeReceiver string) error {
	if len(feeReceiver) == 0 {
		return nil
	}

	if !isEthereumAddress(feeReceiver) {
		return NewValidationError("feeReceiver", "invalid")
	}

	return nil
}

func (r *getRouteEncodeParamsValidator) validateFeeAmount(feeAmount string) error {
	if len(feeAmount) == 0 {
		return nil
	}

	if _, ok := new(big.Int).SetString(feeAmount, 10); !ok {
		return NewValidationError("feeAmount", "invalid")
	}

	return nil
}

func (r *getRouteEncodeParamsValidator) validateChargeFeeBy(chargeFeeBy string, feeAmount string) error {
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

func (r *getRouteEncodeParamsValidator) validateSlippageTolerance(slippageTolerance int64) error {
	if slippageTolerance < r.config.SlippageToleranceGTE || slippageTolerance > r.config.SlippageToleranceLTE {
		return NewValidationError("slippageTolerance", "invalid")
	}

	return nil
}

func (r *getRouteEncodeParamsValidator) validatePermit(permit string) error {
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

func (r *getRouteEncodeParamsValidator) validateGasPrice(gasPriceStr string) error {
	if len(gasPriceStr) == 0 {
		return nil
	}

	_, ok := new(big.Float).SetString(gasPriceStr)
	if !ok {
		return NewValidationError("gasPrice", "invalid")
	}

	return nil
}
