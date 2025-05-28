package usd0pp

import (
	"math/big"
	"time"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	paused    bool
	startTime int64
	endTime   int64

	gas Gas
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		paused:    extra.Paused,
		startTime: extra.StartTime,
		endTime:   extra.EndTime,
		gas:       defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if s.paused {
		return nil, ErrPoolPaused
	}

	// NOTE: only allow to mint USD0PP from USD0, so tokenIn has to be USD0 and tokenOut has to be USD0++
	if params.TokenAmountIn.Token != s.Pool.Info.Tokens[0] && params.TokenOut != s.Info.Tokens[1] {
		return nil, ErrorInvalidTokenIn
	}

	if params.TokenAmountIn.Amount.Sign() < 0 {
		return nil, ErrorInvalidTokenInAmount
	}

	// assume block.timestamp is current time
	blockTimestamp := time.Now().Unix()

	var amountOut, err = s.mint(params.TokenAmountIn.Amount, blockTimestamp)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: params.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            s.gas.Mint,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(_ pool.UpdateBalanceParams) {

}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) CanSwapTo(token string) []string {
	if token == USD0PP {
		return []string{USD0}
	}
	return []string{}
}

func (s *PoolSimulator) CanSwapFrom(token string) []string {
	if token == USD0 {
		return []string{USD0PP}
	}
	return []string{}
}

// https://etherscan.io/address/0x52fef6a6ad48246a4c74824b9bf39ab26b77094d#code#F1#L184
func (s *PoolSimulator) mint(amountIn *big.Int, blockTimestamp int64) (*big.Int, error) {
	if blockTimestamp < s.startTime {
		return nil, ErrBondNotStarted
	}
	if blockTimestamp >= s.endTime {
		return nil, ErrBondEnded
	}

	amountOut := new(big.Int).Set(amountIn)
	return amountOut, nil
}
