package shared

import (
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func GetBufferTokens(req *ethrpc.Request, poolTokens []*entity.PoolToken, bufferTokens []string,
	vaultExplorer string) func() ([]*ExtraBuffer, []common.Address) {
	res := make([]*big.Int, len(bufferTokens))
	underlyingTokens := make([]common.Address, len(bufferTokens))
	for i, bufferToken := range bufferTokens {
		if bufferToken != "" {
			req.AddCall(&ethrpc.Call{
				ABI:    ERC4626ABI,
				Target: bufferToken,
				Method: ERC4626MethodConvertToAssets,
				Params: []any{big.NewInt(1e18)},
			}, []any{&res[i]})
		} else {
			req.AddCall(&ethrpc.Call{
				ABI:    VaultExplorerABI,
				Target: vaultExplorer,
				Method: VaultMethodGetBufferAsset,
				Params: []any{common.HexToAddress(poolTokens[i].Address)},
			}, []any{&underlyingTokens[i]}).AddCall(&ethrpc.Call{
				ABI:    ERC4626ABI,
				Target: poolTokens[i].Address,
				Method: ERC4626MethodConvertToAssets,
				Params: []any{big.NewInt(1e18)},
			}, []any{&res[i]})
		}
	}
	return func() ([]*ExtraBuffer, []common.Address) {
		return lo.Map(res, func(v *big.Int, _ int) *ExtraBuffer {
			if v == nil {
				return nil
			}
			return &ExtraBuffer{
				Rate: uint256.MustFromBig(v),
			}
		}), underlyingTokens
	}
}
