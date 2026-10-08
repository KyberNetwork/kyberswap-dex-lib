package crystalob

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// PoolSimulator is one Crystal market: a limit order book plus an optional constant-product AMM.
// Tokens are [quote, base]; a buy is quote -> base and walks the asks.
type PoolSimulator struct {
	pool.Pool
	bids, asks []Level
	rq, rb     *uint256.Int // nil when the AMM is disabled
	*market                 // fixed for this snapshot, shared by clones
}

type market struct {
	scale, tickSize, maxPrice, buyWorst uint256.Int
	takerFee, makerRebate, ammFee       uint256.Int
	router                              string
	hasNative                           bool
}

// No age check: pool-service skips saving (and re-stamping) unchanged state, so a quiet market's
// timestamp ages while its book is still current.
var (
	_                             = pool.RegisterFactory0(DexType, NewPoolSimulator)
	_ pool.IPoolSupportNativeSwap = (*PoolSimulator)(nil)
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}
	if extra.TakerFee == 0 || extra.MakerRebate == 0 || staticExtra.TickSize == nil || staticExtra.TickSize.IsZero() {
		return nil, ErrMarketInactive
	}
	buyWorst, ok := defaultBuyWorstPrice(staticExtra.MarketType, staticExtra.TickSize, staticExtra.MaxPrice)
	if !ok {
		return nil, ErrMarketInactive
	}

	p := &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig(r) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		bids:   extra.Bids,
		asks:   extra.Asks,
		rq:     extra.ReserveQ,
		rb:     extra.ReserveB,
		market: &market{router: staticExtra.Router, hasNative: staticExtra.HasNative},
	}
	if p.rq == nil || p.rb == nil {
		p.rq, p.rb = nil, nil
	}
	p.scale.Set(staticExtra.ScaleFactor)
	p.tickSize.Set(staticExtra.TickSize)
	p.maxPrice.Set(staticExtra.MaxPrice)
	p.buyWorst.Set(buyWorst)
	p.takerFee.SetUint64(extra.TakerFee)
	p.makerRebate.SetUint64(extra.MakerRebate)
	p.ammFee.SetUint64(lo.Ternary[uint64](staticExtra.MarketType == 4, ammFeeFour, ammFeeDefault))
	return p, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn := params.TokenAmountIn.Token
	idxIn, idxOut := p.GetTokenIndex(tokenIn), p.GetTokenIndex(params.TokenOut)
	if idxIn < 0 || idxOut < 0 || idxIn == idxOut {
		return nil, ErrInvalidToken
	}
	if params.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrInvalidAmount
	}
	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrAmountTooLarge
	}

	r, err := p.marketOrder(idxIn == 0, amountIn)
	if err != nil {
		return nil, err
	}
	gas := gasBase + int64(r.levels)*gasPerLevel
	if r.usedAMM {
		gas += gasAMM
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut:         &pool.TokenAmount{Token: params.TokenOut, Amount: r.amountOut.ToBig()},
		Fee:                    &pool.TokenAmount{Token: p.Info.Tokens[0], Amount: r.fee.ToBig()},
		RemainingTokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: amountIn.Sub(amountIn, &r.amountIn).ToBig()},
		Gas:                    gas,
		SwapInfo:               r.si,
	}, nil
}

// UpdateBalance drops consumed levels/orders copy-on-write, so clones keep sharing old slices.
func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}
	levels := lo.Ternary(si.IsBuy, &p.asks, &p.bids)
	rest := (*levels)[si.LevelsDone:]
	if si.OrdersDone > 0 || si.PartialRem != nil {
		lvl := rest[0]
		newLvl := make(Level, 1, len(lvl)-si.OrdersDone)
		newLvl[0] = lvl[0]
		remaining := lvl[1+si.OrdersDone:]
		if si.PartialRem != nil {
			newLvl, remaining = append(newLvl, si.PartialRem), remaining[1:]
		}
		newRest := make([]Level, len(rest))
		copy(newRest, rest)
		newRest[0] = append(newLvl, remaining...)
		rest = newRest
	}
	*levels = rest
	if si.ReserveQ != nil {
		p.rq, p.rb = si.ReserveQ, si.ReserveB
	}
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	return &cloned
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return MetaInfo{ApprovalAddress: p.router, HasNative: p.hasNative, BlockNumber: p.Info.BlockNumber}
}

// Crystal.swap wraps native input (tokenIn == eth) and unwraps native output for weth markets.
func (p *PoolSimulator) SwapReceiveNativeIn(tokenIn, _ string, chainId valueobject.ChainID) bool {
	return p.hasNative && valueobject.IsWrappedNative(tokenIn, chainId)
}

func (p *PoolSimulator) SwapReturnNativeOut(_, tokenOut string, chainId valueobject.ChainID) bool {
	return p.hasNative && valueobject.IsWrappedNative(tokenOut, chainId)
}
