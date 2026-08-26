package limitorder

import (
	"context"
	"strings"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type RFQHandler struct {
	pool.RFQHandler
	config *Config
	client *httpClient
}

func NewRFQHandler(config *Config) *RFQHandler {
	client := NewHTTPClientWithConfig(config)
	return &RFQHandler{
		config: config,
		client: client,
	}
}

func (h *RFQHandler) RFQ(ctx context.Context, params pool.RFQParams) (*pool.RFQResult, error) {
	swapInfoBytes, err := json.Marshal(params.SwapInfo)
	if err != nil {
		return nil, err
	}

	var swapInfo SwapInfo
	if err = json.Unmarshal(swapInfoBytes, &swapInfo); err != nil {
		return nil, ErrInvalidSwapInfo
	}

	for _, o := range swapInfo.FilledOrders {
		var receiver = o.Receiver
		if len(receiver) == 0 || strings.EqualFold(receiver, valueobject.ZeroAddress) {
			receiver = o.Maker
		}
		if strings.EqualFold(receiver, params.Recipient) {
			logger.WithFields(logger.Fields{
				"params":  params,
				"orderId": o.OrderID,
				"error":   ErrSameSenderMaker,
			}).Error("rejected")
			return nil, ErrSameSenderMaker
		}
	}

	orderIds := lo.Map(swapInfo.FilledOrders, func(o *FilledOrderInfo, _ int) int64 { return o.OrderID })
	result, err := h.client.GetOpSignatures(ctx, ChainID(h.config.ChainID), orderIds)
	if err != nil {
		logger.WithFields(logger.Fields{
			"params": params,
			"error":  err,
		}).Errorf("failed to get operator signatures")
		return nil, err
	}

	signaturesById := lo.SliceToMap(result,
		func(sig *operatorSignatures) (int64, *operatorSignatures) { return sig.ID, sig })

	// The service only signs orders that are still active, so an id it left out has been
	// filled, cancelled or expired since the pool snapshot was taken. Fallback orders are
	// spare depth the fill does not need, so drop the dead ones instead of failing the whole
	// encode over them; a missing required order still has to fail.
	kept := swapInfo.FilledOrders[:0]
	for _, o := range swapInfo.FilledOrders {
		if _, signed := signaturesById[o.OrderID]; signed {
			kept = append(kept, o)
			continue
		}
		if !o.IsFallBack {
			logger.WithFields(logger.Fields{
				"params":  params,
				"orderId": o.OrderID,
			}).Error("missing operator signature for a required order")
			return nil, ErrMissingOperatorSignature
		}
	}
	swapInfo.FilledOrders = kept

	return &pool.RFQResult{
		NewAmountOut: nil, // at the moment we don't use the new amount out of Limit Order, nil will ignore it
		Extra: OpSignatureExtra{
			SwapInfo:               swapInfo,
			OperatorSignaturesById: signaturesById,
		},
	}, nil
}

func (h *RFQHandler) BatchRFQ(context.Context, []pool.RFQParams) ([]*pool.RFQResult, error) {
	return nil, nil
}
