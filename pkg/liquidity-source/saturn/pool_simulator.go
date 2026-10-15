package saturn

import (
	"fmt"
	"math/big"
	"slices"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	swapFacilityPaused  bool
	usdatPaused         bool
	extensionApproved   bool
	assetAllowed        bool
	assetCap            *uint256.Int
	assetBalance        *uint256.Int
	assetDecimals       uint8
	whitelistEnabled    bool
	executorWhitelisted bool

	gas Gas
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	gas := defaultGas
	if staticExtra.Gas.Mint > 0 {
		gas = staticExtra.Gas
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, _ int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, _ int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		swapFacilityPaused:  extra.SwapFacilityPaused,
		usdatPaused:         extra.USDatPaused,
		extensionApproved:   extra.ExtensionApproved,
		assetAllowed:        extra.AssetAllowed,
		assetCap:            lo.CoalesceOrEmpty(extra.AssetCap, uint256.NewInt(0)),
		assetBalance:        lo.CoalesceOrEmpty(extra.AssetBalance, uint256.NewInt(0)),
		assetDecimals:       extra.AssetDecimals,
		whitelistEnabled:    extra.WhitelistEnabled,
		executorWhitelisted: extra.ExecutorWhitelisted,
		gas:                 gas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if params.TokenAmountIn.Token != s.Info.Tokens[0] {
		return nil, ErrInvalidTokenIn
	}
	if params.TokenOut != s.Info.Tokens[1] {
		return nil, ErrInvalidTokenOut
	}
	if params.TokenAmountIn.Amount == nil || params.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	if s.swapFacilityPaused {
		return nil, ErrSwapFacilityPaused
	}
	if s.usdatPaused {
		return nil, ErrPoolPaused
	}
	if !s.extensionApproved {
		return nil, ErrExtensionNotApproved
	}
	if !s.assetAllowed {
		return nil, ErrAssetNotAllowed
	}
	if s.whitelistEnabled && !s.executorWhitelisted {
		return nil, ErrExecutorNotWhitelisted
	}

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	extensionAmount, err := convertAmount(amountIn, s.assetDecimals, pyusdxDecimals)
	if err != nil {
		return nil, err
	}
	if extensionAmount.IsZero() {
		return nil, ErrInvalidAmountOut
	}

	effectiveAssetAmount, err := convertAmount(extensionAmount, pyusdxDecimals, s.assetDecimals)
	if err != nil {
		return nil, err
	}

	var nextAssetBalance uint256.Int
	nextAssetBalance.Add(s.assetBalance, effectiveAssetAmount)
	if nextAssetBalance.Cmp(s.assetCap) > 0 {
		return nil, ErrAssetCapReached
	}

	result := &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: extensionAmount.ToBig()},
		Fee:            &pool.TokenAmount{Token: params.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            s.gas.Mint,
		SwapInfo: SwapInfo{
			EffectiveAssetAmount: new(uint256.Int).Set(effectiveAssetAmount),
			ExtensionAmount:      new(uint256.Int).Set(extensionAmount),
		},
	}

	if amountIn.Cmp(effectiveAssetAmount) < 0 {
		return nil, fmt.Errorf("effective amount exceeds amountIn")
	}
	if amountIn.Cmp(effectiveAssetAmount) > 0 {
		var remaining uint256.Int
		remaining.Sub(amountIn, effectiveAssetAmount)
		result.RemainingTokenAmountIn = &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: remaining.ToBig()}
	}

	return result, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok || swapInfo.EffectiveAssetAmount == nil {
		return
	}

	s.assetBalance.Add(s.assetBalance, swapInfo.EffectiveAssetAmount)
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = slices.Clone(s.Info.Reserves)
	cloned.assetCap = new(uint256.Int).Set(s.assetCap)
	cloned.assetBalance = new(uint256.Int).Set(s.assetBalance)
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return PoolMeta{BlockNumber: s.Info.BlockNumber}
}

func convertAmount(amount *uint256.Int, fromDecimals, toDecimals uint8) (*uint256.Int, error) {
	if fromDecimals == toDecimals {
		return new(uint256.Int).Set(amount), nil
	}

	if fromDecimals > toDecimals {
		var out uint256.Int
		out.Div(amount, big256.TenPow(uint64(fromDecimals-toDecimals)))
		return &out, nil
	}

	var out uint256.Int
	if _, overflow := out.MulOverflow(amount, big256.TenPow(uint64(toDecimals-fromDecimals))); overflow {
		return nil, ErrOverflow
	}

	return &out, nil
}
