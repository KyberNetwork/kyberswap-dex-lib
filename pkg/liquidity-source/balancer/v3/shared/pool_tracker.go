package shared

import (
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

func GetBufferTokens(req *ethrpc.Request, bufferTokens []string) func() []*ExtraBuffer {
	var (
		rates        = make([][]Rate, len(bufferTokens))
		maxDeposits  = make([]*big.Int, len(bufferTokens))
		maxWithdraws = make([]*big.Int, len(bufferTokens))
	)

	for i, bufferToken := range bufferTokens {
		if bufferToken == "" {
			continue
		}

		rates[i] = make([]Rate, len(erc4626.PrefetchAmounts))
		for j, amt := range erc4626.PrefetchAmounts {
			req.AddCall(&ethrpc.Call{
				ABI:    ERC4626ABI,
				Target: bufferToken,
				Method: ERC4626MethodConvertToAssets,
				Params: []any{amt.ToBig()},
			}, []any{&rates[i][j].RedeemRate})
			req.AddCall(&ethrpc.Call{
				ABI:    ERC4626ABI,
				Target: bufferToken,
				Method: ERC4626MethodConvertToShares,
				Params: []any{amt.ToBig()},
			}, []any{&rates[i][j].DepositRate})
		}
		req.AddCall(&ethrpc.Call{
			ABI:    ERC4626ABI,
			Target: bufferToken,
			Method: ERC4626MethodMaxDeposit,
			Params: []any{VaultAddress},
		}, []any{&maxDeposits[i]})
		req.AddCall(&ethrpc.Call{
			ABI:    ERC4626ABI,
			Target: bufferToken,
			Method: ERC4626MethodMaxWithdraw,
			Params: []any{VaultAddress},
		}, []any{&maxWithdraws[i]})
	}

	return func() []*ExtraBuffer {
		return lo.Map(rates, func(rates []Rate, i int) *ExtraBuffer {
			if bufferTokens[i] == "" {
				return nil
			}

			var extra = ExtraBuffer{
				DepositRates: make([]*uint256.Int, len(rates)),
				RedeemRates:  make([]*uint256.Int, len(rates)),
				MaxDeposit:   uint256.MustFromBig(maxDeposits[i]),
				MaxWithdraw:  uint256.MustFromBig(maxWithdraws[i]),
			}

			for j, rate := range rates {
				extra.DepositRates[j] = uint256.MustFromBig(rate.DepositRate)
				extra.RedeemRates[j] = uint256.MustFromBig(rate.RedeemRate)
			}

			return &extra
		})
	}
}
