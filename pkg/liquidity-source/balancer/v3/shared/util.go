package shared

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func FromBig(v *big.Int, _ int) *uint256.Int {
	return big256.FromBig(v)
}

func FromBigs(v []*big.Int) []*uint256.Int {
	return lo.Map(v, FromBig)
}

func BatchRouter(chainID valueobject.ChainID, exchange string) (common.Address, bool) {
	prefix := strings.SplitN(exchange, "-", 2)[0]
	router, ok := BatchRouterMap[prefix]
	if !ok {
		router = BatchRouterMap[""]
	}
	v, ok := router[chainID]
	return v, ok
}

func Vault(chainID valueobject.ChainID, exchange string) common.Address {
	prefix := strings.SplitN(exchange, "-", 2)[0]
	if override, ok := VaultOverrideMap[prefix][chainID]; ok {
		return override
	}
	vault, ok := VaultMap[prefix]
	if !ok {
		vault = VaultMap[""]
	}
	return vault
}

var (
	feeMask  = uint256.NewInt(1<<24 - 1)
	feeScale = uint256.NewInt(1e11) // FEE_SCALING_FACTOR
)

const aggregateSwapFeeOffset = 42 // PoolConfigConst.AGGREGATE_SWAP_FEE_OFFSET

// AggregateSwapFee returns the explorer's aggregate swap fee, or decodes it from poolConfigBits when the
// explorer call failed (no VaultExplorer deployed, so the Vault is used as the explorer).
func (r *RpcResult) AggregateSwapFee() *uint256.Int {
	if r.AggregateSwapFeePercentage != nil {
		v, _ := uint256.FromBig(r.AggregateSwapFeePercentage)
		return v
	}
	v := new(uint256.Int).SetBytes32(r.PoolData.PoolConfigBits[:])
	v.Rsh(v, aggregateSwapFeeOffset)
	v.And(v, feeMask)
	return v.Mul(v, feeScale)
}
