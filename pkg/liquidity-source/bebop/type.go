package bebop

type QueryParams = string

const (
	ParamsSellTokens       QueryParams = "sell_tokens"
	ParamsBuyTokens        QueryParams = "buy_tokens"
	ParamsSellAmounts      QueryParams = "sell_amounts"
	ParamsBuyAmounts       QueryParams = "buy_amounts"
	ParamsTakerAddress     QueryParams = "taker_address"
	ParamsReceiverAddress  QueryParams = "receiver_address"
	ParamsSource           QueryParams = "source"
	ParamsApproveType      QueryParams = "approval_type"
	ParamsSkipValidation   QueryParams = "skip_validation"
	ParamsBuyTokensRatios  QueryParams = "buy_tokens_ratios"
	ParamsSellTokensRatios QueryParams = "sell_tokens_ratios"
	ParamsGasLess          QueryParams = "gasless"
	ParamsSourceAuth       QueryParams = "source-auth"
)

type QuoteParams struct {
	// The tokens that will be supplied by the taker
	SellTokens string
	// The tokens that will be supplied by the maker
	BuyTokens string
	// The amount of each taker token, order respective to taker_tokens (in wei)
	SellAmounts string
	// The amount of each maker token, order respective to maker_tokens (in wei)
	BuyAmounts string
	// Address which will sign the order
	TakerAddress string
	// Address which will receive the taker tokens. (Defaults to taker_address if not specified)
	ReceiverAddress string
	// Referral partner that will be associated with the quote
	Source string
	// Type of Approval: Standard/Permit/Permit2
	ApprovalType string
	// Ratios of maker tokens to receive for each taker token
	BuyTokensRatios string
	// Ratios of taker tokens to receive for each maker token
	SellTokensRatios string
	// The list of solvers to include
	IncludeSolvers string
}

type TokenResult struct {
	Amount         string  `json:"amount"`
	Decimals       int     `json:"decimals"`
	PriceUsd       float64 `json:"priceUsd"`
	Symbol         string  `json:"symbol"`
	Price          float64 `json:"price"`
	PriceBeforeFee float64 `json:"priceBeforeFee"`
}

type QuoteFail struct {
	Error struct {
		ErrorCode int    `json:"errorCode"`
		Message   string `json:"message"`
	} `json:"error"`
}

func (r QuoteFail) Failed() bool {
	return r.Error.ErrorCode != 0 || r.Error.Message != ""
}

type QuoteSingleOrderResult struct {
	Type         string  `json:"type"`
	Status       string  `json:"status"`
	QuoteID      string  `json:"quoteId"`
	ChainID      int     `json:"chainId"`
	ApprovalType string  `json:"approvalType"`
	NativeToken  string  `json:"nativeToken"`
	Taker        string  `json:"taker"`
	Receiver     string  `json:"receiver"`
	Expiry       int     `json:"expiry"`
	Slippage     float64 `json:"slippage"`
	GasFee       struct {
		Native string  `json:"native"`
		Usd    float64 `json:"usd"`
	} `json:"gasFee"`
	BuyTokens          map[string]TokenResult `json:"buyTokens"`
	SellTokens         map[string]TokenResult `json:"sellTokens"`
	SettlementAddress  string                 `json:"settlementAddress"`
	ApprovalTarget     string                 `json:"approvalTarget"`
	RequiredSignatures []any                  `json:"requiredSignatures"`
	PriceImpact        float64                `json:"priceImpact"`
	Warnings           []any                  `json:"warnings"`
	Tx                 struct {
		To       string `json:"to"`
		Value    string `json:"value"`
		Data     string `json:"data"`
		From     string `json:"from"`
		Gas      int    `json:"gas"`
		GasPrice int64  `json:"gasPrice"`
	} `json:"tx"`
	ToSign struct { // the toSign part uses snake_case
		PartnerID      int    `json:"partner_id"`
		Expiry         int    `json:"expiry"`
		TakerAddress   string `json:"taker_address"`
		MakerAddress   string `json:"maker_address"`
		MakerNonce     string `json:"maker_nonce"`
		TakerToken     string `json:"taker_token"`
		MakerToken     string `json:"maker_token"`
		TakerAmount    string `json:"taker_amount"`
		MakerAmount    string `json:"maker_amount"`
		Receiver       string `json:"receiver"`
		PackedCommands string `json:"packed_commands"`
	} `json:"toSign"`
	OnchainOrderType  string `json:"onchainOrderType"`
	PartialFillOffset int    `json:"partialFillOffset"`
}
