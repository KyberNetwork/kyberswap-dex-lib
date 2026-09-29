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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	TradeRate0             *uint256.Int
	TradeRate1             *uint256.Int
	PriceScale             *uint256.Int
	LiquidityAsset         common.Address
	LiquidityAssetDecimals uint8
	WithdrawsQueued        *uint256.Int
	WithdrawsClaimed       *uint256.Int
	supportedSwapType      SwapType
	armType                ArmType
	hasWithdrawalQueue     bool
	// BaseAssets[i] describes Tokens[i+1] and is only populated for ArmType Pricable4626.
	BaseAssets []BaseAssetInfo
	gas        Gas
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
		supportedSwapType:      extra.SwapTypes,
		armType:                extra.ArmType,
		hasWithdrawalQueue:     extra.HasWithdrawalQueue,
		TradeRate0:             extra.TradeRate0,
		TradeRate1:             extra.TradeRate1,
		PriceScale:             extra.PriceScale,
		LiquidityAsset:         extra.LiquidityAsset,
		LiquidityAssetDecimals: extra.LiquidityAssetDecimals,
		WithdrawsQueued:        extra.WithdrawsQueued,
		WithdrawsClaimed:       extra.WithdrawsClaimed,
		BaseAssets:             extra.BaseAssets,
		gas:                    extra.Gas,
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

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}

	if amountIn.Cmp(number.Zero) <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	if p.armType == Pricable4626 {
		return p.calcAmountOutPricable4626(tokenAmountIn, tokenOut, indexIn, indexOut, amountIn)
	}

	swapType := lo.Ternary(indexIn < indexOut, ZeroToOne, OneToZero)
	if p.supportedSwapType != swapType && p.supportedSwapType != Both {
		return nil, ErrUnsupportedSwap
	}

	amountOut := new(uint256.Int)
	switch p.armType {
	case Pegged:
		amountOut.Set(amountIn)
	case Pricable:
		price := lo.Ternary(indexIn == 0, p.TradeRate0, p.TradeRate1)
		amountOut.MulDivOverflow(amountIn, price, p.PriceScale)
	default:
		return nil, ErrUnsupportedArmType
	}

	reserveOut := uint256.MustFromBig(p.Info.Reserves[indexOut])
	if p.hasWithdrawalQueue && common.HexToAddress(tokenOut).Cmp(p.LiquidityAsset) == 0 {
		//uint256 outstandingWithdrawals = withdrawsQueued - withdrawsClaimed;
		//amount + outstandingWithdrawals <= IERC20(liquidityAsset).balanceOf(address(this)),
		reserveOut.Sub(reserveOut, p.WithdrawsQueued).Add(reserveOut, p.WithdrawsClaimed)
	}

	if reserveOut.Sign() <= 0 || !amountOut.Lt(reserveOut) {
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

// calcAmountOutPricable4626 prices a swap for the upgraded AbstractARM contract, which only allows
// swaps between the liquidity asset (index 0) and one of its N base assets (indices 1..N) - a star
// topology, no base-to-base swaps - each priced and converted independently via baseAssetConfigs().
func (p *PoolSimulator) calcAmountOutPricable4626(
	tokenAmountIn pool.TokenAmount, tokenOut string, indexIn, indexOut int, amountIn *uint256.Int,
) (*pool.CalcAmountOutResult, error) {
	if (indexIn == 0) == (indexOut == 0) {
		// Either both sides are the liquidity asset (indexIn == indexOut == 0, invalid) or neither
		// side is (a base-to-base swap, unsupported on-chain).
		return nil, ErrUnsupportedSwap
	}

	baseIdx := indexIn - 1
	if indexIn == 0 {
		baseIdx = indexOut - 1
	}
	cfg := &p.BaseAssets[baseIdx]

	amountOut := new(uint256.Int)
	var liquidityRemaining *uint256.Int
	if indexIn == 0 {
		// Trader sells the liquidity asset for the base asset (ARM sells the base asset).
		shares := convertLiquidityAmountToBase(cfg, p.LiquidityAssetDecimals, amountIn)
		// sellPrice is liquidity-per-base and the ARM charges a premium when selling, so divide.
		amountOut.MulDivOverflow(shares, p.PriceScale, cfg.SellPrice)
		liquidityRemaining = cfg.SellLiquidityRemaining
	} else {
		// Trader sells the base asset for the liquidity asset (ARM buys the base asset).
		assets := convertBaseAmountToLiquidity(cfg, p.LiquidityAssetDecimals, amountIn)
		amountOut.MulDivOverflow(assets, cfg.BuyPrice, p.PriceScale)
		liquidityRemaining = cfg.BuyLiquidityRemaining
	}

	if amountOut.Gt(liquidityRemaining) {
		return nil, ErrInsufficientLiquidity
	}

	reserveOut := uint256.MustFromBig(p.Info.Reserves[indexOut])
	if p.hasWithdrawalQueue && indexOut == 0 {
		reserveOut.Sub(reserveOut, p.WithdrawsQueued).Add(reserveOut, p.WithdrawsClaimed)
	}
	if reserveOut.Sign() <= 0 || !amountOut.Lt(reserveOut) {
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
		Gas: int64(lo.Ternary(indexIn == 0, p.gas.ZeroToOne, p.gas.OneToZero)),
	}, nil
}

// convertBaseAmountToLiquidity mirrors AbstractARM's _convertToAssets: converts a base-asset amount
// (native base decimals) to its liquidity-asset value (native liquidity decimals).
func convertBaseAmountToLiquidity(cfg *BaseAssetInfo, liquidityAssetDecimals uint8, baseAmount *uint256.Int) *uint256.Int {
	out := new(uint256.Int)
	if cfg.PeggedToLiquidityAsset {
		return scaleDecimals(out, baseAmount, cfg.Decimals, liquidityAssetDecimals)
	}
	out.MulDivOverflow(baseAmount, cfg.ConvertRateAssetsPerShare, big256.TenPow(cfg.Decimals))
	return out
}

// convertLiquidityAmountToBase mirrors AbstractARM's _convertToShares.
func convertLiquidityAmountToBase(cfg *BaseAssetInfo, liquidityAssetDecimals uint8, liquidityAmount *uint256.Int) *uint256.Int {
	out := new(uint256.Int)
	if cfg.PeggedToLiquidityAsset {
		return scaleDecimals(out, liquidityAmount, liquidityAssetDecimals, cfg.Decimals)
	}
	out.MulDivOverflow(liquidityAmount, cfg.ConvertRateSharesPerAsset, big256.TenPow(liquidityAssetDecimals))
	return out
}

// scaleDecimals mirrors AbstractARM's _scaleBaseToLiquidity/_scaleLiquidityToBase: decimals are
// constrained to {6, 18} on-chain, so the only adjustment is a factor of 1e12.
func scaleDecimals(out, amount *uint256.Int, fromDecimals, toDecimals uint8) *uint256.Int {
	if fromDecimals == toDecimals {
		return out.Set(amount)
	}
	if toDecimals > fromDecimals {
		return out.Mul(amount, big256.TenPow(toDecimals-fromDecimals))
	}
	return out.Div(amount, big256.TenPow(fromDecimals-toDecimals))
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
	return pool.MetaInfo{BlockNumber: p.Info.BlockNumber}
}
