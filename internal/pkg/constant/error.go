package constant

const (
	DomainErrCodeTokensAreIdentical = "DOMAIN:TOKENS_ARE_IDENTICAL"
	DomainErrMsgTokensAreIdentical  = "Domain error: tokenIn and tokenOut are identical"

	ClientErrCodeTokensAreIdentical = 40010
	ClientErrMsgTokensAreIdentical  = "tokenIn and tokenOut are identical"

	ClientErrCodeDeadlineIsInThePast = 40011
	ClientErrMsgDeadlineIsInThePast  = "deadline is in the past"

	ClientErrCodeCouldNotFindRoute = 50010
	ClientErrMsgCouldNotFindRoute  = "could not find route"
)
