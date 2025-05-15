package honey

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type (
	PoolSimulator struct {
		pool.Pool
		gas                    Gas
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
	}

	Gas struct {
		Swap int64
	}
)

var (
	honeyToken                       = "0xfcbd14dc51f0a4d49d5e53c2e0950e0bc26d0dce"
	U_1e18                           = uint256.MustFromDecimal("1000000000000000000")
	U_10                             = uint256.MustFromDecimal("10")
	ErrInvalidToken                  = errors.New("invalid token")
	ErrInvalidAmountIn               = errors.New("invalid amount in")
	ErrInsufficientInputAmount       = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrAssetFullyLiquidatedCantCheck = errors.New("asset fully liquidated, can't check")
	ErrBasketMode                    = errors.New("basket mode")
)

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
		gas:                    defaultGas,
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
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = param.TokenAmountIn
		tokenOut      = param.TokenOut
	)

	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
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

	var isMint bool
	var assetIndex int

	if tokenAmountIn.Token != honeyToken {
		isMint = true
		assetIndex = lo.IndexOf(s.registeredAssets, tokenAmountIn.Token)
	} else {
		assetIndex = lo.IndexOf(s.registeredAssets, tokenOut)
	}
	assetAmount := uint256.MustFromBig(tokenAmountIn.Amount)
	if isMint && !s.forcedBasketMode && !s.isBasketEnabledMint && !s.isBadCollateral[assetIndex] && s.isPegged[assetIndex] {
		shares := s.convertToShares(assetAmount, assetIndex)
		amountOut, feeShares, _ := s.getHoneyMintedFromShares(shares, assetIndex)
		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
			Fee:            &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: feeShares.ToBig()},
			Gas:            s.gas.Swap,
		}, nil
	}

	if !isMint && !s.forcedBasketMode && !s.isBasketEnabledRedeem {
		sharesForRedeem, feeReceiverFeeShares, _ := s.getSharesRedeemedFromHoney(amountIn, assetIndex)
		redeemedAssets := s.convertToAssets(sharesForRedeem, assetIndex)
		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexOut], Amount: redeemedAssets.ToBig()},
			Fee:            &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: feeReceiverFeeShares.ToBig()},
			Gas:            s.gas.Swap,
		}, nil
	}

	return nil, ErrBasketMode
}

func (s *PoolSimulator) convertToShares(assets *uint256.Int, assetIndex int) (share *uint256.Int) {
	var exponent uint8
	share = new(uint256.Int)
	if s.vaultsDecimals[assetIndex] >= s.assetsDecimals[assetIndex] {
		exponent = s.vaultsDecimals[assetIndex] - s.assetsDecimals[assetIndex]
		share.Mul(assets, share.Exp(U_10, uint256.NewInt(uint64(exponent))))
	} else {
		exponent = s.assetsDecimals[assetIndex] - s.vaultsDecimals[assetIndex]
		share.Div(assets, share.Exp(U_10, uint256.NewInt(uint64(exponent))))
	}
	return
}

func (s *PoolSimulator) getHoneyMintedFromShares(shares *uint256.Int, assetIndex int) (honeyAmount *uint256.Int, feeReceiverFeeShares *uint256.Int, polFeeCollectorFeeShares *uint256.Int) {
	honeyAmount = new(uint256.Int)
	honeyAmount.Mul(shares, s.mintRates[assetIndex]).Div(honeyAmount, U_1e18)
	feeShares := new(uint256.Int).Sub(shares, honeyAmount)
	polFeeCollectorFeeShares = new(uint256.Int).Set(feeShares)
	polFeeCollectorFeeShares.Mul(polFeeCollectorFeeShares, s.polFeeCollectorFeeRate).Div(polFeeCollectorFeeShares, U_1e18)
	feeReceiverFeeShares = new(uint256.Int).Sub(feeShares, polFeeCollectorFeeShares)
	return
}

func (s *PoolSimulator) getSharesRedeemedFromHoney(amountIn *uint256.Int, assetIndex int) (shares *uint256.Int, feeReceiverFeeShares *uint256.Int, polFeeCollectorFeeShares *uint256.Int) {
	shares = new(uint256.Int)
	shares.Mul(amountIn, s.redeemRates[assetIndex]).Div(shares, U_1e18)
	feeShares := new(uint256.Int).Sub(amountIn, shares)
	polFeeCollectorFeeShares = new(uint256.Int).Set(feeShares)
	polFeeCollectorFeeShares.Mul(polFeeCollectorFeeShares, s.polFeeCollectorFeeRate).Div(polFeeCollectorFeeShares, U_1e18)
	feeReceiverFeeShares = new(uint256.Int).Sub(feeShares, polFeeCollectorFeeShares)
	return
}

func (s *PoolSimulator) convertToAssets(shares *uint256.Int, assetIndex int) (assets *uint256.Int) {
	var exponent uint8
	assets = new(uint256.Int)
	if s.vaultsDecimals[assetIndex] >= s.assetsDecimals[assetIndex] {
		exponent = s.vaultsDecimals[assetIndex] - s.assetsDecimals[assetIndex]
		assets.Div(shares, assets.Exp(U_10, uint256.NewInt(uint64(exponent))))
	} else {
		exponent = s.assetsDecimals[assetIndex] - s.vaultsDecimals[assetIndex]
		assets.Mul(shares, assets.Exp(U_10, uint256.NewInt(uint64(exponent))))
	}
	return
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {

}

func (s *PoolSimulator) CanSwapTo(address string) []string {
	result := make([]string, 0, len(s.Info.Tokens))
	var tokenIndex = s.GetTokenIndex(address)
	if tokenIndex < 0 {
		return result
	}

	if address == honeyToken {
		return s.registeredAssets
	}

	return []string{honeyToken}
}

func (s *PoolSimulator) CanSwapFrom(address string) []string {
	return s.CanSwapTo(address)
}
