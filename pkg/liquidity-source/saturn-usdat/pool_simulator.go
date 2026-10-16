package saturnusdat

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// PoolSimulator models SwapFacility.swap(USDC, USDat, amount, recipient): a
// fee-free mint of USDat against USDC, bounded by USDat.assetCap(USDC).
// Tokens are [USDC, USDat]; only USDC -> USDat exists on-chain.
//
// The whitelist gates the caller and recipient, which on a single hop are the
// configured executor (see Config.Executor). Freeze checks are not modelled.
type PoolSimulator struct {
	pool.Pool
	Extra
	StaticExtra
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	if extra.AssetCap == nil || extra.AssetBalance == nil || len(entityPool.Tokens) != 2 {
		return nil, ErrInvalidExtra
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, _ int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
				func(item string, _ int) *big.Int { return bignumber.NewBig10(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		Extra:       extra,
		StaticExtra: staticExtra,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn, tokenOut := param.TokenAmountIn.Token, param.TokenOut
	if tokenIn != s.Info.Tokens[0] || tokenOut != s.Info.Tokens[1] {
		return nil, ErrInvalidSwapPath
	}
	if param.TokenAmountIn.Amount == nil || param.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrZeroAmount
	}
	amount, overflow := uint256.FromBig(param.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	// Check order follows the on-chain call path.
	switch {
	case s.FacilityPaused:
		return nil, ErrFacilityPaused
	case !s.ExtensionApproved:
		return nil, ErrExtensionNotApproved
	case !s.AssetAllowed:
		return nil, ErrAssetNotAllowed
	case s.USDatPaused:
		return nil, ErrUSDatPaused
	case s.NotWhitelisted:
		return nil, ErrNotWhitelisted
	}

	extAmount, effective, err := wrapAmounts(s.AssetDecimals, s.ExtensionDecimals, amount, s.AssetCap, s.AssetBalance)
	if err != nil {
		return nil, err
	}

	res := &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: extAmount.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenOut, Amount: bignumber.ZeroBI},
		Gas:            defaultGas,
		SwapInfo:       SwapInfo{AssetAmount: &effective},
	}
	if effective.Lt(amount) { // dust refunded by SwapFacility
		var dust uint256.Int
		res.RemainingTokenAmountIn = &pool.TokenAmount{Token: tokenIn, Amount: dust.Sub(amount, &effective).ToBig()}
	}
	return res, nil
}

// UpdateBalance consumes capacity: assetBalance += USDC actually pulled.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	info, ok := params.SwapInfo.(SwapInfo)
	if !ok || info.AssetAmount == nil {
		return
	}
	// Copy-on-write so clones sharing the old pointer are unaffected.
	s.AssetBalance = new(uint256.Int).Add(s.AssetBalance, info.AssetAmount)
	s.Info.Reserves = s.reserves()
}

// CloneState is shallow: UpdateBalance replaces AssetBalance and Info.Reserves wholesale
// (copy-on-write) and never writes through the shared pointers.
func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{ApprovalAddress: s.Facility, BlockNumber: s.Info.BlockNumber}
}

// CanSwapTo returns the tokens that can be swapped into `token`: USDC -> USDat only.
func (s *PoolSimulator) CanSwapTo(token string) []string {
	if token == s.Info.Tokens[1] {
		return []string{s.Info.Tokens[0]}
	}
	return []string{}
}

// CanSwapFrom returns the tokens `token` can be swapped into.
func (s *PoolSimulator) CanSwapFrom(token string) []string {
	if token == s.Info.Tokens[0] {
		return []string{s.Info.Tokens[1]}
	}
	return []string{}
}

func (s *PoolSimulator) reserves() []*big.Int {
	return Reserves(&s.Extra)
}

// Reserves reports remaining mintable capacity on both sides (USDC accepted,
// USDat mintable), or zero when swaps would revert on pause/approval flags.
func Reserves(e *Extra) []*big.Int {
	var remain, ext uint256.Int
	if !e.FacilityPaused && !e.USDatPaused && !e.NotWhitelisted && e.ExtensionApproved && e.AssetAllowed {
		remain = remainingCapacity(e.AssetCap, e.AssetBalance)
		if convertAmounts(&ext, e.AssetDecimals, e.ExtensionDecimals, &remain) != nil {
			ext.Clear()
		}
	}
	return []*big.Int{remain.ToBig(), ext.ToBig()}
}
