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
	"github.com/KyberNetwork/router-service/internal/pkg/repository/blackjack"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type buildRouteParamsValidator struct {
	nowFunc func() time.Time

	config        BuildRouteParamsConfig
	blackjackRepo blackjack.IBlackjackRepository
	mu            sync.Mutex
}

func NewBuildRouteParamsValidator(
	nowFunc func() time.Time,
	config BuildRouteParamsConfig,
	blackjackRepo blackjack.IBlackjackRepository,
) *buildRouteParamsValidator {
	return &buildRouteParamsValidator{
		nowFunc:       nowFunc,
		config:        config,
		blackjackRepo: blackjackRepo,
	}
}

func (v *buildRouteParamsValidator) Validate(ctx context.Context, params params.BuildRouteParams) error {
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

	if err := v.validateSenderAndRecipient(ctx, params.Source, params.Sender, params.Recipient); err != nil {
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

	if !IsEthereumAddress(tokenIn) {
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

	if !IsEthereumAddress(tokenOut) {
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

	if !IsEthereumAddress(feeReceiver) {
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

func (v *buildRouteParamsValidator) validateSenderAndRecipient(ctx context.Context, clientID, sender, recipient string) error {
	if len(recipient) == 0 {
		return NewValidationError("recipient", "required")
	}

	if !IsEthereumAddress(recipient) {
		return NewValidationError("recipient", "invalid")
	}

	// We will not require `sender` for now.
	// We will monitor this field with client-id, then make the decision later.
	if len(sender) == 0 {
		logger.Warnf("Client-id: %s, sender is empty", clientID)
	} else {
		if !IsEthereumAddress(sender) {
			logger.Warnf("Client-id: %s , sender is not ethereum address: %s", clientID, sender)
			sender = ""
		}
	}

	if v.config.FeatureFlags.IsBlackjackEnabled {
		return v.checkBlacklistedWallet(ctx, sender, recipient)
	}

	if v.config.BlacklistedRecipientSet[strings.ToLower(recipient)] {
		return NewValidationError("recipient", "invalid")
	}

	return nil
}

func (v *buildRouteParamsValidator) checkBlacklistedWallet(ctx context.Context, sender, recipient string) error {
	// Blackjack doesn't allow the wallet is empty
	wallets := []string{recipient}
	if len(sender) != 0 {
		wallets = append(wallets, sender)
	}

	blacklistedWallet, err := v.blackjackRepo.GetAddressBlacklisted(ctx, wallets)
	if err != nil {
		// Blackjack is `nice to have` in Aggregator, so we will bypass it if the request gets error.
		logger.Warnf("[checkBlacklistedWallet] blackjackRepo.GetAddressBlacklisted gets error, wallets: %v, error: %s", wallets, err)
		return nil
	}

	if blacklistedWallet[sender] {
		return NewValidationError("sender", "blacklisted wallet")
	}

	if blacklistedWallet[recipient] {
		return NewValidationError("recipient", "blacklisted wallet")
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
