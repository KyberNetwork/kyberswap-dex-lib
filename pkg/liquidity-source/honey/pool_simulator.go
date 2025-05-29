package honey

import (
	"math/big"
	"slices"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	forcedBasketMode       bool
	registeredAssets       []string
	isBasketEnabledMint    bool
	isBasketEnabledRedeem  bool
	isPegged               []bool
	isBadCollateral        []bool
	mintRates              []*uint256.Int
	redeemRates            []*uint256.Int
	polFeeCollectorFeeRate *uint256.Int
	assetsDecimals         []uint8
	vaultsDecimals         []uint8
	vaultsMaxRedeems       []*uint256.Int
}

var ()

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
		isBasketEnabledMint:    extra.IsBasketEnabledMint,
		isBasketEnabledRedeem:  extra.IsBasketEnabledRedeem,
		isPegged:               extra.IsPegged,
		isBadCollateral:        extra.IsBadCollateral,
		registeredAssets:       extra.RegisteredAssets,
		mintRates:              extra.MintRates,
		redeemRates:            extra.RedeemRates,
		polFeeCollectorFeeRate: extra.PolFeeCollectorFeeRate,
		forcedBasketMode:       extra.ForceBasketMode,
		assetsDecimals:         extra.AssetsDecimals,
		vaultsDecimals:         extra.VaultsDecimals,
		vaultsMaxRedeems:       extra.VaultsMaxRedeems,
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

	var assetIndex int
	isMint := tokenAmountIn.Token != honeyToken
	if isMint {
		assetIndex = lo.IndexOf(p.registeredAssets, tokenAmountIn.Token)
	} else {
		assetIndex = lo.IndexOf(p.registeredAssets, tokenOut)
	}

	if isMint && !p.forcedBasketMode && !p.isBasketEnabledMint && !p.isBadCollateral[assetIndex] && p.isPegged[assetIndex] {
		assetAmount := uint256.MustFromBig(tokenAmountIn.Amount)
		shares := p.convertToShares(assetAmount, assetIndex)
		amountOut, feeShares, _ := p.getHoneyMintedFromShares(shares, assetIndex)
		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{Token: p.Pool.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
			Fee:            &pool.TokenAmount{Token: p.Pool.Info.Tokens[indexIn], Amount: feeShares.ToBig()},
			Gas:            defaultGas,
			SwapInfo:       SwapInfo{deltaShares: amountOut, assetIndex: assetIndex},
		}, nil
	} else if !isMint && !p.forcedBasketMode && !p.isBasketEnabledRedeem {
		if assetIndex >= len(p.vaultsMaxRedeems) || p.vaultsMaxRedeems[assetIndex].Cmp(amountIn) < 0 {
			return nil, ErrMaxRedeemAmountExceeded
		}
		shares, feeShares, _ := p.getSharesRedeemedFromHoney(amountIn, assetIndex)
		redeemedAssets := p.convertToAssets(shares, assetIndex)
		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{Token: p.Pool.Info.Tokens[indexOut], Amount: redeemedAssets.ToBig()},
			Fee:            &pool.TokenAmount{Token: p.Pool.Info.Tokens[indexIn], Amount: feeShares.ToBig()},
			Gas:            defaultGas,
			SwapInfo:       SwapInfo{deltaShares: new(uint256.Int).Neg(amountIn), assetIndex: assetIndex},
		}, nil
	}

	return nil, ErrBasketMode
}

func (p *PoolSimulator) convertToShares(assets *uint256.Int, assetIndex int) (share *uint256.Int) {
	var exponent uint8
	share = new(uint256.Int)
	if p.vaultsDecimals[assetIndex] >= p.assetsDecimals[assetIndex] {
		exponent = p.vaultsDecimals[assetIndex] - p.assetsDecimals[assetIndex]
		share.Mul(assets, share.Exp(U10, uint256.NewInt(uint64(exponent))))
	} else {
		exponent = p.assetsDecimals[assetIndex] - p.vaultsDecimals[assetIndex]
		share.Div(assets, share.Exp(U10, uint256.NewInt(uint64(exponent))))
	}
	return
}

func (p *PoolSimulator) getHoneyMintedFromShares(shares *uint256.Int,
	assetIndex int) (honeyAmount, feeReceiverFeeShares, polFeeCollectorFeeShares *uint256.Int) {
	honeyAmount, _ = new(uint256.Int).MulDivOverflow(shares, p.mintRates[assetIndex], U1e18)
	feeReceiverFeeShares = new(uint256.Int).Sub(shares, honeyAmount)
	polFeeCollectorFeeShares, _ = new(uint256.Int).MulDivOverflow(feeReceiverFeeShares, p.polFeeCollectorFeeRate, U1e18)
	feeReceiverFeeShares.Sub(feeReceiverFeeShares, polFeeCollectorFeeShares)
	return
}

func (p *PoolSimulator) getSharesRedeemedFromHoney(amountIn *uint256.Int,
	assetIndex int) (shares, feeReceiverFeeShares, polFeeCollectorFeeShares *uint256.Int) {
	shares, _ = new(uint256.Int).MulDivOverflow(amountIn, p.redeemRates[assetIndex], U1e18)
	feeReceiverFeeShares = new(uint256.Int).Sub(amountIn, shares)
	polFeeCollectorFeeShares, _ = new(uint256.Int).MulDivOverflow(feeReceiverFeeShares, p.polFeeCollectorFeeRate, U1e18)
	feeReceiverFeeShares.Sub(feeReceiverFeeShares, polFeeCollectorFeeShares)
	return
}

func (p *PoolSimulator) convertToAssets(shares *uint256.Int, assetIndex int) (assets *uint256.Int) {
	var exponent uint8
	assets = new(uint256.Int)
	if p.vaultsDecimals[assetIndex] >= p.assetsDecimals[assetIndex] {
		exponent = p.vaultsDecimals[assetIndex] - p.assetsDecimals[assetIndex]
		assets.Div(shares, assets.Exp(U10, uint256.NewInt(uint64(exponent))))
	} else {
		exponent = p.assetsDecimals[assetIndex] - p.vaultsDecimals[assetIndex]
		assets.Mul(shares, assets.Exp(U10, uint256.NewInt(uint64(exponent))))
	}
	return
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.vaultsMaxRedeems = slices.Clone(p.vaultsMaxRedeems)
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	if swapInfo, ok := params.SwapInfo.(SwapInfo); ok {
		p.vaultsMaxRedeems[swapInfo.assetIndex] = new(uint256.Int).Add(
			p.vaultsMaxRedeems[swapInfo.assetIndex], swapInfo.deltaShares)
	}
}

func (p *PoolSimulator) CanSwapTo(address string) []string {
	result := make([]string, 0, len(p.Info.Tokens))
	var tokenIndex = p.GetTokenIndex(address)
	if tokenIndex < 0 {
		return result
	}

	if address == honeyToken {
		return p.registeredAssets
	}

	return []string{honeyToken}
}

func (p *PoolSimulator) CanSwapFrom(address string) []string {
	return p.CanSwapTo(address)
}
