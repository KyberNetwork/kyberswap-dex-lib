package rest

import (
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	aggregatorerrors "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/errors"
	usecasecore "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase/core"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils/eth"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/valueobject"

	"github.com/KyberNetwork/kyberswap-error/pkg/errors"
)

type FindRouteRequest struct {
	TokenIn    string `form:"tokenIn" binding:"required"`
	TokenOut   string `form:"tokenOut" binding:"required"`
	AmountIn   string `form:"amountIn" binding:"required"`
	SaveGas    string `form:"saveGas"`
	Dexes      string `form:"dexes"`
	GasInclude string `form:"gasInclude"`
	GasPrice   string `form:"gasPrice"`
	Debug      string `form:"debug"`
}

type FindEncodedRouteRequest struct {
	FindRouteRequest
	EncodedRequestParams
}

type EncodedRequestParams struct {
	SlippageTolerance string `form:"slippageTolerance"`
	ChargeFeeBy       string `form:"chargeFeeBy"`
	FeeReceiver       string `form:"feeReceiver"`
	IsInBps           string `form:"isInBps"`
	FeeAmount         string `form:"feeAmount"`
	Deadline          string `form:"deadline"`
	To                string `form:"to"`
	ClientData        string `form:"clientData"`
	Referral          string `form:"referral"`
	Permit            string `form:"permit"`
}

func (r *FindEncodedRouteRequest) Validate() *errors.DomainError {
	if err := r.ValidateAmountIn(); err != nil {
		return err
	}

	if err := r.ValidateTokens(); err != nil {
		return err
	}

	if err := r.ValidateTo(); err != nil {
		return err
	}

	if err := r.ValidateFeeReceiver(); err != nil {
		return err
	}

	if err := r.ValidateFeeAmount(); err != nil {
		return err
	}

	if err := r.ValidateChargeFeeBy(); err != nil {
		return err
	}

	if err := r.ValidateSlippageTolerance(); err != nil {
		return err
	}

	if err := r.ValidatePermit(); err != nil {
		return err
	}

	return nil
}

func (r *FindEncodedRouteRequest) ValidateAmountIn() *errors.DomainError {
	amountInBi, ok := new(big.Int).SetString(r.AmountIn, 10)
	if !ok || amountInBi.Cmp(constant.Zero) <= 0 {
		return errors.NewDomainErrorInvalid(nil, "amountIn")
	}

	return nil
}

func (r *FindEncodedRouteRequest) ValidateTokens() *errors.DomainError {
	if strings.EqualFold(r.TokenIn, r.TokenOut) {
		return aggregatorerrors.NewDomainErrTokensAreIdentical()
	}

	return nil
}

func (r *FindEncodedRouteRequest) ValidateTo() *errors.DomainError {
	if r.To == "" {
		return errors.NewDomainErrorRequired(nil, "to")
	}

	if !eth.ValidateAddress(r.To) {
		return errors.NewDomainErrorInvalid(nil, "to")
	}

	return nil
}

func (r *FindEncodedRouteRequest) ValidateFeeReceiver() *errors.DomainError {
	if r.FeeReceiver == "" {
		return nil
	}

	if !eth.ValidateAddress(r.FeeReceiver) {
		return errors.NewDomainErrorInvalid(nil, "feeReceiver")
	}

	return nil
}

func (r *FindEncodedRouteRequest) ValidateFeeAmount() *errors.DomainError {
	if r.FeeAmount == "" {
		return nil
	}

	feeAmountBi, ok := new(big.Int).SetString(r.FeeAmount, 10)
	if !ok || feeAmountBi.Cmp(constant.Zero) <= 0 {
		return errors.NewDomainErrorInvalid(nil, "feeAmount")
	}

	if valueobject.ChargeFeeBy(r.ChargeFeeBy) == valueobject.ChargeFeeByCurrencyIn {
		amountInBi, _ := new(big.Int).SetString(r.AmountIn, 10)

		extraFee := valueobject.ExtraFee{
			IsInBps:     utils.IsEnable(r.IsInBps),
			ChargeFeeBy: valueobject.ChargeFeeBy(r.ChargeFeeBy),
			FeeAmount:   feeAmountBi,
			FeeReceiver: r.FeeReceiver,
		}

		amountInAfterFee := usecasecore.CalcAmountInAfterFee(amountInBi, extraFee)

		if amountInAfterFee.Cmp(constant.Zero) <= 0 {
			return errors.NewDomainErrorInvalid(nil, "feeAmount")
		}
	}

	return nil
}

func (r *FindEncodedRouteRequest) ValidateChargeFeeBy() *errors.DomainError {
	if r.FeeAmount == "" {
		return nil
	}

	if r.ChargeFeeBy != constant.ChargeFeeByCurrencyIn && r.ChargeFeeBy != constant.ChargeFeeByCurrencyOut {
		return errors.NewDomainErrorInvalid(nil, "chargeFeeBy")
	}

	return nil
}

func (r *FindEncodedRouteRequest) ValidateSlippageTolerance() *errors.DomainError {
	if r.SlippageTolerance == "" {
		return nil
	}

	slippageTolerance, err := strconv.ParseInt(r.SlippageTolerance, 10, 64)
	if err != nil {
		return nil
	}

	if slippageTolerance < 0 || slippageTolerance > constant.MaximumSlippage {
		return errors.NewDomainErrorOutOfRange(nil, "slippageTolerance")
	}

	return nil
}

func (r *FindEncodedRouteRequest) ValidatePermit() *errors.DomainError {
	// Return early when permit is empty
	if len(r.Permit) == 0 || r.Permit == constant.EmptyHex {
		return nil
	}

	permitBytes := common.FromHex(r.Permit)

	// The permit can only be empty or 32 * 7 bytes
	// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/974c6c248fd536292c3a9eac7306c62f8bace4da/contracts/dependency/Permitable.sol#L34
	if len(permitBytes) != 0 && len(permitBytes) != constant.PermitBytesLength {
		return errors.NewDomainErrorInvalid(nil, "permit")
	}

	return nil
}
