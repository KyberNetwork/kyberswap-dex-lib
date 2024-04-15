package nativev1

type QueryParams = string

const (
	ParamsChain              QueryParams = "chain"
	ParamsTokenIn            QueryParams = "token_in"
	ParamsTokenOut           QueryParams = "token_out"
	ParamsAmountWei          QueryParams = "amount_wei"
	ParamsFromAddress        QueryParams = "from_address"
	ParamsBeneficiaryAddress QueryParams = "beneficiary_address"
	ParamsToAddress          QueryParams = "to_address"
	ParamsExpiryTime         QueryParams = "expiry_time"
	ParamsSlippage           QueryParams = "slippage"
)

type QuoteParams struct {
	// The unique identifier that identifies the blockchain.
	ChainID uint
	// Address of the token to be sold.
	TokenIn string
	// Address of the token to be bought.
	TokenOut string
	// Amount of token to be sold, in wei unit.
	AmountWei string
	// Address of the user that sells the token_in.
	FromAddress string
	// Address of the end user that initiated the swap request.
	BeneficiaryAddress string
	// Address of the user that receives the token_out. If empty, this address will be the same as from_address.
	ToAddress string
	// Expiry time in seconds. This variable will be passed on to the market makers to consider when they give the
	// quote. Market makers would be able to give better quote for a shorter expiry time. It will affect the
	// deadlineTimestamp in the response if the quote from that market maker is chosen. If not provided we will just use
	// the default expiry time by the liquidity source.
	ExpiryTime string
	// Number in percent. For example, passing the value 5 means 5%, 0.1 means 0.1% slippage tolerance. By default it's 0.
	Slippage string
}

type QuoteResult struct {
	// The http status code
	StatusCode int `json:"statusCode"`
	// The error message if any
	Message string `json:"message"`

	// // The address that will send the calldata to the Native router.
	// From string `json:"from"`
	// The address of the Native router.
	To string `json:"to"`
	// The raw input data that will be executed by the NativeRouter.
	Calldata string `json:"calldata"`
	// // The msg.value for the transaction. Will be 0 if the seller token is a non-native token.
	// Value string `json:"value"`
	// Amount of token to be sold, in wei unit.
	AmountOut string `json:"amountOut"`
	// The offset position (in bytes) of the param amountIn. You can modify this value freely. Will be undefined if the
	// target liquidity pool is not a native pool (non-PMM pool).
	AmountInOffset int `json:"amountInOffset"`
	// The offset position (in bytes) of the param amountOutMinimum. You can modify this value to protect yourself from
	// slippage accordingly. Will be undefined if the target liquidity pool is not a native pool (non-PMM pool).
	AmountOutMinimumOffset int `json:"amountOutMinimumOffset"`
}
