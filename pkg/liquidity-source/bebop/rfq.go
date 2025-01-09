package bebop

import (
	"context"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/mitchellh/mapstructure"
)

type IClient interface {
	QuoteSingleOrderResult(ctx context.Context, params QuoteParams) (QuoteSingleOrderResult, error)
}

type RFQHandler struct {
	pool.RFQHandler
	config *Config
	client IClient
}

func NewRFQHandler(config *Config, client IClient) *RFQHandler {
	return &RFQHandler{
		config: config,
		client: client,
	}
}

func (h *RFQHandler) RFQ(ctx context.Context, params pool.RFQParams) (*pool.RFQResult, error) {
	var swapInfo SwapInfo
	if err := mapstructure.WeakDecode(params.SwapInfo, &swapInfo); err != nil {
		return nil, err
	}
	logger.Infof("params.SwapInfo: %v -> swapInfo: %v", params.SwapInfo, swapInfo)
	p := QuoteParams{
		SellTokens:  swapInfo.BaseToken,
		BuyTokens:   swapInfo.QuoteToken,
		SellAmounts: swapInfo.BaseTokenAmount,
		// BuyAmounts:   not used,
		TakerAddress:    params.RFQSender,
		ReceiverAddress: params.RFQRecipient,
	}
	result, err := h.client.QuoteSingleOrderResult(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("quote single order result: %w", err)
	}

	newAmountOut, err := getAmountOutFromToSign(result.OnchainOrderType, result.ToSign)
	if err != nil {
		return nil, fmt.Errorf("get amount out from to sign: %w", err)
	}

	return &pool.RFQResult{
		NewAmountOut: newAmountOut,
		Extra:        result,
	}, nil
}

func getAmountOutFromToSign(onchainOrderType string, rawTxSign json.RawMessage) (*big.Int, error) {
	switch onchainOrderType {
	case OnchainOrderTypeSingleOrder:
		return getAmountOutOfSingleOrderToSign(rawTxSign)
	case OnchainOrderTypeAggregateOrder:
		return getAmountOutOfAggregateOrderToSign(rawTxSign)
	case OnchainOrderTypeOrderWithPermit2:
		return getAmountOutOfOrderWithPermit2ToSign(rawTxSign)
	case OnchainOrderTypeOrderWithBatchPermit2:
		return getAmountOutOfOrderWithBatchPermit2ToSign(rawTxSign)
	default:
		return nil, fmt.Errorf("unsupported onchain order type: %s", onchainOrderType)
	}
}

func getAmountOutOfSingleOrderToSign(rawTxSign json.RawMessage) (*big.Int, error) {
	var toSign SingleOrderToSign
	if err := json.Unmarshal(rawTxSign, &toSign); err != nil {
		return nil, fmt.Errorf("unmarshal single order result: %w", err)
	}
	amountOut, ok := new(big.Int).SetString(toSign.MakerAmount, 10)
	if !ok {
		return nil, fmt.Errorf("invalid maker amount: %s", toSign.MakerAmount)
	}
	return amountOut, nil
}

func getAmountOutOfAggregateOrderToSign(rawTxSign json.RawMessage) (*big.Int, error) {
	var toSign AggregateOrderToSign
	if err := json.Unmarshal(rawTxSign, &toSign); err != nil {
		return nil, fmt.Errorf("unmarshal aggregate order result: %w", err)
	}

	// With the aggregate order, it has some fields with format:
	// - TakerAddress: common.Address
	// - MakerAddress: [m]common.Address
	// - TakerTokens: [m][n]common.Address
	// - MakerTokens: [m][n]common.Address
	// - TakerAmounts: [m][n]*big.Int
	// - MakerAmounts: [m][n]*big.Int
	// With m is number of makers and n is number of swap token pairs.
	// Because we currently only support swap 1-1 token pair so n is always 1.
	// So we can simplify the check by only checking the first element of each field in the logic below.

	totalAmountOut := big.NewInt(0)

	for _, amounts := range toSign.MakerAmounts {
		if len(amounts) != 1 {
			return nil, fmt.Errorf("invalid maker amounts: %v", amounts)
		}
		amountOut, ok := new(big.Int).SetString(amounts[0], 10)
		if !ok {
			return nil, fmt.Errorf("invalid maker amount: %s", amounts[0])
		}
		totalAmountOut.Add(totalAmountOut, amountOut)
	}

	return totalAmountOut, nil
}

func getAmountOutOfOrderWithPermit2ToSign(rawTxSign json.RawMessage) (*big.Int, error) {
	var toSign OrderWithPermit2ToSign
	if err := json.Unmarshal(rawTxSign, &toSign); err != nil {
		return nil, fmt.Errorf("unmarshal order with permit2 result: %w", err)
	}
	amountOut, ok := new(big.Int).SetString(toSign.Witness.MakerAmount, 10)
	if !ok {
		return nil, fmt.Errorf("invalid maker amount: %s", toSign.Witness.MakerAmount)
	}
	return amountOut, nil
}

func getAmountOutOfOrderWithBatchPermit2ToSign(rawTxSign json.RawMessage) (*big.Int, error) {
	var toSign OrderWithBatchPermit2ToSign
	if err := json.Unmarshal(rawTxSign, &toSign); err != nil {
		return nil, fmt.Errorf("unmarshal order with batch permit2 result: %w", err)
	}

	// logic here same as getAmountOutOfAggregateOrderToSign
	totalAmountOut := big.NewInt(0)
	for _, amounts := range toSign.Witness.MakerAmounts {
		if len(amounts) != 1 {
			return nil, fmt.Errorf("invalid maker amounts: %v", amounts)
		}
		amountOut, ok := new(big.Int).SetString(amounts[0], 10)
		if !ok {
			return nil, fmt.Errorf("invalid maker amount: %s", amounts[0])
		}
		totalAmountOut.Add(totalAmountOut, amountOut)
	}

	return totalAmountOut, nil
}

func (h *RFQHandler) BatchRFQ(context.Context, []pool.RFQParams) ([]*pool.RFQResult, error) {
	return nil, nil
}
