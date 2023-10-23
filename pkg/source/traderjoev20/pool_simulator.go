package traderjoev20

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

type PoolSimulator struct {
	pool.Pool
}

func NewPoolSimulator(entityPool pool.Pool) (*PoolSimulator, error) {
	// TODO: implement this

	return &PoolSimulator{
		Pool: pool.Pool{Info: entityPool.Info},
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	return nil, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {

}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {

	return nil
}
