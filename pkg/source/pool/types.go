package pool

type CalcAmountInParams struct {
	TokenAmountOut TokenAmount
	TokenIn        string
	Limit          SwapLimit
}

type CalcAmountInResult struct {
	TokenAmountIn *TokenAmount
	Fee           *TokenAmount
	Gas           int64
	SwapInfo      interface{}
}

type CalcAmountOutParams struct {
	TokenAmountIn TokenAmount
	TokenOut      string
	Limit         SwapLimit
}

type CalcAmountOutResult struct {
	TokenAmountOut *TokenAmount
	Fee            *TokenAmount
	Gas            int64
	SwapInfo       interface{}
}
