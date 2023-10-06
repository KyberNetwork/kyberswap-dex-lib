package gmxglp

import "math/big"

const (
	glpManagerMethodGlp          = "glp"
	glpManagerMethodGetAumInUsdg = "getAumInUsdg"
	erc20MethodTotalSupply       = "totalSupply"
)

type GlpManager struct {
	// getAumInUsdg(true)
	MaximiseAumInUsdg *big.Int `json:"maximiseAumInUsdg"`
	// getAumInUsdg(false)
	NotMaximiseAumInUsdg *big.Int `json:"notMaximiseAumInUsdg"`
	GlpTotalSupply       *big.Int `json:"glpSupply"`

	Glp string `json:"glp"`
}
