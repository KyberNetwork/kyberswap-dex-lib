package midas

import (
	"math/big"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

type PoolSimulator struct {
	pool.Pool
	depositVault    IDepositVault
	redemptionVault IRedemptionVault
	canDeposit      bool
	canRedeem       bool
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

	mTokenDecimals := ep.Tokens[0].Decimals
	tokenDecimals := ep.Tokens[1].Decimals

	var (
		depositVault    IDepositVault
		redemptionVault IRedemptionVault
	)

	if staticExtra.CanDeposit {
		switch *staticExtra.DepositVaultType {
		case DepositVaultDefault:
			depositVault = NewDepositVault(extra.DepositVault, mTokenDecimals, tokenDecimals, staticExtra.DepositVault)
		case DepositVaultWithUstb:
			return nil, ErrNotSupported
		}
	}

	if staticExtra.CanRedeem {
		switch *staticExtra.RedemptionVaultType {
		case RedemptionVaultWithSwapper:
			redemptionVault = NewRedemptionVaultSwapper(extra.RedemptionVault, mTokenDecimals,
				tokenDecimals, staticExtra.DepositVault)
		default:
			return nil, ErrNotSupported
		}
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
		canDeposit:      staticExtra.CanDeposit,
		canRedeem:       staticExtra.CanRedeem,
		depositVault:    depositVault,
		redemptionVault: redemptionVault,
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

	var (
		swapInfo *SwapInfo
		err      error
	)

	isDeposit := indexIn != 0
	if isDeposit {
		swapInfo, err = s.depositVault.DepositInstant(amountIn)
	} else {
		swapInfo, err = s.redemptionVault.RedeemInstant(amountIn)
	}
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  s.Info.Tokens[indexOut],
			Amount: swapInfo.AmountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  s.Info.Tokens[indexIn],
			Amount: swapInfo.Fee.ToBig(),
		},
		Gas:      swapInfo.Gas,
		SwapInfo: swapInfo,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo := params.SwapInfo.(*SwapInfo)
	if swapInfo.IsDeposit {
		if err := s.depositVault.UpdateState(swapInfo); err != nil {
			logger.Errorf("failed to update deposit vault state: %v", err)
		}
	} else if err := s.redemptionVault.UpdateState(swapInfo); err != nil {
		logger.Errorf("failed to update redemption vault state: %v", err)
	}

}

func (s *PoolSimulator) GetMetaInfo(_, _ string) interface{} {
	return Meta{
		BlockNumber: s.Info.BlockNumber,
	}
}
