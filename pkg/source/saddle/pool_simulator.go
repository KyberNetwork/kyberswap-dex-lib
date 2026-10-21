package saddle

import (
	"errors"
	"math/big"
	"slices"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var errNoOutput = errors.New("i'm dead here")

type PoolSimulator struct {
	pool.Pool
	reserves    []uint256.Int // pool.Info.Reserves as uint256
	swapFee     uint256.Int   // pool.Info.SwapFee as uint256
	Multipliers []uint256.Int
	// extra fields
	InitialA           uint256.Int
	FutureA            uint256.Int
	InitialATime       int64
	FutureATime        int64
	AdminFee           uint256.Int
	DefaultWithdrawFee uint256.Int
	LpToken            string
	LpSupply           uint256.Int
	Paused             bool
	gas                Gas
}

var _ = pool.RegisterFactory0(DexTypeSaddle, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	numTokens := len(entityPool.Tokens)
	if len(entityPool.Reserves) <= numTokens || len(staticExtra.PrecisionMultipliers) < numTokens {
		return nil, errInvalidAmount
	}
	s := &PoolSimulator{
		reserves:     make([]uint256.Int, numTokens),
		Multipliers:  make([]uint256.Int, numTokens),
		InitialATime: extra.InitialATime,
		FutureATime:  extra.FutureATime,
		LpToken:      staticExtra.LpToken,
		Paused:       extra.Paused,
		gas:          DefaultGas,
	}
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)
	for i := range numTokens {
		tokens[i] = entityPool.Tokens[i].Address
		if err := s.reserves[i].SetFromDecimal(entityPool.Reserves[i]); err != nil {
			return nil, err
		}
		reserves[i] = s.reserves[i].ToBig()
		if err := s.Multipliers[i].SetFromDecimal(staticExtra.PrecisionMultipliers[i]); err != nil {
			return nil, err
		}
	}
	for _, f := range []struct {
		dst *uint256.Int
		src string
	}{
		{&s.swapFee, extra.SwapFee}, {&s.InitialA, extra.InitialA}, {&s.FutureA, extra.FutureA},
		{&s.AdminFee, extra.AdminFee}, {&s.LpSupply, entityPool.Reserves[numTokens]},
	} {
		if err := f.dst.SetFromDecimal(f.src); err != nil {
			return nil, err
		}
	}
	// only have withdrawFee in saddle v1, default to 0
	if err := s.DefaultWithdrawFee.SetFromDecimal(extra.DefaultWithdrawFee); err != nil {
		s.DefaultWithdrawFee.Clear()
	}

	s.Pool = pool.Pool{
		Info: pool.PoolInfo{
			Address:     strings.ToLower(entityPool.Address),
			SwapFee:     s.swapFee.ToBig(),
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: entityPool.BlockNumber,
		},
	}
	return s, nil
}

func (t *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if t.Paused {
		return nil, ErrPoolPaused
	}

	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	var amountIn uint256.Int
	if err := toU256(&amountIn, tokenAmountIn.Amount); err != nil {
		return nil, err
	}
	a, err := getAPrecise(t.FutureATime, &t.FutureA, t.InitialATime, &t.InitialA)
	if err != nil {
		return nil, err
	}
	var amountOut, fee uint256.Int
	var gas int64
	if tokenAmountIn.Token == t.LpToken {
		// withdraw
		var tokenIndexTo = t.Info.GetTokenIndex(tokenOut)
		if tokenIndexTo < 0 {
			return nil, errNoOutput
		}
		if err := calculateWithdrawOneToken(t.reserves, t.Multipliers, &a, &t.swapFee, &t.DefaultWithdrawFee,
			&t.LpSupply, tokenIndexTo, &amountIn, &amountOut, &fee); err != nil {
			return nil, err
		}
		gas = t.gas.RemoveLiquidity
	} else if tokenOut == t.LpToken {
		// deposit
		var tokenIndexFrom = t.Info.GetTokenIndex(tokenAmountIn.Token)
		if tokenIndexFrom < 0 {
			return nil, errNoOutput
		}
		var amountsArr [maxTokens]uint256.Int
		if len(t.reserves) > maxTokens {
			return nil, errTooManyTokens
		}
		amounts := amountsArr[:len(t.reserves)]
		amounts[tokenIndexFrom] = amountIn
		if err := calculateTokenAmount(t.reserves, t.Multipliers, &a, &t.DefaultWithdrawFee, &t.LpSupply, amounts,
			true, &amountOut); err != nil {
			return nil, err
		}
		gas = t.gas.AddLiquidity
	} else {
		// swap from token to token
		var tokenIndexFrom = t.Info.GetTokenIndex(tokenAmountIn.Token)
		var tokenIndexTo = t.Info.GetTokenIndex(tokenOut)
		if tokenIndexFrom < 0 || tokenIndexTo < 0 {
			return nil, errNoOutput
		}
		if err := calculateSwap(t.reserves, t.Multipliers, &a, &t.swapFee, tokenIndexFrom, tokenIndexTo, &amountIn,
			&amountOut, &fee); err != nil {
			return nil, err
		}
		gas = t.gas.Swap
	}
	if amountOut.IsZero() {
		return nil, errNoOutput
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenOut, Amount: fee.ToBig()},
		Gas:            gas,
	}, nil
}

func (t *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *t
	cloned.Info.Reserves = slices.Clone(t.Info.Reserves)
	cloned.reserves = slices.Clone(t.reserves)
	return &cloned
}

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output, fee := params.TokenAmountIn, params.TokenAmountOut, params.Fee
	var inputAmount, outputAmount, adminShare uint256.Int
	inputAmount.SetFromBig(input.Amount)
	outputAmount.SetFromBig(output.Amount)
	if input.Token == t.LpToken {
		// withdraw: output = output + fee * adminFee
		adminShare.SetFromBig(fee.Amount)
		adminShare.MulDivOverflow(&adminShare, &t.AdminFee, feeDenominator)
		outputAmount.Add(&outputAmount, &adminShare)
		t.LpSupply.Sub(&t.LpSupply, &inputAmount)
	} else if output.Token == t.LpToken {
		// deposit
		t.LpSupply.Add(&t.LpSupply, &outputAmount)
	} else {
		// swap fee
		// output = output + output * swapFee * adminFee
		adminShare.MulDivOverflow(&outputAmount, &t.swapFee, feeDenominator)
		adminShare.MulDivOverflow(&adminShare, &t.AdminFee, feeDenominator)
		outputAmount.Add(&outputAmount, &adminShare)
	}
	for i := range t.Info.Tokens {
		if t.Info.Tokens[i] == input.Token {
			t.reserves[i].Add(&t.reserves[i], &inputAmount)
			t.Info.Reserves[i] = t.reserves[i].ToBig()
		}
		if t.Info.Tokens[i] == output.Token {
			t.reserves[i].Sub(&t.reserves[i], &outputAmount)
			t.Info.Reserves[i] = t.reserves[i].ToBig()
		}
	}
}

func (t *PoolSimulator) CanSwapFrom(address string) []string { return t.CanSwapTo(address) }

func (t *PoolSimulator) CanSwapTo(address string) []string {
	var ret = make([]string, 0)
	var tokenIndex = t.GetTokenIndex(address)
	if tokenIndex < 0 && address != t.LpToken {
		return nil
	}
	for i := 0; i < len(t.Info.Tokens); i += 1 {
		if i != tokenIndex {
			ret = append(ret, t.Info.Tokens[i])
		}
	}
	if address != t.LpToken {
		ret = append(ret, t.LpToken)
	}
	return ret
}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	var fromId = t.GetTokenIndex(tokenIn)
	var toId = t.GetTokenIndex(tokenOut)
	return Meta{
		TokenInIndex:  fromId,
		TokenOutIndex: toId,
		PoolLength:    len(t.Info.Tokens),
		LpToken:       t.LpToken,
		BlockNumber:   t.Info.BlockNumber,
	}
}
