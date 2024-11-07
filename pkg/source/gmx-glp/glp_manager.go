package gmxglp

import "math/big"

const (
	glpManagerMethodGlp          = "glp"
	glpManagerMethodGetAumInUsdg = "getAumInUsdg"
	erc20MethodTotalSupply       = "totalSupply"
)

type GlpManager struct {
	Address  string
	Glp      string `json:"glp"`
	StakeGlp string `json:"stakeGlp"`

	// getAumInUsdg(true)
	MaximiseAumInUsdg *big.Int `json:"maximiseAumInUsdg"`
	// getAumInUsdg(false)
	NotMaximiseAumInUsdg *big.Int `json:"notMaximiseAumInUsdg"`
	GlpTotalSupply       *big.Int `json:"glpSupply"`
}
