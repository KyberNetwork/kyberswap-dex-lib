//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple GlpManager
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt

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
