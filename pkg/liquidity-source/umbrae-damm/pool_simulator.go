package umbraedamm

import (
	"math/big"
	"slices"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	// reserves[0] = reserveX, reserves[1] = reserveY, matching the pair's tokenX/tokenY order.
	reserves []*uint256.Int
	feeBps   *uint256.Int
	feeToken string // lowercased; fee is always charged in this token
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	reserves := make([]*uint256.Int, len(ep.Reserves))
	for i, r := range ep.Reserves {
		v, err := uint256.FromDecimal(r)
		if err != nil {
			return nil, ErrInvalidReserve
		}
		reserves[i] = v
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(ep.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig(r) }),
			BlockNumber: ep.BlockNumber,
		}},
		reserves: reserves,
		feeBps:   uint256.NewInt(extra.FeeBps),
		feeToken: strings.ToLower(extra.FeeToken),
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexIn, indexOut := s.GetTokenIndex(param.TokenAmountIn.Token), s.GetTokenIndex(param.TokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(param.TokenAmountIn.Amount)
	if overflow || amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	reserveIn, reserveOut := s.reserves[indexIn], s.reserves[indexOut]
	if reserveIn.IsZero() || reserveOut.IsZero() {
		return nil, ErrInsufficientLiquidity
	}

	// Fee is always charged in feeToken: on input when the input side is feeToken, else on output.
	feeOnInput := strings.EqualFold(param.TokenAmountIn.Token, s.feeToken)
	amountOut, fee, reserveInDelta, reserveOutDelta := getAmountOut(amountIn, reserveIn, reserveOut, s.feeBps, feeOnInput)

	if amountOut.Sign() <= 0 {
		return nil, ErrInsufficientOutput
	}
	if reserveOutDelta.Cmp(reserveOut) >= 0 {
		return nil, ErrInsufficientLiquidity
	}

	feeToken := param.TokenAmountIn.Token
	if !feeOnInput {
		feeToken = param.TokenOut
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: param.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: feeToken, Amount: fee.ToBig()},
		Gas:            defaultGas,
		SwapInfo:       SwapInfo{ReserveInDelta: reserveInDelta.ToBig(), ReserveOutDelta: reserveOutDelta.ToBig()},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	// Reserve deltas come from CalcAmountOut; fees exit reserves into accumulators, so reserveOut
	// drops by the full pre-fee output and reserveIn rises by the post-fee input (K stays constant).
	inDelta := params.TokenAmountIn.Amount
	outDelta := params.TokenAmountOut.Amount
	if si, ok := params.SwapInfo.(SwapInfo); ok {
		if si.ReserveInDelta != nil {
			inDelta = si.ReserveInDelta
		}
		if si.ReserveOutDelta != nil {
			outDelta = si.ReserveOutDelta
		}
	}

	// Reassign (copy-on-write) so cloned states never share these pointers.
	s.reserves[indexIn] = new(uint256.Int).Add(s.reserves[indexIn], uint256.MustFromBig(inDelta))
	s.reserves[indexOut] = new(uint256.Int).Sub(s.reserves[indexOut], uint256.MustFromBig(outDelta))
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.reserves = slices.Clone(s.reserves)
	return &cloned
}

// GetApprovalAddress returns the pair itself: DAMM has no router — the user approves the pair and
// calls pair.swap() directly.
func (s *PoolSimulator) GetApprovalAddress(_, _ string) string {
	return s.Info.Address
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{BlockNumber: s.Info.BlockNumber}
}
