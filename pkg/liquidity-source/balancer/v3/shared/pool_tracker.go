package shared

import (
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

func GetBufferTokens(req *ethrpc.Request, bufferTokens []string) func() []*ExtraBuffer {
	res := make([]*big.Int, len(bufferTokens))
	for i, bufferToken := range bufferTokens {
		if bufferToken != "" {
			req.AddCall(&ethrpc.Call{
				ABI:    ERC4626ABI,
				Target: bufferToken,
				Method: ERC4626MethodConvertToAssets,
				Params: []any{big.NewInt(1e18)},
			}, []any{&res[i]})
		}
	}
	return func() []*ExtraBuffer {
		return lo.Map(res, func(v *big.Int, _ int) *ExtraBuffer {
			if v == nil {
				return nil
			}
			return &ExtraBuffer{
				Rate: uint256.MustFromBig(v),
			}
		})
	}
}
