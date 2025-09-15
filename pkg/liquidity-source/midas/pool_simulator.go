package midas

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

type PoolSimulator struct {
	pool.Pool
	IRedemptionVault

	tokenDecimals  uint8
	mTokenDecimals uint8

	tokenRemoved           bool
	tokenConfig            *TokenConfig
	depositInstantFnPaused bool
	redeemInstantFnPaused  bool
	dailyLimits            *uint256.Int
	mTokenRate             *uint256.Int
	tokenRate              *uint256.Int
	minAmount              *uint256.Int
	instantFee             *uint256.Int
	instantDailyLimit      *uint256.Int

	depositVaultPaused    bool
	depositVault          common.Address
	redemptionVaultPaused bool
	redemptionVault       common.Address
}

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  ep.Address,
			Exchange: ep.Exchange,
			Type:     ep.Type,
			Tokens: lo.Map(ep.Tokens, func(item *entity.PoolToken, index int) string {
				return item.Address
			}),
			Reserves: lo.Map(ep.Reserves, func(item string, index int) *big.Int {
				return bignumber.NewBig(item)
			}),
			BlockNumber: ep.BlockNumber,
		}},
		mTokenDecimals: ep.Tokens[0].Decimals,
		tokenDecimals:  ep.Tokens[1].Decimals,

		tokenRemoved:      extra.TokenRemoved,
		tokenConfig:       extra.TokenConfig,
		dailyLimits:       extra.InstantDailyLimit,
		tokenRate:         extra.TokenRate,
		mTokenRate:        extra.MTokenRate,
		minAmount:         extra.MinAmount,
		instantFee:        extra.InstantFee,
		instantDailyLimit: extra.InstantDailyLimit,

		depositInstantFnPaused: extra.DepositInstantFnPaused,
		depositVaultPaused:     extra.DepositVaultPaused,
		depositVault:           staticExtra.DepositVault,

		redeemInstantFnPaused: extra.RedeemInstantFnPaused,
		redemptionVaultPaused: extra.RedemptionVaultPaused,
		redemptionVault:       staticExtra.RedemptionVault,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if s.tokenRemoved {
		return nil, ErrTokenRemoved
	}

	var (
		amountIn = uint256.MustFromBig(params.TokenAmountIn.Amount)
		tokenIn  = params.TokenAmountIn.Token
		tokenOut = params.TokenOut
	)
	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	var (
		swapInfo *SwapInfo
		err      error
	)

	isDeposit := indexIn != 0
	if isDeposit {
		swapInfo, err = s.deposit(amountIn)
	} else {
		swapInfo, err = s.redeem(amountIn)
	}
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  s.Info.Tokens[indexOut],
			Amount: swapInfo.amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  s.Info.Tokens[indexIn],
			Amount: swapInfo.fee,
		},
		Gas:      0,
		SwapInfo: swapInfo,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	// TODO: update limits
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) interface{} {
	return Meta{
		BlockNumber: s.Info.BlockNumber,
	}
}

func (s *PoolSimulator) deposit(amountIn *uint256.Int) (*SwapInfo, error) {
	if s.depositVaultPaused {
		return nil, ErrDepositVaultPaused
	}

	if s.depositInstantFnPaused {
		return nil, ErrFnPaused
	}

	amountInUsd, tokenInUSDRate, err := s.convertTokenToUsd(amountIn)
	if err != nil {
		return nil, err
	}

	prevAllowance := s.tokenConfig.Allowance
	if amountIn.Gt(prevAllowance) && prevAllowance.Lt(u256.UMax) {
		return nil, ErrMVExceedAllowance
	}

	feeTokenAmount := s.truncate(s.getFeeAmount(amountIn), s.tokenDecimals)

	var feeInUsd uint256.Int
	feeInUsd.MulDivOverflow(feeTokenAmount, tokenInUSDRate, u256.BONE)

	mTokenAmount, err := s.convertUsdToMToken(new(uint256.Int).Sub(amountInUsd, &feeInUsd))
	if err != nil {
		return nil, err
	}

	return &SwapInfo{
		IsDeposit:      true,
		DepositVault:   &s.depositVault,
		AssetsInBase18: s.convertToBase18(amountIn, s.tokenDecimals).ToBig(),
		amountOut:      mTokenAmount.ToBig(),
		fee:            feeTokenAmount.ToBig(),
	}, nil
}

func (s *PoolSimulator) convertTokenToUsd(amount *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	rate := lo.Ternary(s.tokenConfig.Stable, StableCoinRate, s.tokenRate)

	if rate.Sign() == 0 {
		return nil, nil, ErrRateZero
	}

	amountInUsd, _ := new(uint256.Int).MulDivOverflow(amount, rate, u256.BONE)

	return amountInUsd, rate, nil
}

func (s *PoolSimulator) getFeeAmount(amount *uint256.Int) *uint256.Int {
	feePercent := new(uint256.Int).Add(s.tokenConfig.Fee, s.instantFee)
	if feePercent.Gt(u256.UBasisPoint) {
		feePercent.Set(u256.UBasisPoint)
	}
	feePercent.MulDivOverflow(amount, feePercent, u256.UBasisPoint)

	return feePercent
}

func (s *PoolSimulator) truncate(value *uint256.Int, decimals uint8) *uint256.Int {
	if value.Sign() == 0 || decimals == 18 {
		return value
	}

	diff := 18 - decimals
	if diff > 0 {
		value.Div(value, u256.TenPow(diff)).Mul(value, u256.TenPow(diff))
	}

	return value.Mul(value, u256.TenPow(-diff)).Div(value, u256.TenPow(-diff))
}

func (s *PoolSimulator) convertToBase18(amount *uint256.Int, decimals uint8) *uint256.Int {
	if amount.Sign() == 0 || decimals == 18 {
		return new(uint256.Int).Set(amount)
	}

	diff := 18 - decimals
	if diff > 0 {
		return new(uint256.Int).Mul(amount, u256.TenPow(diff))
	}

	return new(uint256.Int).Div(amount, u256.TenPow(-diff))
}

func (s *PoolSimulator) convertUsdToMToken(amountUsd *uint256.Int) (*uint256.Int, error) {
	if s.mTokenRate.Sign() == 0 {
		return nil, ErrRateZero
	}

	amountMToken, _ := new(uint256.Int).MulDivOverflow(amountUsd, u256.BONE, s.mTokenRate)

	return amountMToken, nil
}

func (s *PoolSimulator) redeem(amountIn *uint256.Int) (*SwapInfo, error) {
	if s.redemptionVaultPaused {
		return nil, ErrRedemptionVaultPaused
	}

	if s.redeemInstantFnPaused {
		return nil, ErrFnPaused
	}

	amountInUsd, tokenInUSDRate, err := s.convertTokenToUsd(amountIn)
	if err != nil {
		return nil, err
	}

	prevAllowance := s.tokenConfig.Allowance
	if amountIn.Gt(prevAllowance) && prevAllowance.Lt(u256.UMax) {
		return nil, ErrMVExceedAllowance
	}

	feeTokenAmount := s.truncate(s.getFeeAmount(amountIn), s.tokenDecimals)

	var feeInUsd uint256.Int
	feeInUsd.MulDivOverflow(feeTokenAmount, tokenInUSDRate, u256.BONE)

	mTokenAmount, err := s.convertUsdToMToken(new(uint256.Int).Sub(amountInUsd, &feeInUsd))
	if err != nil {
		return nil, err
	}

	return &SwapInfo{
		IsDeposit:       false,
		RedemptionVault: &s.redemptionVault,
		amountOut:       mTokenAmount.ToBig(),
		fee:             feeTokenAmount.ToBig(),
	}, nil
}
