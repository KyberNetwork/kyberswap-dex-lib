package poolfactory

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/msgpack/v5"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func init() {
	if err := msgpack.RegisterConcreteType(&UnimplementedPool{}); err != nil {
		panic(err)
	}
}

type UnimplementedPool struct {
	address  string
	exchange string
	dexType  string
}

func NewUnimplementedPool(address, exchange, dexType string) *UnimplementedPool {
	return &UnimplementedPool{
		address:  address,
		exchange: exchange,
		dexType:  dexType,
	}
}

func (p *UnimplementedPool) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	return nil, fmt.Errorf("unimplemented address=%s exchange=%s dexType=%s", p.address, p.exchange, p.dexType)
}
func (*UnimplementedPool) UpdateBalance(params pool.UpdateBalanceParams)    {}
func (*UnimplementedPool) CanSwapTo(address string) []string                { return nil }
func (*UnimplementedPool) CanSwapFrom(address string) []string              { return nil }
func (*UnimplementedPool) GetTokens() []string                              { return nil }
func (*UnimplementedPool) GetReserves() []*big.Int                          { return nil }
func (p *UnimplementedPool) GetAddress() string                             { return p.address }
func (p *UnimplementedPool) GetExchange() string                            { return p.exchange }
func (p *UnimplementedPool) GetType() string                                { return p.dexType }
func (*UnimplementedPool) GetMetaInfo(tokenIn, tokenOut string) interface{} { return nil }
func (*UnimplementedPool) GetTokenIndex(address string) int                 { return 0 }
func (*UnimplementedPool) CalculateLimit() map[string]*big.Int              { return nil }

var _ pool.IPoolSimulator = &UnimplementedPool{}
