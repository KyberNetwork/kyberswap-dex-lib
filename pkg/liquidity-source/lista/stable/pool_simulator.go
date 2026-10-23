package stable

import (
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/base"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	ErrInvalidValue    = errors.New("invalid value")
	ErrPriceDiffToken0 = errors.New("price difference for token0 exceeds threshold")
	ErrPriceDiffToken1 = errors.New("price difference for token1 exceeds threshold")

	errInvalidState   = errors.New("pool state out of uint256 range")
	errOverflow       = errors.New("uint256 overflow")
	errUnderflow      = errors.New("uint256 underflow")
	errRampNotStarted = errors.New("amplification ramp has not started")
	errNegativeDy     = errors.New("negative dy")
)

// PoolSimulator mirrors curve/base's swap math (lista's StableSwap is a Curve StableSwap fork) in
// uint256, plus the oracle price check lista runs after every swap.
type PoolSimulator struct {
	pool.Pool
	reserves     []uint256.Int // pool.Info.Reserves as uint256
	rates        []uint256.Int
	multipliers  []uint256.Int
	initialA     uint256.Int
	futureA      uint256.Int
	initialATime int64
	futureATime  int64
	swapFee      uint256.Int
	adminFee     uint256.Int
	aPrecision   uint256.Int
	nCoins       uint256.Int
	gas          int64

	decimals           []uint8
	isNativeCoins      []bool
	oraclePrices       [2]*uint256.Int
	priceDiffThreshold [2]*uint256.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	// curve/base parses the pool exactly as before the port; its big.Int state is converted below.
	curveBaseSimulator, err := base.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	decimals := make([]uint8, len(entityPool.Tokens))
	for i, token := range entityPool.Tokens {
		decimals[i] = token.Decimals
	}

	n := len(curveBaseSimulator.Info.Tokens)
	s := &PoolSimulator{
		Pool:               curveBaseSimulator.Pool,
		reserves:           make([]uint256.Int, n),
		rates:              make([]uint256.Int, n),
		multipliers:        make([]uint256.Int, n),
		initialATime:       curveBaseSimulator.InitialATime,
		futureATime:        curveBaseSimulator.FutureATime,
		gas:                base.DefaultGas.Exchange,
		decimals:           decimals,
		isNativeCoins:      staticExtra.IsNativeCoins,
		oraclePrices:       [2]*uint256.Int{uint256.MustFromBig(extra.OraclePrices[0]), uint256.MustFromBig(extra.OraclePrices[1])},
		priceDiffThreshold: [2]*uint256.Int{uint256.MustFromBig(extra.PriceDiffThreshold[0]), uint256.MustFromBig(extra.PriceDiffThreshold[1])},
	}
	s.nCoins.SetUint64(uint64(n))
	for i := range n {
		if err := setU256(&s.reserves[i], curveBaseSimulator.Info.Reserves[i]); err != nil {
			return nil, err
		}
		if err := setU256(&s.rates[i], curveBaseSimulator.Rates[i]); err != nil {
			return nil, err
		}
		if err := setU256(&s.multipliers[i], curveBaseSimulator.Multipliers[i]); err != nil {
			return nil, err
		}
	}
	for _, f := range []struct {
		dst *uint256.Int
		src *big.Int
	}{
		{&s.initialA, curveBaseSimulator.InitialA}, {&s.futureA, curveBaseSimulator.FutureA},
		{&s.swapFee, curveBaseSimulator.Info.SwapFee}, {&s.adminFee, curveBaseSimulator.AdminFee},
		{&s.aPrecision, curveBaseSimulator.APrecision},
	} {
		if err := setU256(f.dst, f.src); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func setU256(dst *uint256.Int, src *big.Int) error {
	if src == nil || src.Sign() < 0 || dst.SetFromBig(src) {
		return errInvalidState
	}
	return nil
}

func (t *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *t
	cloned.Info.Reserves = slices.Clone(t.Info.Reserves)
	cloned.reserves = slices.Clone(t.reserves)
	return &cloned
}

func (t *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	i, j := t.GetTokenIndex(param.TokenAmountIn.Token), t.GetTokenIndex(param.TokenOut)
	if i < 0 || j < 0 {
		return nil, fmt.Errorf("tokenIndexFrom %v or TokenOutIndex %v is not correct", i, j)
	}
	var dx uint256.Int
	if param.TokenAmountIn.Amount.Sign() < 0 || dx.SetFromBig(param.TokenAmountIn.Amount) {
		return nil, errInvalidState
	}
	var dy, fee uint256.Int
	if err := t.getDy(i, j, &dx, &dy, &fee); err != nil {
		return nil, err
	}
	if dy.IsZero() {
		return nil, fmt.Errorf("tokenIndexFrom %v or TokenOutIndex %v is not correct", i, j)
	}

	var updatedBalances [maxCoins]uint256.Int
	t.prepareUpdateBalance(i, j, &dx, &dy, updatedBalances[:len(t.reserves)])
	if err := t.checkPriceDiff(updatedBalances[:len(t.reserves)]); err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: param.TokenOut, Amount: dy.ToBig()},
		Fee:            &pool.TokenAmount{Token: param.TokenOut, Amount: fee.ToBig()},
		Gas:            t.gas,
	}, nil
}

// UpdateBalance is curve/base's: the output side also loses the admin share of the swap fee, taken
// as outputAmount * fee / FEE_DENOMINATOR * admin_fee / FEE_DENOMINATOR.
func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	var in, out, adminShare uint256.Int
	in.SetFromBig(params.TokenAmountIn.Amount)
	out.SetFromBig(params.TokenAmountOut.Amount)
	adminShare.Div(adminShare.Mul(&out, &t.swapFee), feeDenominator)
	adminShare.Div(adminShare.Mul(&adminShare, &t.adminFee), feeDenominator)
	out.Add(&out, &adminShare)
	for i, token := range t.Info.Tokens {
		if token == params.TokenAmountIn.Token {
			t.reserves[i].Add(&t.reserves[i], &in)
			t.Info.Reserves[i] = t.reserves[i].ToBig()
		}
		if token == params.TokenAmountOut.Token {
			t.reserves[i].Sub(&t.reserves[i], &out)
			t.Info.Reserves[i] = t.reserves[i].ToBig()
		}
	}
}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	var fromId = t.GetTokenIndex(tokenIn)
	var toId = t.GetTokenIndex(tokenOut)

	meta := Meta{
		TokenInIndex:  fromId,
		TokenOutIndex: toId,
		Underlying:    false,
		BlockNumber:   t.Info.BlockNumber,
	}
	if len(t.isNativeCoins) == len(t.Info.Tokens) {
		meta.TokenInIsNative = t.isNativeCoins[fromId]
		meta.TokenOutIsNative = t.isNativeCoins[toId]
	}

	return meta
}

func (s *PoolSimulator) SwapReceiveNativeIn(tokenIn, tokenOut string, _ valueobject.ChainID) bool {
	meta := s.GetMetaInfo(tokenIn, tokenOut).(Meta)
	return meta.TokenInIsNative
}

func (s *PoolSimulator) SwapReturnNativeOut(tokenIn, tokenOut string, _ valueobject.ChainID) bool {
	meta := s.GetMetaInfo(tokenIn, tokenOut).(Meta)
	return meta.TokenOutIsNative
}

// prepareUpdateBalance mirrors the on-chain balance update after exchange, writing the post-swap
// stored balances that checkPriceDiff operates on into updatedBalances.
func (t *PoolSimulator) prepareUpdateBalance(i, j int, dx, dy *uint256.Int, updatedBalances []uint256.Int) {
	fd := big256.TenPow(10)

	// admin_fee = outAmt * swapFee / fd * adminFee / fd
	var out uint256.Int
	big256.MulDivDown(&out, dy, &t.swapFee, fd)
	big256.MulDivDown(&out, &out, &t.adminFee, fd)
	out.Add(dy, &out) // balance decrease = user output + admin fee portion

	copy(updatedBalances, t.reserves)
	updatedBalances[i].Add(&t.reserves[i], dx)
	updatedBalances[j].Sub(&t.reserves[j], &out)
}

func (t *PoolSimulator) checkPriceDiff(updatedBalances []uint256.Int) error {
	if len(updatedBalances) != 2 ||
		len(t.decimals) != 2 ||
		len(t.multipliers) != 2 {
		return ErrInvalidValue
	}

	// dx = $100 worth of each token in raw units: 100 * 10^(18+decimals) / oraclePrice
	hundred := uint256.NewInt(100)
	dx0 := big256.MulDivDown(new(uint256.Int), hundred, big256.TenPow(int(t.decimals[0])+18), t.oraclePrices[0])
	dx1 := big256.MulDivDown(hundred, hundred, big256.TenPow(int(t.decimals[1])+18), t.oraclePrices[1])

	// Both directions quote the same post-swap balances, so they share A and D.
	var xp [2]uint256.Int
	if err := t.xpMem(updatedBalances, xp[:]); err != nil {
		return err
	}
	a, err := t._A()
	if err != nil {
		return err
	}
	var d uint256.Int
	if err := t.getD(xp[:], &a, &d); err != nil {
		return err
	}
	var dy0, dy1 uint256.Int
	if err := t.getDyWithoutFee(xp[:], &a, &d, 0, 1, dx0, &dy1); err != nil {
		return err
	}
	if err := t.getDyWithoutFee(xp[:], &a, &d, 1, 0, dx1, &dy0); err != nil {
		return err
	}

	mul0 := &t.multipliers[0]
	mul1 := &t.multipliers[1]

	// price0 = (dx1 * mul1 * oracle1) / (dy0 * mul0)  — implied price of token0 in oracle units
	dx1.Mul(dx1, mul1)                                             // dx1 = dx1_xp
	dy0.Mul(&dy0, mul0)                                            // dy0 = dy0_xp
	price0 := big256.MulDivDown(dx1, dx1, t.oraclePrices[1], &dy0) // reuse dx1 as price0

	// price1 = (dx0 * mul0 * oracle0) / (dy1 * mul1)  — implied price of token1 in oracle units
	dx0.Mul(dx0, mul0)                                             // dx0 = dx0_xp
	dy1.Mul(&dy1, mul1)                                            // dy1 = dy1_xp
	price1 := big256.MulDivDown(dx0, dx0, t.oraclePrices[0], &dy1) // reuse dx0 as price1

	// check: |price - oracle| * 1e18 <= oracle * threshold
	var diff, rhs uint256.Int
	if price0.Gt(t.oraclePrices[0]) {
		diff.Sub(price0, t.oraclePrices[0])
	} else {
		diff.Sub(t.oraclePrices[0], price0)
	}
	diff.Mul(&diff, big256.BONE)
	rhs.Mul(t.oraclePrices[0], t.priceDiffThreshold[0])
	if diff.Gt(&rhs) {
		return ErrPriceDiffToken0
	}

	if price1.Gt(t.oraclePrices[1]) {
		diff.Sub(price1, t.oraclePrices[1])
	} else {
		diff.Sub(t.oraclePrices[1], price1)
	}
	diff.Mul(&diff, big256.BONE)
	rhs.Mul(t.oraclePrices[1], t.priceDiffThreshold[1])
	if diff.Gt(&rhs) {
		return ErrPriceDiffToken1
	}

	return nil
}
