package maverick

import (
	"encoding/json"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"math/big"
)

type PoolSimulator struct {
	pool.Pool
	state Extra
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var tokens = make([]string, 2)
	var swapFeeFl = new(big.Float).Mul(big.NewFloat(entityPool.SwapFee), bignumber.BoneFloat)
	var swapFee, _ = swapFeeFl.Int(nil)

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  entityPool.Address,
				SwapFee:  swapFee,
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   tokens,
				Checked:  false,
			},
		},
		state: extra,
	}, nil
}

func (t *PoolSimulator) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	return nil, nil
}

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	return
}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return nil
}
