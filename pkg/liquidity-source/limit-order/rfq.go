package limitorder

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
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
	// filled, cancelled or expired since the pool snapshot was taken. Drop it: the executor
	// fills the list in order, so the signed orders after it (fallbacks included) take its share.
	kept := swapInfo.FilledOrders[:0]
	var droppedRequired []int64
	for _, o := range swapInfo.FilledOrders {
		if _, signed := signaturesById[o.OrderID]; signed {
			kept = append(kept, o)
		} else if !o.IsFallBack {
			droppedRequired = append(droppedRequired, o.OrderID)
		}
	}
	swapInfo.FilledOrders = kept

	// nil keeps the quoted amount out; a replaced required order reports the lower one, so
	// the caller's RFQ slippage check decides whether the route still holds.
	var newAmountOut *big.Int
	if len(droppedRequired) > 0 {
		if newAmountOut, err = refillSignedOrders(&swapInfo); err != nil {
			logger.WithFields(logger.Fields{
				"params":   params,
				"orderIds": droppedRequired,
				"error":    err,
			}).Error("missing operator signature for a required order")
			return nil, ErrMissingOperatorSignature
		}
		logger.WithFields(logger.Fields{
			"orderIds":     droppedRequired,
			"newAmountOut": newAmountOut.String(),
		}).Info("replaced unsigned required orders with signed ones")
	}

	return &pool.RFQResult{
		NewAmountOut: newAmountOut,
		Extra: OpSignatureExtra{
			SwapInfo:               swapInfo,
			OperatorSignaturesById: signaturesById,
		},
	}, nil
}

func (h *RFQHandler) BatchRFQ(context.Context, []pool.RFQParams) ([]*pool.RFQResult, error) {
	return nil, nil
}

// refillSignedOrders re-runs calcAmountOutWithSwapInfo's exact-in fill (fillOrder) over
// swapInfo.FilledOrders using the remaining amounts recorded at quote time, rewriting each
// order's fill. Orders past the one that completes amountIn go back to being fallbacks.
// Returns the new amount out.
func refillSignedOrders(swapInfo *SwapInfo) (*big.Int, error) {
	var amountIn uint256.Int
	if amountIn.SetFromDecimal(swapInfo.AmountIn) != nil {
		return nil, ErrInvalidSwapInfo
	}

	var (
		amountOut, remainingMaking, remainingTaking, amountInAfterFee, planned uint256.Int
		terms                                                                  orderTerms
		fill                                                                   orderFill
	)
	// A later order of the same maker may have been capped by that maker's balance at quote
	// time, so making filled beyond the original plan comes off its remaining amount.
	overfilledByMaker := make(map[string]uint256.Int)
	fulfilled := false
	for _, info := range swapInfo.FilledOrders {
		plannedMaking := info.FilledMakingAmount
		info.markFallback()
		if fulfilled {
			continue
		}
		if err := info.loadTerms(&terms, &remainingMaking, &remainingTaking); err != nil {
			return nil, err
		}
		if extra, ok := overfilledByMaker[info.Maker]; ok {
			if _, underflow := remainingMaking.SubOverflow(&remainingMaking, &extra); underflow {
				continue
			}
			big256.MulDivDown(&remainingTaking, &remainingMaking, &terms.taking, &terms.making)
		}
		if remainingMaking.IsZero() || remainingTaking.IsZero() {
			continue
		}
		if terms.amountAfterTakerFee(&amountInAfterFee, &amountIn).IsZero() {
			continue
		}
		done := fillOrder(&fill, &terms, &remainingTaking, &amountInAfterFee)
		if fill.making.IsZero() {
			continue
		}

		amountOut.Add(&amountOut, &fill.making)
		amountOut.Sub(&amountOut, &fill.makerFee)
		info.FilledTakingAmount, info.FilledMakingAmount, info.FeeAmount, info.IsFallBack =
			fill.taking.Dec(), fill.making.Dec(), fill.makerFee.Dec(), false
		if planned.SetFromDecimal(plannedMaking) != nil {
			return nil, fmt.Errorf("%w: order %d filled making %q", ErrInvalidSwapInfo, info.OrderID, plannedMaking)
		} else if fill.making.Gt(&planned) {
			addFilledMakingAmount(overfilledByMaker, info.Maker, planned.Sub(&fill.making, &planned))
		}
		if done {
			fulfilled = true
		} else {
			amountIn.Sub(&amountIn, &fill.spent)
		}
	}
	if !fulfilled {
		return nil, ErrCannotFulfillAmountIn
	}

	return amountToBig(&amountOut), nil
}

// loadTerms loads f's terms into t and its remaining amounts recorded at quote time into
// remainingMaking and remainingTaking. Routes quoted before RemainingMakingAmount existed have
// none and can't be refilled.
func (f *FilledOrderInfo) loadTerms(t *orderTerms, remainingMaking, remainingTaking *uint256.Int) error {
	if t.making.SetFromDecimal(f.MakingAmount) != nil || t.taking.SetFromDecimal(f.TakingAmount) != nil ||
		remainingMaking.SetFromDecimal(f.RemainingMakingAmount) != nil ||
		remainingTaking.SetFromDecimal(f.RemainingTakingAmount) != nil {
		return fmt.Errorf("%w: order %d has no refill amounts", ErrInvalidSwapInfo, f.OrderID)
	}
	if t.making.IsZero() || t.taking.IsZero() {
		return fmt.Errorf("%w: order %d has zero amounts", ErrInvalidSwapInfo, f.OrderID)
	}
	t.feePct.SetUint64(uint64(f.MakerTokenFeePercent))
	t.takerAssetFee = f.IsTakerAssetFee

	return nil
}
