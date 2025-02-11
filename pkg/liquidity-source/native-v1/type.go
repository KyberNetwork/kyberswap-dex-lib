package nativev1

import (
	"github.com/KyberNetwork/logger"
	"github.com/mitchellh/mapstructure"
)

type QuoteParams struct {
	// Chain name, ref: https://docs.native.org/native-dev/integration/swap-api/get-chains
	Chain string `mapstructure:"chain"`
	// Address of the token to be sold.
	TokenIn string `mapstructure:"token_in"`
	// Address of the token to be bought.
	TokenOut string `mapstructure:"token_out"`
	// Amount of token to be sold, in wei unit.
	AmountWei string `mapstructure:"amount_wei"`
	// Address of the user that sells the token_in.
	FromAddress string `mapstructure:"from_address"`
	// Address of the end user that initiated the swap request.
	BeneficiaryAddress string `mapstructure:"beneficiary_address"`
	// Address of the user that receives the token_out. If empty, this address will be the same as from_address.
	ToAddress string `mapstructure:"to_address"`
	// Expiry time in seconds. This variable will be passed on to the market makers to consider when they give the
	// quote. Market makers would be able to give better quote for a shorter expiry time. It will affect the
	// deadlineTimestamp in the response if the quote from that market maker is chosen. If not provided we will just use
	// the default expiry time by the liquidity source.
	ExpiryTime string `mapstructure:"expiry_time"`
	// Number in percent. For example, passing the value 5 means 5%, 0.1 means 0.1% slippage tolerance. By default it's 0.
	Slippage string `mapstructure:"slippage"`
}

func (p *QuoteParams) ToMap() (ret map[string]string) {
	if err := mapstructure.Decode(p, &ret); err != nil {
		logger.WithFields(logger.Fields{"params": p, "error": err}).Error("failed to decode to map")
	}
	return ret
}

type QuoteResult struct {
	// The http status code
	StatusCode int `json:"statusCode"`
	// The error message if any
	Message string `json:"message"`

	// The tx request to be executed by the NativeRouter.
	TxRequest TxRequest `json:"txRequest"`
	// Amount of token to be sold, in wei unit.
	AmountOut string `json:"amountOut"`
	// The offset position (in bytes) of the param amountIn. You can modify this value freely. Will be undefined if the
	// target liquidity pool is not a native pool (non-PMM pool).
	AmountInOffset int `json:"amountInOffset"`
	// The offset position (in bytes) of the param amountOutMinimum. You can modify this value to protect yourself from
	// slippage accordingly. Will be undefined if the target liquidity pool is not a native pool (non-PMM pool).
	AmountOutMinimumOffset int `json:"amountOutMinimumOffset"`
}

type TxRequest struct {
	// The address of the Native router.
	Target string `json:"target"`
	// The raw input data that will be executed by the NativeRouter.
	Calldata string `json:"calldata"`
	// Value    string `json:"value"`
}

type (
	StaticExtra struct {
		MarketMaker string `json:"marketMaker"`
	}

	Extra struct {
		ZeroToOnePriceLevels []PriceLevel `json:"0to1"`
		OneToZeroPriceLevels []PriceLevel `json:"1to0"`
		MinIn0               float64      `json:"min0"`
		MinIn1               float64      `json:"min1"`
		PriceTolerance       uint         `json:"tlrnce,omitempty"`
		ExpirySecs           uint         `json:"exp,omitempty"`
	}
	PriceLevel struct {
		Quote float64 `json:"q"`
		Price float64 `json:"p"`
	}

	SwapInfo struct {
		BaseToken        string `json:"b" mapstructure:"b"`
		BaseTokenAmount  string `json:"bAmt" mapstructure:"bAmt"`
		QuoteToken       string `json:"q" mapstructure:"q"`
		QuoteTokenAmount string `json:"qAmt" mapstructure:"qAmt"`
		MarketMaker      string `json:"mm,omitempty" mapstructure:"mm"`
		ExpirySecs       uint   `json:"exp,omitempty" mapstructure:"exp"`
	}

	Gas struct {
		Quote int64
	}

	MetaInfo struct {
		Timestamp int64 `json:"timestamp"`
	}
)
