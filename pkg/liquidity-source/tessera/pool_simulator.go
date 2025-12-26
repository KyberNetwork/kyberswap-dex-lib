package tessera

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	extra Extra
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	tokens := make([]string, len(entityPool.Tokens))
	for i, t := range entityPool.Tokens {
		tokens[i] = strings.ToLower(t.Address)
	}

	reserves := make([]*big.Int, len(entityPool.Reserves))
	for i, r := range entityPool.Reserves {
		reserves[i], _ = new(big.Int).SetString(r, 10)
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:     entityPool.Address,
				Exchange:    entityPool.Exchange,
				Type:        entityPool.Type,
				Tokens:      tokens,
				Reserves:    reserves,
				BlockNumber: entityPool.BlockNumber,
			},
		},
		extra: extra,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := params.TokenAmountIn
	tokenOut := strings.ToLower(params.TokenOut)
	tokenIn := strings.ToLower(tokenAmountIn.Token)

	if s.GetTokenIndex(tokenIn) < 0 || s.GetTokenIndex(tokenOut) < 0 {
		return nil, ErrInvalidToken
	}

	if !s.extra.TradingEnabled {
		return nil, ErrTradingDisabled
	}

	if !s.extra.IsInitialised {
		return nil, ErrNotInitialised
	}

	var isBaseToQuote bool
	if strings.EqualFold(tokenIn, s.Info.Tokens[0]) {
		isBaseToQuote = true
	}

	amountInRaw := uint256.MustFromBig(tokenAmountIn.Amount)

	var maxAvailable *big.Int
	if isBaseToQuote {
		maxAvailable = s.Info.Reserves[0]
	} else {
		maxAvailable = s.Info.Reserves[1]
	}

	// Now only support swaps up to this limit with high accuracy.
	// Quoter may accept larger amounts but interpolation has no data points beyond this range.
	if tokenAmountIn.Amount.Cmp(maxAvailable) > 0 {
		return nil, ErrSwapReverted
	}

	var amountOut *uint256.Int
	var err error

	if isBaseToQuote {
		amountOut, err = GetClosestRate(amountInRaw, s.extra.BaseToQuotePrefetches)
	} else {
		amountOut, err = GetClosestRate(amountInRaw, s.extra.QuoteToBasePrefetches)
	}

	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: bignumber.ZeroBI,
		},
		Gas: defaultGas,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	tokenIn := params.TokenAmountIn.Token

	if strings.EqualFold(tokenIn, s.Info.Tokens[0]) {
		s.Info.Reserves[0] = new(big.Int).Add(s.Info.Reserves[0], params.TokenAmountIn.Amount)
	} else {
		s.Info.Reserves[1] = new(big.Int).Sub(s.Info.Reserves[1], params.TokenAmountOut.Amount)
	}
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return struct {
		BlockNumber    uint64
		TradingEnabled bool
		IsInitialised  bool
	}{
		BlockNumber:    s.Info.BlockNumber,
		TradingEnabled: s.extra.TradingEnabled,
		IsInitialised:  s.extra.IsInitialised,
	}
}
