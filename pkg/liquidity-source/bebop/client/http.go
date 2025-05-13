package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/KyberNetwork/kutils/klog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-resty/resty/v2"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bebop"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
)

const (
	pathQuote = "v3/quote"

	errCodeBadRequest             = 101
	errCodeInsufficientLiquidity  = 102
	errCodeGasCalculationError    = 103
	errCodeMinSize                = 104
	errCodeTokenNotSupported      = 105
	errCodeGasExceedsSize         = 106
	errCodeUnexpectedPermitsError = 107
)

var (
	ErrRFQFailed = errors.New("rfq failed")

	ErrRFQBadRequest             = errors.New("rfq: The API request is invalid - incorrect format or missing required fields")
	ErrRFQInsufficientLiquidity  = errors.New("rfq: There is insufficient liquidity to serve the requested trade size for the given tokens")
	ErrRFQGasCalculationError    = errors.New("rfq: There was a failure in calculating the gas estimate for this quotes transaction cost - this can occur when gas is fluctuating wildly")
	ErrRFQMinSize                = errors.New("rfq: User is trying to trade smaller than the minimum acceptable size for the given tokens")
	ErrRFQTokenNotSupported      = errors.New("rfq: The token user is trying to trade is not supported by Bebop at the moment")
	ErrRFQGasExceedsSize         = errors.New("rfq: Execution cost (gas) doesn't cover the trade size")
	ErrRFQUnexpectedPermitsError = errors.New("rfq: Unexpected error when a user approves tokens via Permit or Permit2 signatures")
)

type HTTPClient struct {
	config *bebop.HTTPClientConfig
	client *resty.Client
}

func NewHTTPClient(config *bebop.HTTPClientConfig) *HTTPClient {
	client := resty.New().
		SetBaseURL(config.BaseURL).
		SetTimeout(config.Timeout.Duration).
		SetRetryCount(config.RetryCount).
		SetHeader(bebop.ParamsSourceAuth, config.Authorization)

	return &HTTPClient{
		config: config,
		client: client,
	}
}

func (c *HTTPClient) Quote(ctx context.Context,
	params bebop.QuoteParams) (bebop.QuoteResult, error) {
	req := c.client.R().
		SetContext(ctx).
		SetQueryParam(bebop.ParamsSellTokens, toChecksumHex(params.SellTokens)). // must be checksum-ed
		SetQueryParam(bebop.ParamsBuyTokens, toChecksumHex(params.BuyTokens)).   // must be checksum-ed
		SetQueryParam(bebop.ParamsSellAmounts, params.SellAmounts).
		SetQueryParam(bebop.ParamsTakerAddress, toChecksumHex(params.TakerAddress)).
		SetQueryParam(bebop.ParamsReceiverAddress, toChecksumHex(params.ReceiverAddress)).
		SetQueryParam(bebop.ParamsOriginAddress, toChecksumHex(params.OriginAddress)).
		SetQueryParam(bebop.ParamsApproveType, "Standard").
		SetQueryParam(bebop.ParamsSkipValidation, "true"). // not checking balance
		SetQueryParam(bebop.ParamsGasLess, "false").       // self-execution
		SetQueryParam(bebop.ParamsSource, c.config.Name)

	var result struct {
		bebop.QuoteResult
		bebop.QuoteFail
	}
	resp, err := req.SetResult(&result).SetError(&result).Get(pathQuote)
	if err != nil {
		return bebop.QuoteResult{}, err
	}

	if !resp.IsSuccess() || result.Failed() {
		klog.WithFields(ctx, klog.Fields{
			"rfq.client":        bebop.DexType,
			"rfq.resp":          util.MaxBytesToString(resp.Body(), 256),
			"rfq.status":        resp.StatusCode(),
			"rfq.error.code":    result.Error.ErrorCode,
			"rfq.error.message": result.Error.Message,
		}).Error("quote failed")
		err = parseRFQError(result.Error.ErrorCode)
		return bebop.QuoteResult{}, fmt.Errorf("%w: %s", err, result.Error.Message)
	}

	return result.QuoteResult, nil
}

func toChecksumHex(hex string) string {
	if hex == "" {
		return hex
	}
	return common.HexToAddress(hex).Hex()
}

func parseRFQError(errorCode int) error {
	switch errorCode {
	case errCodeBadRequest:
		return ErrRFQBadRequest
	case errCodeInsufficientLiquidity:
		return ErrRFQInsufficientLiquidity
	case errCodeGasCalculationError:
		return ErrRFQGasCalculationError
	case errCodeMinSize:
		return ErrRFQMinSize
	case errCodeTokenNotSupported:
		return ErrRFQTokenNotSupported
	case errCodeGasExceedsSize:
		return ErrRFQGasExceedsSize
	case errCodeUnexpectedPermitsError:
		return ErrRFQUnexpectedPermitsError
	default:
		return ErrRFQFailed
	}
}
