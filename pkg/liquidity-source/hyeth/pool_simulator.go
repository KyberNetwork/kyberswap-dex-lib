package hyeth

import (
	"errors"
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/ethereum/go-ethereum/common"
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
		gas                       Gas
		managerIssueFee           *uint256.Int   // FlashMintHyETHV3 -> issuanceModule -> issuanceSettings(hyETH)
		managerRedeemFee          *uint256.Int   // FlashMintHyETHV3 -> issuanceModule -> issuanceSettings(hyETH)
		component                 common.Address // hyETH -> components() -> erc4626 MetaMorpho
		componentTotalSupply      *uint256.Int   // erc4626 MetaMorpho -> totalSupply()
		componentTotalAsset       *uint256.Int   // erc4626 MetaMorpho -> totalAssets()
		componentHyethBalance     *uint256.Int   // erc4626 MetaMorpho -> balanceOf(hyETH)
		defaultPositionRealUnit   *uint256.Int   // hyETH -> defaultPositionRealUnit(component)
		externalPositionRealUnits []*uint256.Int // hyETH -> getExternalPositionModules + hyETH -> getExternalPositionRealUnit(module) //TODO: not implemented
		hyethTotalSupply          *uint256.Int   // hyETH -> totalSupply()
		maxDeposit                *uint256.Int
		maxRedeem                 *uint256.Int
		isDisabled                bool
	}

	Gas struct {
		Issue  int64
		Redeem int64
	}
)

var (
	hyethToken                 = "0xc4506022fb8090774e8a628d5084eed61d9b99ee"
	ErrInvalidToken            = errors.New("invalid token")
	ErrInvalidAmountIn         = errors.New("invalid amount in")
	ErrInsufficientInputAmount = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrExact1Component         = errors.New("only supports exact 1 component")
	U_1e18                     = uint256.MustFromDecimal("1000000000000000000")

	ErrERC4626DepositMoreThanMax = errors.New("ERC4626: deposit more than max")
	ErrERC4626RedeemMoreThanMax  = errors.New("ERC4626: redeem more than max")
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
		managerIssueFee:           extra.ManagerIssueFee,
		managerRedeemFee:          extra.ManagerRedeemFee,
		component:                 extra.Component,
		componentTotalSupply:      extra.ComponentTotalSupply,
		componentTotalAsset:       extra.ComponentTotalAsset,
		componentHyethBalance:     extra.ComponentHyethBalance,
		defaultPositionRealUnit:   extra.DefaultPositionRealUnit,
		externalPositionRealUnits: extra.ExternalPositionRealUnits,
		hyethTotalSupply:          extra.HyethTotalSupply,
		maxDeposit:                extra.MaxDeposit,
		maxRedeem:                 extra.MaxRedeem,
		isDisabled:                extra.IsDisabled,
		gas:                       defaultGas,
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

	if s.isDisabled {
		return nil, ErrExact1Component
	}

	amountOut := new(uint256.Int)
	var err error
	var postTotalShares, postTotalAssets *uint256.Int

	isRedeem := strings.EqualFold(tokenAmountIn.Token, hyethToken)
	if isRedeem {
		amountOut, err = s.redeemSetForETH(amountIn)
		if err != nil {
			return nil, err
		}
		postTotalShares = new(uint256.Int).Sub(s.componentTotalSupply, amountIn)
		postTotalAssets = new(uint256.Int).Sub(s.componentTotalAsset, amountOut)
	} else {
		amountOut, err = s.issueSetFromETH(amountIn)
		if err != nil {
			return nil, err
		}
		postTotalShares = new(uint256.Int).Add(s.componentTotalSupply, amountOut)
		postTotalAssets = new(uint256.Int).Add(s.componentTotalAsset, amountIn)
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: big.NewInt(0)},
		Gas:            lo.Ternary(isRedeem, s.gas.Redeem, s.gas.Issue),

		SwapInfo: SwapInfo{
			Fee:         lo.Ternary(strings.EqualFold(tokenAmountIn.Token, hyethToken), s.managerRedeemFee, s.managerIssueFee),
			TotalSupply: postTotalShares,
			TotalAssets: postTotalAssets,
		},
	}, nil
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	postSwapState := params.SwapInfo.(SwapInfo)
	s.componentTotalSupply.Set(postSwapState.TotalSupply)
	s.componentTotalAsset.Set(postSwapState.TotalAssets)
}
