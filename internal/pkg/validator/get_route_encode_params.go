package validator

import (
	"context"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type getRouteEncodeParamsValidator struct {
	nowFunc func() time.Time

	config        GetRouteEncodeParamsConfig
	blackjackRepo IBlackjackRepository
	mu            sync.Mutex
}

func NewGetRouteEncodeParamsValidator(
	nowFunc func() time.Time,
	config GetRouteEncodeParamsConfig,
	blackjackRepo IBlackjackRepository,
) *getRouteEncodeParamsValidator {
	return &getRouteEncodeParamsValidator{
		nowFunc:       nowFunc,
		config:        config,
		blackjackRepo: blackjackRepo,
	}
}

func (v *getRouteEncodeParamsValidator) Validate(ctx context.Context, params params.GetRouteEncodeParams) error {
	if err := v.validateTokens(params.TokenIn, params.TokenOut); err != nil {
		return err
	}

	if err := v.validateTokenIn(params.TokenIn); err != nil {
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

	if err := v.validatePermit(params.Permit); err != nil {
		return err
	}

	if err := v.validateSlippageTolerance(params.SlippageTolerance); err != nil {
		return err
	}

	if err := v.validateDeadline(params.Deadline); err != nil {
		return err
	}

	if err := v.validateGasPrice(params.GasPrice); err != nil {
		return err
	}

	if err := v.validateTo(ctx, params.To); err != nil {
		return err
	}

	if err := v.validateSources(params.Dexes); err != nil {
		return err
	}

	return nil
}

func (v *getRouteEncodeParamsValidator) ApplyConfig(config GetRouteEncodeParamsConfig) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.config = config
}

func (v *getRouteEncodeParamsValidator) validateAmountIn(amountInParams string) error {
	amountInBi, ok := new(big.Int).SetString(amountInParams, 10)
	if !ok || amountInBi.Cmp(constant.Zero) <= 0 {
		return NewValidationError("amountIn", "invalid")
	}

	return nil
}

func (v *getRouteEncodeParamsValidator) validateTokens(tokenIn, tokenOut string) error {
	if strings.EqualFold(tokenIn, tokenOut) {
		return NewValidationError("tokenIn-out", "identical")
	}

	return nil
}

func (v *getRouteEncodeParamsValidator) validateTokenIn(tokenIn string) error {
	if len(tokenIn) == 0 {
		return NewValidationError("tokenIn", "required")
	}

	if !IsEthereumAddress(tokenIn) {
		return NewValidationError("tokenIn", "invalid")
	}

	return nil
}

func (v *getRouteEncodeParamsValidator) validateTokenOut(tokenOut string) error {
	if len(tokenOut) == 0 {
		return NewValidationError("tokenOut", "required")
	}

	if !IsEthereumAddress(tokenOut) {
		return NewValidationError("tokenOut", "invalid")
	}

	return nil
}

func (v *getRouteEncodeParamsValidator) validateTo(ctx context.Context, to string) error {
	if len(to) == 0 {
		return NewValidationError("to", "required")
	}

	if !IsEthereumAddress(to) {
		return NewValidationError("to", "invalid")
	}

	if v.config.FeatureFlags.IsBlackjackEnabled {
		return v.checkBlacklistedWallet(ctx, to)
	}

	if v.config.BlacklistedRecipientSet[strings.ToLower(to)] {
		return NewValidationError("to", "invalid")
	}

	return nil
}

func (v *getRouteEncodeParamsValidator) checkBlacklistedWallet(ctx context.Context, to string) error {
	blacklistedWallet, err := v.blackjackRepo.Check(ctx, []string{to})
	if err != nil {
		logger.
			WithFields(logger.Fields{"request_id": requestid.GetRequestIDFromCtx(ctx), "error": err.Error()}).
			Debug("failed to check from blackjack")
		return nil
	}

	if blacklistedWallet[to] {
		return NewValidationError("to", "blacklisted wallet")
	}

	return nil
}

func (v *getRouteEncodeParamsValidator) validateFeeReceiver(feeReceiver string) error {
	if len(feeReceiver) == 0 {
		return nil
	}

	if !IsEthereumAddress(feeReceiver) {
		return NewValidationError("feeReceiver", "invalid")
	}

	return nil
}

func (v *getRouteEncodeParamsValidator) validateFeeAmount(feeAmount string) error {
	if len(feeAmount) == 0 {
		return nil
	}

	if _, ok := new(big.Int).SetString(feeAmount, 10); !ok {
		return NewValidationError("feeAmount", "invalid")
	}

	return nil
}

func (v *getRouteEncodeParamsValidator) validateChargeFeeBy(chargeFeeBy string, feeAmount string) error {
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

func (v *getRouteEncodeParamsValidator) validateSlippageTolerance(slippageTolerance int64) error {
	if slippageTolerance < v.config.SlippageToleranceGTE || slippageTolerance > v.config.SlippageToleranceLTE {
		return NewValidationError("slippageTolerance", "invalid")
	}

	return nil
}

func (v *getRouteEncodeParamsValidator) validatePermit(permit string) error {
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

func (v *getRouteEncodeParamsValidator) validateGasPrice(gasPriceStr string) error {
	if len(gasPriceStr) == 0 {
		return nil
	}

	_, ok := new(big.Float).SetString(gasPriceStr)
	if !ok {
		return NewValidationError("gasPrice", "invalid")
	}

	return nil
}

func (v *getRouteEncodeParamsValidator) validateDeadline(deadline int64) error {
	if deadline == 0 {
		return nil
	}

	if deadline < v.nowFunc().Unix() {
		return NewValidationError("deadline", "in the past")
	}

	return nil
}

func (v *getRouteEncodeParamsValidator) validateSources(sources string) error {
	dexes := utils.TransformSliceParams(sources)
	for _, dex := range dexes {
		if !valueobject.IsAnExchange(valueobject.Exchange(dex)) {
			return NewValidationError("AvailableSources", "invalid")
		}
	}
	return nil
}
