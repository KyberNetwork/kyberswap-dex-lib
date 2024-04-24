//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple basePool

package meta

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/plain"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/aave"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/base"
	plainoracle "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/plain-oracle"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type basePool struct {
	aave        *aave.AavePool          `msg:"aave,omitempty"`
	base        *base.PoolBaseSimulator `msg:"base,omitempty"`
	plainOracle *plainoracle.Pool       `msg:"plainOracle,omitempty"`
	plain       *plain.PoolSimulator    `msg:"plain,omitempty"`
}

func newBasePool(pool ICurveBasePool) basePool {
	switch p := pool.(type) {
	case *aave.AavePool:
		return basePool{aave: p}
	case *base.PoolBaseSimulator:
		return basePool{base: p}
	case *plainoracle.Pool:
		return basePool{plainOracle: p}
	case *plain.PoolSimulator:
		return basePool{plain: p}
	}
	panic("unreachable")
}

func (b *basePool) get() ICurveBasePool {
	if b.aave != nil {
		return b.aave
	}
	if b.base != nil {
		return b.base
	}
	if b.plainOracle != nil {
		return b.plainOracle
	}
	if b.plain != nil {
		return b.plain
	}
	panic("unreachable")
}

func (b *basePool) GetInfo() pool.PoolInfo           { return b.get().GetInfo() }
func (b *basePool) GetTokenIndex(address string) int { return b.get().GetTokenIndex(address) }
func (b *basePool) GetVirtualPrice() (vPrice *big.Int, D *big.Int, err error) {
	return b.get().GetVirtualPrice()
}
func (b *basePool) GetDy(i int, j int, dx *big.Int, dCached *big.Int) (*big.Int, *big.Int, error) {
	return b.get().GetDy(i, j, dx, dCached)
}
func (b *basePool) CalculateTokenAmount(amounts []*big.Int, deposit bool) (*big.Int, error) {
	return b.get().CalculateTokenAmount(amounts, deposit)
}
func (b *basePool) CalculateWithdrawOneCoin(tokenAmount *big.Int, i int) (*big.Int, *big.Int, error) {
	return b.get().CalculateWithdrawOneCoin(tokenAmount, i)
}
func (b *basePool) AddLiquidity(amounts []*big.Int) (*big.Int, error) {
	return b.get().AddLiquidity(amounts)
}
func (b *basePool) RemoveLiquidityOneCoin(tokenAmount *big.Int, i int) (*big.Int, error) {
	return b.get().RemoveLiquidityOneCoin(tokenAmount, i)
}
