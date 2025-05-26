package mkr_sky

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	fee  *big.Int
	rate *big.Int
	gas  int64
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, fmt.Errorf("failed to unmarshal static extra: %w", err)
	}

	numTokens := len(entityPool.Tokens)
	if numTokens != 2 {
		return nil, fmt.Errorf("invalid pool tokens %v, %v", entityPool, numTokens)
	}
	if numTokens != len(entityPool.Reserves) {
		return nil, fmt.Errorf("invalid pool reserves %v, %v", entityPool, numTokens)
	}

	var tokens = make([]string, numTokens)
	var reserves = make([]*big.Int, numTokens)
	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  strings.ToLower(entityPool.Address),
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   tokens,
				Reserves: reserves,
			},
		},
		rate: staticExtra.Rate,
		fee:  big.NewInt(int64(entityPool.SwapFee)),
		gas:  DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenInIndex := p.GetTokenIndex(param.TokenAmountIn.Token)
	tokenOutIndex := p.GetTokenIndex(param.TokenOut)

	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, fmt.Errorf("invalid token indices: in=%d, out=%d", tokenInIndex, tokenOutIndex)
	}

	amountOut := p._MkrToSky(param.TokenAmountIn.Amount)
	if amountOut.Sign() <= 0 {
		return nil, fmt.Errorf("invalid output amount: %s", amountOut.String())
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  param.TokenOut,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  param.TokenAmountIn.Token,
			Amount: bignumber.ZeroBI,
		},
		Gas: p.gas,
	}, nil
}

// _MkrToSky converts MKR amount to SKY amount using the pool's rate and fee
func (p *PoolSimulator) _MkrToSky(mkrAmt *big.Int) *big.Int {
	var skyAmt, tmp big.Int
	skyAmt.Mul(mkrAmt, p.rate)

	if p.fee.Sign() > 0 {
		tmp.Mul(&skyAmt, p.fee)
		tmp.Div(&tmp, WAD)
		skyAmt.Sub(&skyAmt, &tmp)
	}

	return &skyAmt
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {}

func (p *PoolSimulator) CanSwapTo(address string) []string {
	if strings.EqualFold(address, SkyAddress) {
		return []string{MkrAddress}
	}
	return []string{}
}

func (p *PoolSimulator) CanSwapFrom(address string) []string {
	if strings.EqualFold(address, SkyAddress) {
		return []string{}
	}
	return []string{MkrAddress}
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return struct {
		BlockNumber uint64 `json:"blockNumber"`
	}{
		BlockNumber: p.Info.BlockNumber,
	}
}
