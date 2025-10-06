package cusd

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"math/big"
	"slices"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	paused             bool
	assetsPaused       []bool
	isWhitelist        bool
	assetPrices        []*uint256.Int
	capPrice           *uint256.Int
	decimals           []uint8
	vaultAssetSupplies []*uint256.Int
	capSupply          *uint256.Int
	fees               []*FeeData
	availableBalances  []*uint256.Int
	assets             []string
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(ep.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: ep.BlockNumber,
		}},
		decimals:           lo.Map(ep.Tokens, func(item *entity.PoolToken, index int) uint8 { return item.Decimals }),
		paused:             extra.Paused,
		assetsPaused:       extra.AssetsPaused,
		isWhitelist:        extra.IsWhitelist,
		assetPrices:        extra.Prices[:len(ep.Tokens)-1],
		capPrice:           extra.Prices[len(ep.Tokens)-1],
		vaultAssetSupplies: extra.VaultAssetSupplies,
		capSupply:          extra.CapSupply,
		fees:               extra.Fees,
		assets:             extra.Assets,
		availableBalances:  extra.AvailableBalances,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var (
		amountIn = uint256.MustFromBig(params.TokenAmountIn.Amount)
		tokenIn  = params.TokenAmountIn.Token
		tokenOut = params.TokenOut
	)
	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	isMint := indexOut == len(s.Info.Tokens)-1
	assetIndex := lo.Ternary(isMint, indexIn, indexOut)

	if s.paused {
		return nil, ErrContractPaused
	}

	if s.assetsPaused[assetIndex] {
		return nil, ErrAssetPaused
	}

	if !lo.Contains(s.assets, s.Pool.Info.Tokens[assetIndex]) {
		return nil, ErrAssetNotSupported
	}

	var fee uint256.Int
	amountOut, newRatio := s.amountOutBeforeFee(assetIndex, isMint, amountIn)
	if !s.isWhitelist {
		amountOut, fee = s.applyFeeSlopes(s.fees[assetIndex], isMint, amountOut, newRatio)
	}

	if !isMint {
		if s.availableBalances[assetIndex].Lt(new(uint256.Int).Add(amountOut, &fee)) {
			return nil, ErrInsufficientReserves
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  s.Info.Tokens[indexOut],
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  s.Info.Tokens[indexIn],
			Amount: fee.ToBig(),
		},
		Gas: lo.Ternary(isMint, defaultMintGas, defaultBurnGas),
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	isMint := indexOut == len(s.Info.Tokens)-1
	assetIndex := lo.Ternary(isMint, indexIn, indexOut)

	amountIn := uint256.MustFromBig(params.TokenAmountIn.Amount)
	amountOutWithFee := uint256.MustFromBig(params.TokenAmountOut.Amount)
	amountOutWithFee.Add(amountOutWithFee, uint256.MustFromBig(params.Fee.Amount))

	if isMint {
		s.vaultAssetSupplies[assetIndex] = new(uint256.Int).Add(s.vaultAssetSupplies[assetIndex], amountIn)
		s.availableBalances[assetIndex] = new(uint256.Int).Add(s.availableBalances[assetIndex], amountIn)
	} else {
		s.vaultAssetSupplies[assetIndex] = new(uint256.Int).Sub(s.vaultAssetSupplies[assetIndex], amountOutWithFee)
		s.availableBalances[assetIndex] = new(uint256.Int).Sub(s.availableBalances[assetIndex], amountOutWithFee)
	}
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return Meta{
		BlockNumber: s.Info.BlockNumber,
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.availableBalances = slices.Clone(s.availableBalances)
	cloned.vaultAssetSupplies = slices.Clone(s.vaultAssetSupplies)

	return &cloned
}

func (s *PoolSimulator) amountOutBeforeFee(assetIndex int, isMint bool,
	amount *uint256.Int) (amountOut *uint256.Int, newRatio *uint256.Int) {
	assetPrice := s.assetPrices[assetIndex]
	assetDecimalsPow := number.TenPow(s.decimals[assetIndex])
	capDecimalsPow := number.TenPow(s.decimals[len(s.decimals)-1])
	capValue, _ := new(uint256.Int).MulDivOverflow(s.capSupply, s.capPrice, capDecimalsPow)
	allocationValue, _ := new(uint256.Int).MulDivOverflow(s.vaultAssetSupplies[assetIndex], assetPrice, assetDecimalsPow)

	if isMint {
		assetValue, _ := new(uint256.Int).MulDivOverflow(amount, assetPrice, assetDecimalsPow)
		if s.capSupply.Sign() == 0 {
			newRatio = new(uint256.Int)
			amountOut, _ = new(uint256.Int).MulDivOverflow(assetValue, capDecimalsPow, assetPrice)
		} else {
			newRatio = new(uint256.Int).Add(allocationValue, assetValue)
			newRatio, _ = newRatio.MulDivOverflow(newRatio, rayPrecision, new(uint256.Int).Add(capValue, assetValue))
			amountOut, _ = new(uint256.Int).MulDivOverflow(assetValue, capDecimalsPow, s.capPrice)
		}
	} else {
		assetValue, _ := new(uint256.Int).MulDivOverflow(amount, s.capPrice, capDecimalsPow)
		if amount.Eq(s.capSupply) {
			newRatio = new(uint256.Int).Set(rayPrecision)
			amountOut, _ = new(uint256.Int).MulDivOverflow(assetValue, assetDecimalsPow, assetPrice)
		} else {
			if allocationValue.Lt(assetValue) || !capValue.Gt(assetValue) {
				newRatio = new(uint256.Int)
			} else {
				newRatio = new(uint256.Int).Sub(allocationValue, assetValue)
				newRatio, _ = newRatio.MulDivOverflow(newRatio, rayPrecision, new(uint256.Int).Sub(capValue, assetValue))
			}
			amountOut, _ = new(uint256.Int).MulDivOverflow(assetValue, assetDecimalsPow, assetPrice)
		}
	}

	return amountOut, newRatio
}

func (s *PoolSimulator) applyFeeSlopes(fees *FeeData, isMint bool, amount, ratio *uint256.Int) (*uint256.Int, uint256.Int) {
	var rate, temp uint256.Int
	if isMint {
		rate.Set(fees.MinMintFee)
		if ratio.Gt(fees.OptimalRatio) {
			if ratio.Gt(fees.MintKinkRatio) {
				excessRatio := new(uint256.Int).Sub(ratio, fees.MintKinkRatio)
				temp.Sub(rayPrecision, fees.MintKinkRatio)
				temp.MulDivOverflow(fees.Slope1, excessRatio, &temp)
				temp.Add(&temp, fees.Slope0)
			} else {
				rate.Mul(fees.Slope0, temp.Sub(ratio, fees.OptimalRatio))
				rate.Div(&rate, temp.Sub(fees.MintKinkRatio, fees.OptimalRatio))
			}
		}
	} else {
		if ratio.Lt(fees.OptimalRatio) {
			if ratio.Lt(fees.BurnKinkRatio) {
				excessRatio := new(uint256.Int).Sub(fees.BurnKinkRatio, ratio)
				rate.MulDivOverflow(fees.Slope1, excessRatio, fees.BurnKinkRatio)
				rate.Add(&rate, fees.Slope0)
			} else {
				temp.Sub(fees.OptimalRatio, ratio)
				rate.Mul(fees.Slope0, &temp)
				temp.Sub(fees.OptimalRatio, fees.BurnKinkRatio)
				rate.Div(&rate, &temp)
			}
		}
	}

	if rate.Gt(rayPrecision) {
		rate.Set(rayPrecision)
	}

	fee, _ := temp.MulDivOverflow(amount, &rate, rayPrecision)

	return temp.Sub(amount, fee), *fee
}
