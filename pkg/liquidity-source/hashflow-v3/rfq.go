package hashflowv3

import (
	"context"
	"math/big"

	"github.com/goccy/go-json"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

const rfqDefaultChainType = "evm"

type Config struct {
	ExcludeMarketMakers []string         `mapstructure:"excludeMarketMakers" json:"excludeMarketMakers"`
	HTTP                HTTPClientConfig `mapstructure:"http" json:"http"`
	Router              string           `mapstructure:"router" json:"router"`
}

type IClient interface {
	RFQ(ctx context.Context, params QuoteParams) (QuoteResult, error)
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
	results, err := h.BatchRFQ(ctx, []pool.RFQParams{params})
	if err != nil {
		return nil, err
	}

	return results[0], nil
}

func (h *RFQHandler) BatchRFQ(ctx context.Context, paramsSlice []pool.RFQParams) ([]*pool.RFQResult, error) {
	if len(paramsSlice) == 0 {
		return nil, errors.New("empty batch params")
	}

	quoteParams := QuoteParams{
		BaseChain: Chain{
			ChainType: rfqDefaultChainType,
			ChainId:   paramsSlice[0].NetworkID,
		},
		QuoteChain: Chain{
			ChainType: rfqDefaultChainType,
			ChainId:   paramsSlice[0].NetworkID,
		},
	}

	for _, params := range paramsSlice {
		swapInfoBytes, err := json.Marshal(params.SwapInfo)
		if err != nil {
			return nil, err
		}

		var swapInfo SwapInfo
		if err = json.Unmarshal(swapInfoBytes, &swapInfo); err != nil {
			return nil, err
		}

		quoteParams.RFQs = append(quoteParams.RFQs, RFQ{
			BaseToken:       swapInfo.BaseToken,
			QuoteToken:      swapInfo.QuoteToken,
			BaseTokenAmount: swapInfo.BaseTokenAmount,
			Trader:          params.RFQRecipient,
			EffectiveTrader: params.Recipient,

			// Intentionally not specify marketMakers field to have higher chance to successfully RFQ
			// MarketMakers: []string{swapInfo.MarketMaker},

			ExcludeMarketMakers: h.config.ExcludeMarketMakers,
			Options:             &RFQOptions{DoNotRetryWithOtherMakers: true},
		})
	}

	quoteResult, err := h.client.RFQ(ctx, quoteParams)
	if err != nil {
		return nil, errors.WithMessage(err, "quote failed")
	}

	if len(quoteResult.Quotes) != len(paramsSlice) {
		return nil, errors.New("mismatch quotes length")
	}

	var results []*pool.RFQResult
	for _, quote := range quoteResult.Quotes {
		newAmountOut, _ := new(big.Int).SetString(quote.QuoteData.QuoteTokenAmount, 10)
		if quote.TargetContract == "" {
			quote.TargetContract = h.config.Router
		}

		if quote.QuoteData.BaseToken != eth.AddressZero.String() {
			quote.ApprovalAddress = quote.TargetContract
		}

		results = append(results, &pool.RFQResult{
			NewAmountOut: newAmountOut,
			Extra:        quote,
		})
	}

	return results, nil
}

func (h *RFQHandler) SupportBatch() bool {
	return true
}
