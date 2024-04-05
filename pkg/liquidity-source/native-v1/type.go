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

	// Indicates if the signing of the firm quote was successful.
	Success bool `json:"success"`
	// Array of order objects.
	Orders []struct {
		Id int `json:"id"`
		// Public address of the signer that will sign this order.
		Signer string `json:"signer"`
		// Native pool contract address that will execute this order.
		Buyer string `json:"buyer"`
		// The address that will send seller token to market maker.
		Seller string `json:"seller"`
		// The ERC20 token address will receive from market maker.
		BuyerToken string `json:"buyerToken"`
		// The ERC20 token address will be sent to market maker.
		SellerToken string `json:"sellerToken"`
		// The token output amount of the order. In wei. Can be modified by the signer to determine the final output
		// amount.
		BuyerTokenAmount string `json:"buyerTokenAmount"`
		// The token input amount of the order. In wei.
		SellerTokenAmount string `json:"sellerTokenAmount"`
		Caller            string `json:"caller"`
		// Unique ID for this order request in UUID v5.
		QuoteId string `json:"quoteId"`
		// The expiration time of the order, in block timestamp. Can be modified by the signer to determine the
		// expiration time.
		DeadlineTimestamp int `json:"deadlineTimestamp"`
	} `json:"orders"`
	// Contains the widgetFee details
	WidgetFee struct {
		Signer string `json:"signer"`
		// the address that will receive the widget fee
		FeeRecipient string `json:"feeRecipient"`
		// the amount of fees that the fee recipient will receive.
		FeeRate int `json:"feeRate"`
	} `json:"widgetFee"`
	// The signature of the transaction request and the widget fee.
	WidgetFeeSignature string `json:"widgetFeeSignature"`
	// The address of the seller/swapper.
	Recipient string `json:"recipient"`
	// The byte encoding for the orders object.
	Calldata string `json:"calldata"`
	// The amount of sellerToken in wei, that will be sold to the buyer.
	AmountIn string `json:"amountIn"`
	// The amount of buyerToken in wei, that the buyer will be receiving.
	AmountOut string `json:"amountOut"`
	// Indicates if the order needs to be wrapped.
	ToWrap bool `json:"toWrap"`
	// Indicates if the order needs to be unwrapped.
	ToUnwrap bool `json:"toUnwrap"`
	// This is the raw input data that will be executed by Native fallback.
	FallbackSwapDataArray []string `json:"fallbackSwapDataArray"`
	// Indicates the liquidity provider that is providing this firm-quote.
	Source string `json:"source"`
	// The transaction request to be executed by the NativeRouter
	TxRequest struct {
		// the address of the NativeRouter
		Target   string `json:"target"`
		// contains the raw input data that will be executed by the NativeRouter
		Calldata string `json:"calldata"`
		// native token amount to send
		Value    string `json:"value"`
	} `json:"txRequest"`
}
