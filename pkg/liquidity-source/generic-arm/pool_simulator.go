package genericarm

import (
	"math/big"
	"slices"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	TradeRate0         *uint256.Int
	TradeRate1         *uint256.Int
	PriceScale         *uint256.Int
	LiquidityAsset     common.Address
	WithdrawsQueued    *uint256.Int
	WithdrawsClaimed   *uint256.Int
	supportedSwapType  SwapType
	armType            ArmType
	hasWithdrawalQueue bool
	gas                Gas
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(p entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return nil, err
	}
	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     p.Address,
			Exchange:    p.Exchange,
			Type:        p.Type,
			Tokens:      lo.Map(p.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(p.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: p.BlockNumber,
		}},
		supportedSwapType:  extra.SwapTypes,
		armType:            extra.ArmType,
		hasWithdrawalQueue: extra.HasWithdrawalQueue,
		TradeRate0:         extra.TradeRate0,
		TradeRate1:         extra.TradeRate1,
		PriceScale:         extra.PriceScale,
		LiquidityAsset:     extra.LiquidityAsset,
		WithdrawsQueued:    extra.WithdrawsQueued,
		WithdrawsClaimed:   extra.WithdrawsClaimed,
		gas:                extra.Gas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = param.TokenAmountIn
		tokenOut      = param.TokenOut
	)

	indexIn, indexOut := p.GetTokenIndex(tokenAmountIn.Token), p.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	swapType := lo.Ternary(indexIn < indexOut, ZeroToOne, OneToZero)

	if p.supportedSwapType != swapType && p.supportedSwapType != Both {
		return nil, ErrUnsupportedSwap
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}

	if amountIn.Cmp(number.Zero) <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	amountOut := new(uint256.Int)
	switch p.armType {
	case Pegged:
		amountOut.Set(amountIn)
	case Pricable:
		price := lo.Ternary(indexIn == 0, p.TradeRate0, p.TradeRate1)
		amountOut.Div(new(uint256.Int).Mul(amountIn, price), p.PriceScale)
	default:
		return nil, ErrUnsupportedArmType
	}

	reserveOut := uint256.MustFromBig(p.Info.Reserves[indexOut])
	if p.hasWithdrawalQueue && common.HexToAddress(tokenOut).Cmp(p.LiquidityAsset) == 0 {
		//uint256 outstandingWithdrawals = withdrawsQueued - withdrawsClaimed;
		//amount + outstandingWithdrawals <= IERC20(liquidityAsset).balanceOf(address(this)),
		reserveOut.Sub(reserveOut, p.WithdrawsQueued).Add(reserveOut, p.WithdrawsClaimed)
	}

	if reserveOut.Sign() <= 0 || amountOut.Cmp(reserveOut) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: big.NewInt(0),
		},
		Gas: int64(lo.Ternary(swapType == ZeroToOne, p.gas.ZeroToOne, p.gas.OneToZero)),
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.Info.Reserves = slices.Clone(p.Info.Reserves)
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := p.GetTokenIndex(params.TokenAmountIn.Token), p.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	p.Info.Reserves[indexIn] = new(big.Int).Add(p.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	p.Info.Reserves[indexOut] = new(big.Int).Sub(p.Info.Reserves[indexOut], params.TokenAmountOut.Amount)
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return nil
}
