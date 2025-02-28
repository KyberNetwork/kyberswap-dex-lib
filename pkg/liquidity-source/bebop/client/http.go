package client

import (
	"context"
	"errors"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bebop"
)

const (
	querySourceKey      = "source"
	headerSourceAuthKey = "source-auth"

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
		SetHeader(headerSourceAuthKey, config.Authorization)

	return &HTTPClient{
		config: config,
		client: client,
	}
}

func (c *HTTPClient) QuoteSingleOrderResult(ctx context.Context, params bebop.QuoteParams) (bebop.QuoteSingleOrderResult, error) {
	// token address case-sensitive
	req := c.client.R().
		SetContext(ctx).
		// the SellTokens address must follow the HEX format
		SetQueryParam(bebop.ParamsSellTokens, common.HexToAddress(params.SellTokens).Hex()).
		// the BuyTokens address must follow the HEX format
		SetQueryParam(bebop.ParamsBuyTokens, common.HexToAddress(params.BuyTokens).Hex()).
		SetQueryParam(bebop.ParamsSellAmounts, params.SellAmounts).
		SetQueryParam(bebop.ParamsTakerAddress, common.HexToAddress(params.TakerAddress).Hex()).
		SetQueryParam(bebop.ParamsReceiverAddress, common.HexToAddress(params.ReceiverAddress).Hex()).
		SetQueryParam(bebop.ParamsApproveType, "Standard").
		SetQueryParam(bebop.ParamsSkipValidation, "true"). // not checking balance
		SetQueryParam(bebop.ParamsGasLess, "false").       // self-execution
		SetQueryParam(querySourceKey, c.config.Name)

	var result bebop.QuoteSingleOrderResult
	var fail bebop.QuoteFail
	resp, err := req.SetResult(&result).SetError(&fail).Get(pathQuote)
	if err != nil {
		return bebop.QuoteSingleOrderResult{}, err
	}

	respBytes := resp.Body()
	_ = json.Unmarshal(respBytes, &result)
	_ = json.Unmarshal(respBytes, &fail)

	if !resp.IsSuccess() || fail.Failed() {
		return bebop.QuoteSingleOrderResult{}, parseRFQError(fail.Error.ErrorCode, fail.Error.Message)
	}

	return result, nil
}

func parseRFQError(errorCode int, message string) error {
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
		logger.
			WithFields(logger.Fields{"client": "bebop", "errorCode": errorCode, "message": message}).
			Error("rfq failed")
		return ErrRFQFailed
	}
}
