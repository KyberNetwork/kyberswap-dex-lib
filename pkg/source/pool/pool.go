package pool

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	errLogSampler = &zerolog.BurstSampler{Burst: 2, Period: 15 * time.Second}
)

type Pool struct {
	Info PoolInfo
}

func (p *Pool) CloneState() IPoolSimulator {
	return nil
}

func (p *Pool) GetInfo() PoolInfo {
	return p.Info
}

func (p *Pool) GetTokens() []string {
	return p.Info.Tokens
}

func (p *Pool) GetReserves() []*big.Int {
	return p.Info.Reserves
}

func (p *Pool) CalculateLimit() map[string]*big.Int {
	return nil
}

func (p *Pool) GetApprovalAddress(tokenIn, tokenOut string) string {
	return ""
}

// CanSwapTo is the base method to get all swappable tokens from a pool by a given token address
// Pools with custom logic should override this method
func (p *Pool) CanSwapTo(address string) []string {
	if p.GetTokenIndex(address) == -1 {
		return nil
	}
	result := make([]string, len(p.Info.Tokens)-1)
	i := 0
	for _, token := range p.Info.Tokens {
		if token != address {
			result[i] = token
			i++
		}
	}

	return result
}

// CanSwapFrom by default just call CanSwapTo assuming the pool is bidirectional.
// Pools with custom logic should override this method
func (p *Pool) CanSwapFrom(address string) []string {
	return p.CanSwapTo(address)
}

func (p *Pool) GetAddress() string {
	return p.Info.Address
}

func (p *Pool) GetExchange() string {
	return p.Info.Exchange
}

func (p *Pool) Equals(other IPoolSimulator) bool {
	return p.GetAddress() == other.GetAddress()
}

func (p *Pool) GetTokenIndex(address string) int {
	return p.Info.GetTokenIndex(address)
}

func (p *Pool) GetType() string {
	return p.Info.Type
}

type CalcAmountOutResult struct {
	TokenAmountOut         *TokenAmount
	Fee                    *TokenAmount
	RemainingTokenAmountIn *TokenAmount
	Gas                    int64
	SwapInfo               any
}

func (r *CalcAmountOutResult) IsValid() bool {
	isRemainingValid := r.RemainingTokenAmountIn == nil || (r.RemainingTokenAmountIn != nil && r.RemainingTokenAmountIn.Amount.Sign() >= 0)
	return r.TokenAmountOut != nil && r.TokenAmountOut.Amount != nil && r.TokenAmountOut.Amount.Sign() > 0 && isRemainingValid
}

type UpdateBalanceParams struct {
	TokenAmountIn  TokenAmount
	TokenAmountOut TokenAmount
	Fee            TokenAmount
	SwapInfo       any

	// Inventory is a reference to a per-request inventory balances.
	// key is tokenAddress, balance is big.Float
	// Must use reference (not copy)
	SwapLimit SwapLimit
}

type PoolInfo struct {
	Address     string
	Exchange    string
	Type        string
	Tokens      []string
	Reserves    []*big.Int
	SwapFee     *big.Int
	BlockNumber uint64
}

func (t *PoolInfo) GetTokenIndex(address string) int {
	for i, poolToken := range t.Tokens {
		if poolToken == address {
			return i
		}
	}
	return -1
}

type CalcAmountOutParams struct {
	TokenAmountIn TokenAmount
	TokenOut      string
	Limit         SwapLimit
}

type CalcAmountInParams struct {
	TokenAmountOut TokenAmount
	TokenIn        string
	Limit          SwapLimit
}

type CalcAmountInResult struct {
	TokenAmountIn           *TokenAmount
	RemainingTokenAmountOut *TokenAmount
	Fee                     *TokenAmount
	Gas                     int64
	SwapInfo                any
}

// CalcAmountOut wraps pool.CalcAmountOut and catch panic
func CalcAmountOut(ctx context.Context, pool IPoolSimulator, tokenAmountIn TokenAmount, tokenOut string,
	limit SwapLimit) (res *CalcAmountOutResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("calcAmountOut panicked: %v", r)
			lg := log.Ctx(ctx).Sample(errLogSampler)
			lg.Warn().Stack().Err(err).Str("pool", pool.GetAddress()).Send()
		}
	}()

	return pool.CalcAmountOut(
		CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
			Limit:         limit,
		})
}

func FromEntity(e entity.Pool) Pool {
	return Pool{PoolInfo{
		Address:     e.Address,
		Exchange:    e.Exchange,
		Type:        e.Type,
		Tokens:      lo.Map(e.Tokens, func(t *entity.PoolToken, _ int) string { return strings.ToLower(t.Address) }),
		Reserves:    lo.Map(e.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig(r) }),
		SwapFee:     big.NewInt(int64(e.SwapFee)),
		BlockNumber: e.BlockNumber,
	}}
}
