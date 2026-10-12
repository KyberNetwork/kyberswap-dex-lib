package ammalgam

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	reserves [2]uint256.Int
	extra    Extra
	gas      int64
}

var _ = pool.RegisterFactory(DexType, NewPoolSimulator)

func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	ep := params.EntityPool
	if ep.Type != DexType || !validPairAddress(ep.Address) || len(ep.Tokens) != 2 || len(ep.Reserves) != 2 {
		return nil, ErrInvalidPool
	}
	if ep.Tokens[0] == nil || ep.Tokens[1] == nil || !validPairAddress(ep.Tokens[0].Address) ||
		!validPairAddress(ep.Tokens[1].Address) || strings.EqualFold(ep.Tokens[0].Address, ep.Tokens[1].Address) {
		return nil, ErrInvalidPool
	}
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, fmt.Errorf("decode Ammalgam extra: %w", err)
	}
	var staticExtra StaticExtra
	if ep.StaticExtra != "" {
		if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
			return nil, fmt.Errorf("decode Ammalgam static extra: %w", err)
		}
	}
	gas := staticExtra.Gas
	if gas <= 0 {
		gas = defaultGas
	}
	s := &PoolSimulator{extra: extra, gas: gas}
	s.Info = pool.PoolInfo{
		Address: strings.ToLower(ep.Address), Exchange: ep.Exchange, Type: ep.Type,
		Tokens:   []string{strings.ToLower(ep.Tokens[0].Address), strings.ToLower(ep.Tokens[1].Address)},
		Reserves: make([]*big.Int, 2), BlockNumber: ep.BlockNumber,
	}
	for i, raw := range ep.Reserves {
		r, err := uint256.FromDecimal(raw)
		if err != nil || r.Gt(maxReserve112) {
			return nil, ErrInvalidPool
		}
		if extra.ReferenceReserves[i].IsZero() || extra.ReferenceReserves[i].Gt(maxReserve112) ||
			extra.MissingAssets[i].Gt(r) || extra.MissingAssets[i].Eq(r) {
			return nil, ErrInvalidPool
		}
		s.reserves[i] = *r
		s.Info.Reserves[i] = r.ToBig()
	}
	return s, nil
}

func validPairAddress(address string) bool {
	return common.IsHexAddress(address) && !valueobject.IsNative(address) && !valueobject.IsZero(address)
}

func validAmount(amount *big.Int) (uint256.Int, error) {
	var zero uint256.Int
	if amount == nil || amount.Sign() <= 0 {
		return zero, ErrInvalidAmount
	}
	u, overflow := uint256.FromBig(amount)
	if overflow {
		return zero, ErrInvalidAmount
	}
	return *u, nil
}

func (s *PoolSimulator) tokenIndices(tokenIn, tokenOut string) (int, int, error) {
	i := s.GetTokenIndex(strings.ToLower(tokenIn))
	j := s.GetTokenIndex(strings.ToLower(tokenOut))
	if i < 0 || j < 0 || i == j {
		return 0, 0, ErrInvalidToken
	}
	return i, j, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	i, j, err := s.tokenIndices(params.TokenAmountIn.Token, params.TokenOut)
	if err != nil {
		return nil, err
	}
	amountIn, err := validAmount(params.TokenAmountIn.Amount)
	if err != nil {
		return nil, err
	}
	amountOut, err := quoteExactIn(&amountIn, &s.reserves[i], &s.extra.ReferenceReserves[i],
		&s.reserves[j], &s.extra.MissingAssets[i], &s.extra.MissingAssets[j])
	if err != nil {
		return nil, err
	}
	info, err := s.postSwap(i, &amountIn, &amountOut)
	if err != nil {
		return nil, err
	}
	fee, err := s.inputFee(i, &amountIn)
	if err != nil {
		return nil, err
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Info.Tokens[j], Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[i], Amount: fee.ToBig()},
		Gas:            s.gas, SwapInfo: info,
	}, nil
}

// CalcAmountIn quotes the minimum input for a requested output. The on-chain
// pair.swap call still executes as exact-input with that input sent to the pair.
func (s *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	i, j, err := s.tokenIndices(params.TokenIn, params.TokenAmountOut.Token)
	if err != nil {
		return nil, err
	}
	amountOut, err := validAmount(params.TokenAmountOut.Amount)
	if err != nil {
		return nil, err
	}
	amountIn, err := quoteExactOut(&amountOut, &s.reserves[i], &s.extra.ReferenceReserves[i],
		&s.reserves[j], &s.extra.MissingAssets[i], &s.extra.MissingAssets[j])
	if err != nil {
		return nil, err
	}
	if amountIn.IsZero() {
		return nil, ErrInvalidAmount
	}
	info, err := s.postSwap(i, &amountIn, &amountOut)
	if err != nil {
		return nil, err
	}
	fee, err := s.inputFee(i, &amountIn)
	if err != nil {
		return nil, err
	}
	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: s.Info.Tokens[i], Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: s.Info.Tokens[i], Amount: fee.ToBig()},
		Gas:           s.gas, SwapInfo: info,
	}, nil
}

func (s *PoolSimulator) postSwap(inputIndex int, amountIn, amountOut *uint256.Int) (SwapInfo, error) {
	info := SwapInfo{PoolAddress: s.Info.Address, PreReserves: s.reserves}
	outputIndex := 1 - inputIndex
	if amountOut.IsZero() {
		return info, ErrZeroOutput
	}
	available, err := subChecked(&s.reserves[outputIndex], &s.extra.MissingAssets[outputIndex])
	if err != nil || !amountOut.Lt(&available) {
		return info, errSwapAmountOutExceedsAvailable
	}
	newInput, err := addChecked(&s.reserves[inputIndex], amountIn)
	if err != nil || newInput.Gt(maxReserve112) {
		return info, ErrReserveLimit
	}
	newOutput, err := subChecked(&s.reserves[outputIndex], amountOut)
	if err != nil {
		return info, err
	}
	info.Reserves[inputIndex], info.Reserves[outputIndex] = newInput, newOutput
	info.InputIndex, info.AmountIn, info.AmountOut = inputIndex, *amountIn, *amountOut
	return info, nil
}

func (s *PoolSimulator) inputFee(inputIndex int, amountIn *uint256.Int) (uint256.Int, error) {
	feeBipsQ64, err := calculateSwapFeeBipsQ64(amountIn, &s.reserves[inputIndex],
		&s.extra.ReferenceReserves[inputIndex])
	if err != nil {
		return uint256.Int{}, err
	}
	product, err := mulChecked(amountIn, &feeBipsQ64)
	if err != nil {
		return uint256.Int{}, err
	}
	return divChecked(&product, bipsQ64)
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	info, ok := params.SwapInfo.(SwapInfo)
	if !ok || !strings.EqualFold(info.PoolAddress, s.Info.Address) || info.InputIndex < 0 || info.InputIndex > 1 ||
		!info.PreReserves[0].Eq(&s.reserves[0]) || !info.PreReserves[1].Eq(&s.reserves[1]) {
		return
	}
	i, j, err := s.tokenIndices(params.TokenAmountIn.Token, params.TokenAmountOut.Token)
	if err != nil || i != info.InputIndex || j != 1-i || params.TokenAmountIn.Amount == nil ||
		params.TokenAmountOut.Amount == nil || info.AmountIn.ToBig().Cmp(params.TokenAmountIn.Amount) != 0 ||
		info.AmountOut.ToBig().Cmp(params.TokenAmountOut.Amount) != 0 {
		return
	}
	s.reserves = info.Reserves
	for k := range 2 {
		s.Info.Reserves[k] = info.Reserves[k].ToBig()
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	clone := *s
	clone.Info.Tokens = append([]string(nil), s.Info.Tokens...)
	clone.Info.Reserves = []*big.Int{new(big.Int).Set(s.Info.Reserves[0]), new(big.Int).Set(s.Info.Reserves[1])}
	if s.Info.SwapFee != nil {
		clone.Info.SwapFee = new(big.Int).Set(s.Info.SwapFee)
	}
	return &clone
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.RouterMetaInfo{Router: s.Info.Address, BlockNumber: s.Info.BlockNumber}
}
