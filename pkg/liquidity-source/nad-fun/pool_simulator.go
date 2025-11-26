package nadfun

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	isLocked    bool
	isGraduated bool

	virtualNative *uint256.Int
	virtualToken  *uint256.Int

	realNativeReserves *uint256.Int
	realTokenReserves  *uint256.Int

	k           *uint256.Int
	targetToken *uint256.Int

	protocolFee *uint256.Int

	router string
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  entityPool.Address,
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens: lo.Map(entityPool.Tokens,
					func(item *entity.PoolToken, _ int) string { return item.Address }),
				Reserves: lo.Map(entityPool.Reserves,
					func(item string, _ int) *big.Int { return bignumber.NewBig(item) }),
			},
		},

		isLocked:    extra.IsLocked,
		isGraduated: extra.IsGraduated,

		virtualNative: extra.VirtualNative,
		virtualToken:  extra.VirtualToken,

		realNativeReserves: big256.New(entityPool.Reserves[0]),
		realTokenReserves:  big256.New(entityPool.Reserves[1]),

		k:           extra.K,
		targetToken: extra.TargetToken,

		protocolFee: extra.ProtocolFee,

		router: staticExtra.Router,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut

	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	if s.isLocked {
		return nil, ErrPoolLocked
	}

	if s.isGraduated {
		return nil, ErrPoolGraduated
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	var (
		amountOut *uint256.Int
		fee       *uint256.Int
		gas       int64
		si        *SwapInfo
		err       error
	)

	isBuy := indexIn == 0
	if isBuy {
		amountOut, fee, si, gas, err = s.buy(amountIn)
	} else {
		amountOut, fee, si, gas, err = s.sell(amountIn)
	}
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[0], Amount: fee.ToBig()},
		Gas:            gas,
		SwapInfo:       si,
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenAmountOut, tokenIn := params.TokenAmountOut, params.TokenIn

	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	if s.isLocked {
		return nil, ErrPoolLocked
	}

	if s.isGraduated {
		return nil, ErrPoolGraduated
	}

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	var (
		amountIn *uint256.Int
		fee      *uint256.Int
		gas      int64
		si       *SwapInfo
		err      error
	)

	isBuy := indexIn == 0
	if isBuy {
		amountIn, fee, si, gas, err = s.buyExactOut(amountOut)
	} else {
		amountIn, fee, si, gas, err = s.sellExactOut(amountOut)
	}
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: s.Info.Tokens[0], Amount: fee.ToBig()},
		Gas:           gas,
		SwapInfo:      si,
	}, nil
}

func (s *PoolSimulator) buy(amountIn *uint256.Int) (*uint256.Int, *uint256.Int, *SwapInfo, int64, error) {
	fee := getFeeAmount(amountIn, s.protocolFee)

	amountInAfterFee := amountIn.Sub(amountIn, fee)

	amountOut, err := getAmountOut(amountInAfterFee, s.virtualNative, s.virtualToken, s.k)
	if err != nil {
		return nil, nil, nil, 0, err
	}

	newRealTokenReserves := new(uint256.Int).Sub(s.realTokenReserves, amountOut)
	if newRealTokenReserves.Lt(s.targetToken) {
		return nil, nil, nil, 0, ErrTargetExceeded
	}

	newVirtualNative := new(uint256.Int).Add(s.virtualNative, amountInAfterFee)
	newVirtualToken := new(uint256.Int).Sub(s.virtualToken, amountOut)
	if err := checkInvariant(newVirtualNative, newVirtualToken, s.k); err != nil {
		return nil, nil, nil, 0, err
	}

	return amountOut, fee, &SwapInfo{
		Router:                s.router,
		NewVirtualNative:      newVirtualNative,
		NewVirtualToken:       newVirtualToken,
		NewRealNativeReserves: new(uint256.Int).Add(s.realNativeReserves, amountIn),
		NewRealTokenReserves:  newRealTokenReserves,
		IsBuy:                 true,
		IsLocked:              newRealTokenReserves.Eq(s.targetToken),
	}, buyGas, nil
}

func (s *PoolSimulator) sell(amountIn *uint256.Int) (*uint256.Int, *uint256.Int, *SwapInfo, int64, error) {
	amountOut, err := getAmountOut(amountIn, s.virtualToken, s.virtualNative, s.k)
	if err != nil {
		return nil, nil, nil, 0, err
	}

	if amountOut.Gt(s.realNativeReserves) {
		return nil, nil, nil, 0, ErrInsufficientLiquidity
	}

	newVirtualToken := new(uint256.Int).Add(s.virtualToken, amountIn)
	newVirtualNative := new(uint256.Int).Sub(s.virtualNative, amountOut)
	if err := checkInvariant(newVirtualNative, newVirtualToken, s.k); err != nil {
		return nil, nil, nil, 0, err
	}

	fee := getFeeAmount(amountOut, s.protocolFee)

	amountOutAfterFee := amountOut.Sub(amountOut, fee)

	return amountOutAfterFee, fee, &SwapInfo{
		Router:                s.router,
		NewVirtualNative:      newVirtualNative,
		NewVirtualToken:       newVirtualToken,
		NewRealNativeReserves: new(uint256.Int).Sub(s.realNativeReserves, amountOutAfterFee),
		NewRealTokenReserves:  new(uint256.Int).Add(s.realTokenReserves, amountIn),
		IsBuy:                 false,
	}, sellGas, nil
}

func (s *PoolSimulator) buyExactOut(amountOut *uint256.Int) (*uint256.Int, *uint256.Int, *SwapInfo, int64, error) {
	newRealTokenReserves := new(uint256.Int).Sub(s.realTokenReserves, amountOut)
	if newRealTokenReserves.Lt(s.targetToken) {
		return nil, nil, nil, 0, ErrTargetExceeded
	}

	amountInAfterFee, err := getAmountIn(amountOut, s.virtualNative, s.virtualToken, s.k)
	if err != nil {
		return nil, nil, nil, 0, err
	}

	var diff uint256.Int
	diff.Sub(FeeDenom, s.protocolFee)
	if diff.IsZero() {
		return nil, nil, nil, 0, ErrInsufficientLiquidity
	}

	var newVirtualNative, newVirtualToken uint256.Int
	newVirtualNative.Add(s.virtualNative, amountInAfterFee)
	newVirtualToken.Sub(s.virtualToken, amountOut)
	if err := checkInvariant(&newVirtualNative, &newVirtualToken, s.k); err != nil {
		return nil, nil, nil, 0, err
	}

	var amountIn uint256.Int
	amountIn.Mul(amountInAfterFee, FeeDenom)
	amountIn.Add(&amountIn, &diff)
	amountIn.Sub(&amountIn, big256.U1)
	amountIn.Div(&amountIn, &diff)

	fee := getFeeAmount(&amountIn, s.protocolFee)

	return &amountIn, fee, &SwapInfo{
		NewVirtualNative:      &newVirtualNative,
		NewVirtualToken:       &newVirtualToken,
		NewRealNativeReserves: new(uint256.Int).Add(s.realNativeReserves, &amountIn),
		NewRealTokenReserves:  newRealTokenReserves,
		IsLocked:              newRealTokenReserves.Eq(s.targetToken),
	}, buyGas, nil
}

func (s *PoolSimulator) sellExactOut(amountOut *uint256.Int) (*uint256.Int, *uint256.Int, *SwapInfo, int64, error) {
	if amountOut.Gt(s.realNativeReserves) {
		return nil, nil, nil, 0, ErrInsufficientLiquidity
	}

	var diff uint256.Int
	diff.Sub(FeeDenom, s.protocolFee)
	if diff.IsZero() {
		return nil, nil, nil, 0, ErrInsufficientLiquidity
	}

	var amountOutBeforeFee uint256.Int
	amountOutBeforeFee.Mul(amountOut, FeeDenom)
	amountOutBeforeFee.Add(&amountOutBeforeFee, &diff)
	amountOutBeforeFee.Sub(&amountOutBeforeFee, big256.U1)
	amountOutBeforeFee.Div(&amountOutBeforeFee, &diff)

	fee := getFeeAmount(&amountOutBeforeFee, s.protocolFee)

	amountIn, err := getAmountIn(&amountOutBeforeFee, s.virtualToken, s.virtualNative, s.k)
	if err != nil {
		return nil, nil, nil, 0, err
	}

	var newVirtualToken, newVirtualNative uint256.Int
	newVirtualToken.Add(s.virtualToken, amountIn)
	newVirtualNative.Sub(s.virtualNative, &amountOutBeforeFee)
	if err := checkInvariant(&newVirtualNative, &newVirtualToken, s.k); err != nil {
		return nil, nil, nil, 0, err
	}

	return amountIn, fee, &SwapInfo{
		NewVirtualNative:      &newVirtualNative,
		NewVirtualToken:       &newVirtualToken,
		NewRealNativeReserves: new(uint256.Int).Sub(s.realNativeReserves, amountOut),
		NewRealTokenReserves:  new(uint256.Int).Add(s.realTokenReserves, amountIn),
	}, sellGas, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	if params.SwapInfo != nil {
		if si, ok := params.SwapInfo.(*SwapInfo); ok {
			s.virtualNative = si.NewVirtualNative
			s.virtualToken = si.NewVirtualToken
			s.realNativeReserves = si.NewRealNativeReserves
			s.realTokenReserves = si.NewRealTokenReserves
			if si.IsLocked {
				s.isLocked = true
			}
			return
		}
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.virtualNative = s.virtualNative.Clone()
	cloned.virtualToken = s.virtualToken.Clone()
	cloned.realNativeReserves = s.realNativeReserves.Clone()
	cloned.realTokenReserves = s.realTokenReserves.Clone()

	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return struct {
		BlockNumber uint64 `json:"blockNumber"`
	}{
		BlockNumber: s.Info.BlockNumber,
	}
}
