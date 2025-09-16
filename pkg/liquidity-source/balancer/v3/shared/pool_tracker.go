package shared

import (
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

func GetBufferTokens(req *ethrpc.Request, bufferTokens []string) func() []*ExtraBuffer {
	res := make([][]Rate, len(bufferTokens))
	for i, bufferToken := range bufferTokens {
		if bufferToken != "" {
			res[i] = make([]Rate, len(erc4626.PrefetchAmounts))
			for j, amt := range erc4626.PrefetchAmounts {
				req.AddCall(&ethrpc.Call{
					ABI:    ERC4626ABI,
					Target: bufferToken,
					Method: ERC4626MethodConvertToAssets,
					Params: []any{amt.ToBig()},
				}, []any{&res[i][j].RedeemRate})
				req.AddCall(&ethrpc.Call{
					ABI:    ERC4626ABI,
					Target: bufferToken,
					Method: ERC4626MethodConvertToShares,
					Params: []any{amt.ToBig()},
				}, []any{&res[i][j].DepositRate})
			}
		}
	}
	return func() []*ExtraBuffer {
		return lo.Map(res, func(rates []Rate, i int) *ExtraBuffer {
			if bufferTokens[i] == "" {
				return nil
			}

			var extra = ExtraBuffer{
				DepositRates: make([]*uint256.Int, len(rates)),
				RedeemRates:  make([]*uint256.Int, len(rates)),
			}

			for j, rate := range rates {
				extra.DepositRates[j] = uint256.MustFromBig(rate.DepositRate)
				extra.RedeemRates[j] = uint256.MustFromBig(rate.RedeemRate)
			}

			return &extra
		})
	}
}
