package pandafun

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
)

type PoolSimulator struct {
	pool.Pool

	graduated                  bool
	minTradeSize               *big.Int
	amountInBuyRemainingTokens *big.Int
	liquidity                  *big.Int
	buyFee                     *big.Int
	sellFee                    *big.Int
	sqrtPa                     *big.Int
	sqrtPb                     *big.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

var (
	ErrTradeBelowMin         = errors.New("PandaPool: TRADE_BELOW_MIN")
	ErrInsufficientLiquidity = errors.New("PandaPool: INSUFFICIENT_LIQUIDITY")
	ErrPoolGraduated         = errors.New("PandaPool: GRADUATED")
	ErrInvalidToken          = errors.New("invalid token")
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra

	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  entityPool.Address,
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens: lo.Map(entityPool.Tokens,
					func(item *entity.PoolToken, index int) string { return item.Address }),
				Reserves: lo.Map(entityPool.Reserves,
					func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			},
		},

		graduated:                  extra.Graduated,
		minTradeSize:               extra.MinTradeSize,
		amountInBuyRemainingTokens: extra.AmountInBuyRemainingTokens,
		liquidity:                  extra.Liquidity,
		buyFee:                     extra.BuyFee,
		sellFee:                    extra.SellFee,
		sqrtPa:                     extra.SqrtPa,
		sqrtPb:                     extra.SqrtPb,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if p.graduated {
		return nil, ErrPoolGraduated
	}
	tokenIn := params.TokenAmountIn.Token
	tokenOut := params.TokenOut

	// tokens[0] is baseToken, tokens[1] is pandaToken
	if tokenIn == p.Info.Tokens[0] && tokenOut == p.Info.Tokens[1] {
		return p.getAmountOutBuy(params)
	} else if tokenIn == p.Info.Tokens[1] && tokenOut == p.Info.Tokens[0] {
		return p.getAmountOutSell(params)
	} else {
		return nil, ErrInvalidToken
	}
}

func (p *PoolSimulator) getAmountOutBuy(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	amountIn := params.TokenAmountIn.Amount
	if new(big.Int).Add(amountIn, oneGwei).Cmp(p.minTradeSize) < 0 {
		return nil, ErrTradeBelowMin
	}

	if amountIn.Cmp(p.amountInBuyRemainingTokens) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	fee := mulDiv(amountIn, p.buyFee, FEE_SCALE, ROUNDING_UP)

	var deltaBaseReserve, baseReserveNew big.Int
	deltaBaseReserve.Sub(amountIn, fee)

	baseReserveNew.Add(p.Info.Reserves[0], &deltaBaseReserve)

	sqrtPNew := mulDiv(&baseReserveNew, PRICE_SCALE, p.liquidity, ROUNDING_DOWN)
	sqrtPNew.Add(sqrtPNew, p.sqrtPa)

	if sqrtPNew.Cmp(p.sqrtPb) > 0 {
		sqrtPNew.Set(p.sqrtPb)
	}

	pandaReserveNew := mulDiv(
		p.liquidity,
		new(big.Int).Sub(p.sqrtPb, sqrtPNew),
		new(big.Int).Mul(sqrtPNew, p.sqrtPb),
		ROUNDING_UP,
	)

	amountOut := new(big.Int).Sub(p.Info.Reserves[1], pandaReserveNew)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  p.Info.Tokens[1],
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  p.Info.Tokens[0],
			Amount: fee,
		},
		Gas: defaultGas,
	}, nil
}

func (p *PoolSimulator) getAmountOutSell(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	pandaReserveNew := new(big.Int).Add(
		p.Info.Reserves[1],
		params.TokenAmountIn.Amount,
	)

	sqrtPNew := mulDiv(
		p.liquidity,
		p.sqrtPb,
		new(big.Int).Add(
			new(big.Int).Mul(pandaReserveNew, p.sqrtPb),
			p.liquidity,
		),
		ROUNDING_UP,
	)

	if sqrtPNew.Cmp(p.sqrtPa) < 0 {
		sqrtPNew.Set(p.sqrtPa)
	}

	baseReserveNew := mulDiv(
		p.liquidity,
		new(big.Int).Sub(sqrtPNew, p.sqrtPa),
		PRICE_SCALE,
		ROUNDING_UP,
	)

	if p.Info.Reserves[0].Cmp(baseReserveNew) < 0 {
		return nil, ErrInsufficientLiquidity
	}

	deltaBaseReserve := new(big.Int).Sub(p.Info.Reserves[0], baseReserveNew)
	var tmp big.Int
	if tmp.Add(deltaBaseReserve, oneGwei).Cmp(p.minTradeSize) < 0 {
		return nil, ErrTradeBelowMin
	}

	fee := mulDiv(deltaBaseReserve, p.sellFee, FEE_SCALE, ROUNDING_UP)
	amountOut := new(big.Int).Sub(deltaBaseReserve, fee)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  p.Info.Tokens[0],
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  p.Info.Tokens[0],
			Amount: fee,
		},
		Gas: defaultGas,
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	// Since we don't need sqrtP in calculation, we only need to update new reserves
	tokenIn := params.TokenAmountIn.Token
	tokenOut := params.TokenAmountOut.Token

	if tokenIn == p.Info.Tokens[0] && tokenOut == p.Info.Tokens[1] {
		p.Info.Reserves[0].Add(p.Info.Reserves[0], params.TokenAmountIn.Amount)
		p.Info.Reserves[1].Sub(p.Info.Reserves[1], params.TokenAmountOut.Amount)
	} else if tokenIn == p.Info.Tokens[1] && tokenOut == p.Info.Tokens[0] {
		p.Info.Reserves[0].Sub(p.Info.Reserves[0], params.TokenAmountIn.Amount)
		p.Info.Reserves[1].Add(p.Info.Reserves[1], params.TokenAmountIn.Amount)
	}
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}
