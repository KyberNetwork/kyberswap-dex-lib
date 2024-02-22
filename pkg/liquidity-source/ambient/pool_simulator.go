package ambient

import (
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
}

func (s *PoolSimulator) CalcAmountOut(
	param poolpkg.CalcAmountOutParams,
) (*poolpkg.CalcAmountOutResult, error) {
	return nil, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {

}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}
