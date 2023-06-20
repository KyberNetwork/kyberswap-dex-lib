package lido

const (
	DexTypeLido = "lido"

	wstETHMethodStEthPerToken  = "stEthPerToken"
	wstETHMethodTokensPerStEth = "tokensPerStEth"

	erc20MethodTotalSupply = "totalSupply"
	erc20MethodBalanceOf   = "balanceOf"

	defaultTokenWeight = 1
	reserveZero        = "0"
)

var DefaultGas = Gas{Wrap: 50000, Unwrap: 50000}
