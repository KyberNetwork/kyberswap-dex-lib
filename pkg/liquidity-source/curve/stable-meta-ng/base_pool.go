//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple basePool

package stablemetang

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/plain"
	stableng "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/stable-ng"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/holiman/uint256"
)

type basePool struct {
	plain  *plain.PoolSimulator    `msg:"plain,omitempty"`
	stable *stableng.PoolSimulator `msg:"stable,omitempty"`
	meta   *PoolSimulator          `msg:"meta,omitempty"`
}

func newBasePool(pool ICurveBasePool) basePool {
	switch p := pool.(type) {
	case *plain.PoolSimulator:
		return basePool{plain: p}
	case *stableng.PoolSimulator:
		return basePool{stable: p}
	case *PoolSimulator:
		return basePool{meta: p}
	}
	panic("unreachable")
}

func (b *basePool) get() ICurveBasePool {
	if b.plain != nil {
		return b.plain
	}
	if b.stable != nil {
		return b.stable
	}
	if b.meta != nil {
		return b.meta
	}
	panic("unreachable")
}

func (b *basePool) GetInfo() pool.PoolInfo           { return b.get().GetInfo() }
func (b *basePool) GetTokenIndex(address string) int { return b.get().GetTokenIndex(address) }
func (b *basePool) GetVirtualPriceU256(vPrice *uint256.Int, D *uint256.Int) error {
	return b.get().GetVirtualPriceU256(vPrice, D)
}
func (b *basePool) CalculateTokenAmountU256(amounts []uint256.Int, deposit bool, mintAmount *uint256.Int, feeAmounts []uint256.Int) error {
	return b.get().CalculateTokenAmountU256(amounts, deposit, mintAmount, feeAmounts)
}
func (b *basePool) CalculateWithdrawOneCoinU256(tokenAmount *uint256.Int, i int, dy *uint256.Int, dyFee *uint256.Int) error {
	return b.get().CalculateWithdrawOneCoinU256(tokenAmount, i, dy, dyFee)
}
func (b *basePool) ApplyRemoveLiquidityOneCoinU256(i int, tokenAmount, dy, dyFee *uint256.Int) error {
	return b.get().ApplyRemoveLiquidityOneCoinU256(i, tokenAmount, dy, dyFee)
}
func (b *basePool) ApplyAddLiquidity(amounts, feeAmounts []uint256.Int, mintAmount *uint256.Int) error {
	return b.get().ApplyAddLiquidity(amounts, feeAmounts, mintAmount)
}
