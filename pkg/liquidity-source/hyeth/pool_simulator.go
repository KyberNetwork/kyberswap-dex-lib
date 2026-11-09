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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
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
		Info: pool.PoolInfo{
			Address:     p.Address,
			Exchange:    p.Exchange,
			Type:        p.Type,
			Tokens:      lo.Map(p.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(p.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: p.BlockNumber,
		},
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

	// The swap's state after it: the component vault's shares and assets, hyETH's balance of those shares
	// and hyETH's supply. The vault's shares move by the component shares deposited or redeemed, not by the
	// hyETH amount: a hyETH holds about 1.26 shares.
	isRedeem := strings.EqualFold(tokenAmountIn.Token, hyethToken)
	var (
		amountOut *uint256.Int
		post      SwapInfo
		err       error
	)
	if isRedeem {
		var position *uint256.Int
		if amountOut, position, err = s.redeemSetForETH(amountIn); err != nil {
			return nil, err
		}
		// DebtIssuanceModule.redeem burns the quantity and mints the fee to the fee recipient.
		burnt := s.calculateTotalFees(amountIn, false)
		post = SwapInfo{
			TotalSupply:  new(uint256.Int).Sub(s.componentTotalSupply, position),
			TotalAssets:  new(uint256.Int).Sub(s.componentTotalAsset, amountOut),
			HyethBalance: new(uint256.Int).Sub(s.componentHyethBalance, position),
			HyethSupply:  new(uint256.Int).Sub(s.hyethTotalSupply, burnt),
			MaxDeposit:   new(uint256.Int).Add(s.maxDeposit, amountOut),
		}
	} else {
		var componentShares *uint256.Int
		if amountOut, componentShares, err = s.issueSetFromETH(amountIn); err != nil {
			return nil, err
		}
		// DebtIssuanceModule.issue mints the quantity and its fee, and takes the components for both,
		// rounded up.
		minted := s.calculateTotalFees(amountOut, true)
		used, _ := new(uint256.Int).MulDivOverflow(minted, s._getTotalIssuanceUnits(), U_1e18)
		if new(uint256.Int).MulMod(minted, s._getTotalIssuanceUnits(), U_1e18).Sign() > 0 {
			used.AddUint64(used, 1)
		}
		post = SwapInfo{
			TotalSupply:  new(uint256.Int).Add(s.componentTotalSupply, componentShares),
			TotalAssets:  new(uint256.Int).Add(s.componentTotalAsset, amountIn),
			HyethBalance: new(uint256.Int).Add(s.componentHyethBalance, used),
			HyethSupply:  new(uint256.Int).Add(s.hyethTotalSupply, minted),
			MaxDeposit:   new(uint256.Int).Sub(s.maxDeposit, amountIn),
		}
	}
	post.Fee = lo.Ternary(isRedeem, s.managerRedeemFee, s.managerIssueFee)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[indexIn], Amount: big.NewInt(0)},
		Gas:            lo.Ternary(isRedeem, s.gas.Redeem, s.gas.Issue),

		SwapInfo: post,
	}, nil
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	return MetaInfo{
		ApprovalAddress: s.GetApprovalAddress(tokenIn, tokenOut),
		BlockNumber:     s.Info.BlockNumber,
	}
}

func (s *PoolSimulator) GetApprovalAddress(tokenIn, _ string) string {
	if valueobject.IsNative(tokenIn) || valueobject.IsWrappedNative(tokenIn, valueobject.ChainIDEthereum) {
		return ""
	}

	return s.GetAddress()
}

// CloneState copies the simulator. UpdateBalance replaces the state's values instead of writing into them, so the
// copy shares them with the original.
func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	return &cloned
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	post, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}
	s.componentTotalSupply, s.componentTotalAsset = post.TotalSupply, post.TotalAssets
	s.componentHyethBalance, s.hyethTotalSupply, s.maxDeposit = post.HyethBalance, post.HyethSupply, post.MaxDeposit
}

func (s *PoolSimulator) SwapReceiveNativeIn(tokenIn, _ string, chainId valueobject.ChainID) bool {
	return valueobject.IsWrappedNative(tokenIn, chainId)
}

func (s *PoolSimulator) SwapReturnNativeOut(_, tokenOut string, chainId valueobject.ChainID) bool {
	return valueobject.IsWrappedNative(tokenOut, chainId)
}
