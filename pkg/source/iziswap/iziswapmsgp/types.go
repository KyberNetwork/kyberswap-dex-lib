//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple liquidityPoint limitOrderPoint poolInfo
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt

package iziswapmsgp

import (
	"math/big"
	"unsafe"

	"github.com/izumiFinance/iZiSwap-SDK-go/swap"
)

type LiquidityPoint struct {
	LiqudityDelta *big.Int
	Point         int
}

type LimitOrderPoint struct {
	SellingX *big.Int
	SellingY *big.Int
	Point    int
}

type PoolInfo struct {
	CurrentPoint int
	PointDelta   int
	LeftMostPt   int
	RightMostPt  int
	Fee          int
	Liquidity    *big.Int
	LiquidityX   *big.Int
	Liquidities  []LiquidityPoint
	LimitOrders  []LimitOrderPoint
}

func (p *PoolInfo) AsSdk() swap.PoolInfo {
	return *(*swap.PoolInfo)(unsafe.Pointer(p))
}

func FromSdk(p swap.PoolInfo) PoolInfo {
	return *(*PoolInfo)(unsafe.Pointer(&p))
}
