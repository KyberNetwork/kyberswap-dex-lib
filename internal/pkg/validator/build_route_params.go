package validator

import (
	"context"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/account"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type buildRouteParamsValidator struct {
	nowFunc func() time.Time

	config        BuildRouteParamsConfig
	blackjackRepo IBlackjackRepository
	mu            sync.Mutex
}

func NewBuildRouteParamsValidator(
	nowFunc func() time.Time,
	config BuildRouteParamsConfig,
	blackjackRepo IBlackjackRepository,
) *buildRouteParamsValidator {
	return &buildRouteParamsValidator{
		nowFunc:       nowFunc,
		config:        config,
		blackjackRepo: blackjackRepo,
	}
}

func (v *buildRouteParamsValidator) Validate(ctx context.Context, params params.BuildRouteParams) error {
	wallets := []string{params.Recipient}

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

	if err := v.validateSender(params.Sender, &wallets); err != nil {
		return err
	}

	if err := v.validateRecipient(params.Recipient); err != nil {
		return err
	}

	if err := v.validateWallets(ctx, wallets); err != nil {
		return err
	}

	if err := v.validatePermit(params.Permit); err != nil {
		return err
	}

	return nil
}

func (v *buildRouteParamsValidator) ApplyConfig(config BuildRouteParamsConfig) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.config = config
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

	if !account.IsValidAddress(tokenIn) || account.IsZeroAddress(tokenIn) {
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

	if !account.IsValidAddress(tokenOut) || account.IsZeroAddress(tokenOut) {
		return NewValidationError("tokenOut", "invalid")
	}

	return nil
}

func (v *buildRouteParamsValidator) validateSlippageTolerance(slippageTolerance int64) error {
	if slippageTolerance < v.config.SlippageToleranceGTE || slippageTolerance > v.config.SlippageToleranceLTE {
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

	if !account.IsValidAddress(feeReceiver) || account.IsZeroAddress(feeReceiver) {
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

func (v *buildRouteParamsValidator) validateSender(sender string, wallets *[]string) error {
	if !v.config.FeatureFlags.ShouldValidateSender {
		return nil
	}

	if len(sender) == 0 {
		return NewValidationError("sender", "required")
	}

	if !account.IsValidAddress(sender) || account.IsZeroAddress(sender) {
		return NewValidationError("sender", "invalid")
	}

	*wallets = append(*wallets, sender)

	return nil
}

func (v *buildRouteParamsValidator) validateRecipient(recipient string) error {
	if len(recipient) == 0 {
		return NewValidationError("recipient", "required")
	}

	if !account.IsValidAddress(recipient) || account.IsZeroAddress(recipient) {
		return NewValidationError("recipient", "invalid")
	}

	return nil
}

func (v *buildRouteParamsValidator) validateWallets(ctx context.Context, wallets []string) error {
	if !v.config.FeatureFlags.IsBlackjackEnabled {
		return nil
	}

	checkResult, err := v.blackjackRepo.Check(ctx, wallets)
	if err != nil {
		logger.
			WithFields(logger.Fields{"request_id": requestid.GetRequestIDFromCtx(ctx), "error": err.Error()}).
			Debug("failed to check from blackjack")
		return nil
	}

	for wallet, isBlacklisted := range checkResult {
		if isBlacklisted {
			logger.
				WithFields(logger.Fields{"wallet": wallet, "request_id": requestid.GetRequestIDFromCtx(ctx)}).
				Info("blacklisted wallet")

			return NewValidationError("wallets", "invalid")
		}
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
