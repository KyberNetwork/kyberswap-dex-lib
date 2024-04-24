//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple Oracle
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt

package ezeth

import "math/big"

type Oracle struct {
	RoundId         *big.Int `json:"-"`
	Answer          *big.Int `json:"answer"`
	StartedAt       *big.Int `json:"-"`
	UpdatedAt       *big.Int `json:"updatedAt"`
	AnsweredInRound *big.Int `json:"-"`
}

func (o *Oracle) LatestRoundData() (*big.Int, *big.Int) {
	return o.Answer, o.UpdatedAt
}
