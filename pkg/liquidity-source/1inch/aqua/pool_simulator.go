package oneinchaqua

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// PoolSimulator quotes one Aqua strategy of a SwapVM router. Tokens follow the entity pool order.
type PoolSimulator struct {
	pool.Pool
	State       state
	StaticExtra StaticExtra

	program   []Instruction
	launch    bool
	limitKeys [2]string
	token0Lt  bool
}

var (
	_ = pool.RegisterUseSwapLimit(valueobject.ExchangeOneInchAqua)
	_ = pool.RegisterFactory(DexType, func(params pool.FactoryParams) (*PoolSimulator, error) {
		return NewPoolSimulator(params.EntityPool)
	})
)

var (
	_ pool.IPoolSimulator         = (*PoolSimulator)(nil)
	_ pool.ISwapLimitMergePolicy  = (*PoolSimulator)(nil)
	_ pool.IPoolExactOutSimulator = (*PoolSimulator)(nil)
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	if len(entityPool.Tokens) != 2 || len(entityPool.Reserves) != 2 {
		return nil, ErrInvalidState
	}
	s := &PoolSimulator{}
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	s.State = extra.state()
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &s.StaticExtra); err != nil {
		return nil, err
	}
	order, err := s.StaticExtra.order()
	if err != nil {
		return nil, err
	}
	s.launch = s.StaticExtra.Launch
	if s.program, err = ParseOrder(order, s.launch); err != nil {
		return nil, err
	}

	tokens := []string{strings.ToLower(entityPool.Tokens[0].Address), strings.ToLower(entityPool.Tokens[1].Address)}
	if tokens[0] == tokens[1] {
		return nil, ErrInvalidState
	}
	s.token0Lt = tokens[0] < tokens[1]
	maker := strings.ToLower(s.StaticExtra.Maker)
	s.limitKeys = [2]string{limitKey(maker, tokens[0]), limitKey(maker, tokens[1])}
	s.StaticExtra.Maker = maker
	s.StaticExtra.App = strings.ToLower(s.StaticExtra.App)
	if !common.IsHexAddress(s.StaticExtra.Aqua) || common.HexToAddress(s.StaticExtra.Aqua) == (common.Address{}) {
		return nil, ErrInvalidState
	}
	s.StaticExtra.Aqua = strings.ToLower(s.StaticExtra.Aqua)

	reserves := make([]*big.Int, 2)
	for i, r := range entityPool.Reserves {
		reserves[i] = bignumber.NewBig(r)
	}
	s.Info = pool.PoolInfo{
		Address:     strings.ToLower(entityPool.Address),
		Exchange:    entityPool.Exchange,
		Type:        entityPool.Type,
		Tokens:      tokens,
		Reserves:    reserves,
		BlockNumber: entityPool.BlockNumber,
	}
	return s, nil
}

func (s *StaticExtra) order() (*Order, error) {
	traits, err := parseUint256Hex(s.Traits)
	if err != nil {
		return nil, ErrInvalidState
	}
	data, err := hexutil.Decode(s.Data)
	if err != nil {
		return nil, ErrInvalidState
	}
	return &Order{Maker: common.HexToAddress(s.Maker), Traits: *traits, Data: data}, nil
}

func limitKey(maker, token string) string { return maker + ":" + token }

// live reports whether the strategy can be quoted at all.
func (s *PoolSimulator) live() bool { return !s.State.Docked && !s.State.Inactive }

// snapshotLimit is min(maker wallet balance, allowance to Aqua) of token i at the last wallet read.
func (s *PoolSimulator) snapshotLimit(i int) uint256.Int {
	bal, allow := &s.State.Wallet[i], &s.State.Allowance[i]
	if allow.Lt(bal) {
		return *allow
	}
	return *bal
}

// limitOf is the fillable cap of token i: the shared limit when the caller tracks one, else the pool's
// own wallet snapshot. A key the shared limit does not hold means the maker has no balance.
func (s *PoolSimulator) limitOf(i int, limit pool.SwapLimit) uint256.Int {
	if limit == nil {
		return s.snapshotLimit(i)
	}
	v := limit.GetLimit(s.limitKeys[i])
	if v == nil || v.Sign() <= 0 {
		return uint256.Int{}
	}
	var u uint256.Int
	if u.SetFromBig(v) {
		return *big256.UMax // a limit above 2^256 is no cap
	}
	return u
}

// quoteAlloc is everything a successful CalcAmountOut hands out, in one allocation: the result, its two
// token amounts, the amounts as big integers backed by the same object, the swap info and the buffer the
// fee amounts are collected in.
type quoteAlloc struct {
	result    pool.CalcAmountOutResult
	out, fee  pool.TokenAmount
	outBig    big.Int
	feeBig    big.Int
	outWords  [4]big.Word
	feeWords  [4]big.Word
	info      SwapInfo
	run       quoteResult
	feeBuffer [4]uint256.Int
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	in, out := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if in < 0 || out < 0 || in == out {
		return nil, ErrInvalidToken
	}
	if params.TokenAmountIn.Amount == nil || params.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}
	var amountIn uint256.Int
	if amountIn.SetFromBig(params.TokenAmountIn.Amount) {
		return nil, ErrInvalidAmountIn
	}
	if err := s.checkQuotable(); err != nil {
		return nil, err
	}
	q := new(quoteAlloc)
	if err := s.quote(q, in, out, &amountIn, params.Limit); err != nil {
		return nil, err
	}
	return &q.result, nil
}

// checkQuotable refuses a strategy that is terminal, has no wallet snapshot or holds a mixed-block state.
func (s *PoolSimulator) checkQuotable() error {
	if !s.live() {
		return ErrStrategyTerminal
	}
	if s.State.WalletBlock == 0 {
		return ErrNoWalletSnapshot
	}
	if s.State.RawBlock != s.State.WalletBlock {
		return ErrStateMixedBlock
	}
	return nil
}

// quote prices an exact-in swap of amountIn (above zero) and fills q.result.
func (s *PoolSimulator) quote(q *quoteAlloc, in, out int, amountIn *uint256.Int, limit pool.SwapLimit) error {
	q.run.FeeIns = q.feeBuffer[:0]
	tokenInLt := (in == 0) == s.token0Lt
	if err := runProgram(&q.run, s.program, &s.State.Raw[in], &s.State.Raw[out], amountIn, tokenInLt); err != nil {
		return err
	}
	return s.settle(q, in, out, amountIn, limit)
}

// settle turns the program result in q.run for amountIn into q.result. It is the one place the caps, the
// fee pulls and the push bound are applied, for CalcAmountOut and for CalcAmountIn.
func (s *PoolSimulator) settle(q *quoteAlloc, in, out int, amountIn *uint256.Int, limit pool.SwapLimit) error {
	rawIn, rawOut := &s.State.Raw[in], &s.State.Raw[out]
	amountOut := &q.run.AmountOut
	if amountOut.IsZero() {
		return ErrZeroAmountOut
	}
	if amountOut.Gt(rawOut) {
		return ErrInsufficientRaw
	}
	if limitOut := s.limitOf(out, limit); amountOut.Gt(&limitOut) {
		return ErrInsufficientWallet
	}

	info := &q.info
	if err := s.collectFees(info, q.run.FeeIns, rawIn, in, limit); err != nil {
		return err
	}
	// The push credits the full amountIn on top of what the fee pulls left in the strategy.
	var after uint256.Int
	after.Sub(rawIn, &info.FeeCollected)
	if _, overflow := after.AddOverflow(&after, amountIn); overflow || after.Gt(uMaxUint248) {
		return ErrOverflow
	}
	info.AmountIn, info.AmountOut = *amountIn, *amountOut

	gas := int64(defaultGas)
	if !info.FeeCollected.IsZero() {
		gas = defaultGasFeeCollected
	}
	q.out = pool.TokenAmount{Token: s.Info.Tokens[out], Amount: big256.ToBigBacked(&q.outBig, &q.outWords, amountOut)}
	q.fee = pool.TokenAmount{Token: s.Info.Tokens[in], Amount: big256.ToBigBacked(&q.feeBig, &q.feeWords, &info.FeeCollected)}
	q.result = pool.CalcAmountOutResult{TokenAmountOut: &q.out, Fee: &q.fee, Gas: gas, SwapInfo: info}
	return nil
}

// collectFees decides which protocol fee pulls the router completes, in pull order, and records the total
// fee and the collected part in info. A pull needs the strategy's tokenIn balance and the maker's wallet
// limit to cover it, after the pulls before it. The current router skips a pull it cannot make; the launch
// router reverts, so the quote is refused.
func (s *PoolSimulator) collectFees(info *SwapInfo, feeIns []uint256.Int, rawIn *uint256.Int, in int, limit pool.SwapLimit) error {
	if len(feeIns) == 0 {
		return nil
	}
	limitIn := s.limitOf(in, limit)
	for i := range feeIns {
		fee := &feeIns[i]
		info.FeeIn.Add(&info.FeeIn, fee) // each fee is a share of what is left of amountIn: the sum stays below it
		if fee.IsZero() {
			continue
		}
		var rawLeft, limitLeft uint256.Int
		rawLeft.Sub(rawIn, &info.FeeCollected) // collected never exceeds either bound: each pull is checked below
		limitLeft.Sub(&limitIn, &info.FeeCollected)
		if !fee.Gt(&rawLeft) && !fee.Gt(&limitLeft) {
			info.FeeCollected.Add(&info.FeeCollected, fee)
			continue
		}
		if s.launch {
			return ErrFeeShortfall
		}
	}
	return nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	return &cloned
}

// UpdateBalance applies a swap on the strategy and on the pool's own wallet snapshot, and on the shared
// limit when the router tracks one.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	info, ok := params.SwapInfo.(*SwapInfo)
	if !ok || info == nil {
		return
	}
	in, out := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if in < 0 || out < 0 || in == out {
		return
	}

	rawIn, rawOut := &s.State.Raw[in], &s.State.Raw[out]
	rawIn.Sub(rawIn, &info.FeeCollected)
	rawIn.Add(rawIn, &info.AmountIn)
	rawOut.Sub(rawOut, &info.AmountOut)

	s.updateWallet(in, out, info)

	s.publishReserves()

	if params.SwapLimit == nil {
		return
	}
	credit := new(big.Int)
	if s.State.Allowance[in].Cmp(uUnlimitedAllowance) >= 0 {
		info.AmountIn.IntoBig(&credit)
	}
	if _, _, err := params.SwapLimit.UpdateLimit(s.limitKeys[out], s.limitKeys[in], info.AmountOut.ToBig(), credit); err != nil {
		return
	}
	if info.FeeIn.IsZero() {
		return
	}
	// UpdateLimit refuses a decrease above the balance instead of flooring at zero.
	cur := params.SwapLimit.GetLimit(s.limitKeys[in])
	if cur == nil {
		return
	}
	debit := info.FeeIn.ToBig()
	if cur.Cmp(debit) < 0 {
		debit = new(big.Int).Set(cur)
	}
	if debit.Sign() > 0 {
		_, _, _ = params.SwapLimit.UpdateLimit(s.limitKeys[in], s.limitKeys[in], debit, new(big.Int))
	}
}

// updateWallet mirrors the swap on the pool's own wallet snapshot: tokenOut leaves the wallet through a
// pull (which also lowers a finite allowance), tokenIn arrives by the taker's push minus the fee that
// left through its pull.
func (s *PoolSimulator) updateWallet(in, out int, info *SwapInfo) {
	subFloor(&s.State.Wallet[out], &info.AmountOut)
	if a := &s.State.Allowance[out]; a.Lt(uUnlimitedAllowance) {
		subFloor(a, &info.AmountOut)
	}
	subFloor(&s.State.Wallet[in], &info.FeeCollected)
	s.State.Wallet[in].Add(&s.State.Wallet[in], &info.AmountIn)
	if a := &s.State.Allowance[in]; a.Lt(uUnlimitedAllowance) {
		subFloor(a, &info.FeeCollected)
	}
}

// subFloor sets z = max(z-x, 0).
func subFloor(z, x *uint256.Int) {
	if z.Lt(x) {
		z.Clear()
		return
	}
	z.Sub(z, x)
}

// reserveBuf holds the two published reserves in one allocation.
type reserveBuf struct {
	values   [2]big.Int
	words    [2][4]big.Word
	pointers [2]*big.Int
}

// publishReserves replaces the pool reserves with the fillable amount of each token, min(strategy balance,
// wallet limit). The slice is new, so a clone made before keeps its own.
func (s *PoolSimulator) publishReserves() {
	b := new(reserveBuf)
	for i := range b.pointers {
		limit := s.snapshotLimit(i)
		fillable := &s.State.Raw[i]
		if limit.Lt(fillable) {
			fillable = &limit
		}
		b.pointers[i] = big256.ToBigBacked(&b.values[i], &b.words[i], fillable)
	}
	s.Info.Reserves = b.pointers[:]
}

// CalculateLimit publishes the maker's wallet limit per token. A strategy that is terminal or has no
// wallet snapshot yet publishes nothing: under the min merge a key without a read would pin the limit
// of every other strategy of the maker at zero.
func (s *PoolSimulator) CalculateLimit() map[string]*big.Int {
	if !s.live() || s.State.WalletBlock == 0 {
		return nil
	}
	l0, l1 := s.snapshotLimit(0), s.snapshotLimit(1)
	return map[string]*big.Int{s.limitKeys[0]: l0.ToBig(), s.limitKeys[1]: l1.ToBig()}
}

// GetLimitMergePolicy several strategies of one maker publish snapshots of the same wallet taken at
// different times; the lowest can only understate.
func (s *PoolSimulator) GetLimitMergePolicy(string) pool.SwapLimitMergePolicy {
	return pool.SwapLimitMergeMin
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{
		BlockNumber:     s.Info.BlockNumber,
		ApprovalAddress: s.StaticExtra.Aqua,
		Router:          s.StaticExtra.App,
		Maker:           s.StaticExtra.Maker,
		Traits:          s.StaticExtra.Traits,
		Data:            s.StaticExtra.Data,
	}
}

func (s *PoolSimulator) GetApprovalAddress(_, _ string) string { return s.StaticExtra.Aqua }

// maxRefineSteps bounds the walk from the closed-form amountIn to the smallest one that quotes the amount
// out. The closed form is exact, so the walk does not move; it guards the result against a rounding rule
// the inversion has wrong.
const maxRefineSteps = 8

// quoteInAlloc is everything a successful CalcAmountIn hands out, in one allocation: the exact-in quote it
// is built from plus its own result.
type quoteInAlloc struct {
	quoteAlloc
	result    pool.CalcAmountInResult
	amountIn  pool.TokenAmount
	remaining pool.TokenAmount
	inBig     big.Int
	inWords   [4]big.Word
	zeroBig   big.Int
}

// CalcAmountIn returns the smallest amountIn for which CalcAmountOut gives at least TokenAmountOut, and
// the CalcAmountOut result of that amountIn. Swaps are sent exact-in, so the router's exact-out rounding is
// not ported. The pipeline is inverted from xycSwap outwards (invertFees).
func (s *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	in, out := s.GetTokenIndex(params.TokenIn), s.GetTokenIndex(params.TokenAmountOut.Token)
	if in < 0 || out < 0 || in == out {
		return nil, ErrInvalidToken
	}
	if params.TokenAmountOut.Amount == nil || params.TokenAmountOut.Amount.Sign() <= 0 {
		return nil, ErrInvalidAmountOut
	}
	var target uint256.Int
	if target.SetFromBig(params.TokenAmountOut.Amount) {
		return nil, ErrInvalidAmountOut
	}
	if err := s.checkQuotable(); err != nil {
		return nil, err
	}

	// No amountIn can pay out more than the strategy balance or the wallet limit; refuse before the search
	// so that the error does not depend on whether the search overflows.
	rawIn, rawOut := &s.State.Raw[in], &s.State.Raw[out]
	if target.Gt(rawOut) {
		return nil, ErrInsufficientRaw
	}
	if limitOut := s.limitOf(out, params.Limit); target.Gt(&limitOut) {
		return nil, ErrInsufficientWallet
	}

	tokenInLt := (in == 0) == s.token0Lt
	effIn, effOut := *rawIn, *rawOut
	if err := effectiveBalances(s.program, &effIn, &effOut, tokenInLt); err != nil {
		return nil, err
	}
	if effIn.IsZero() || effOut.IsZero() {
		return nil, ErrZeroBalance
	}
	if !target.Lt(&effOut) { // xycSwap pays out strictly less than the balance
		return nil, ErrInsufficientRaw
	}

	var den, net, amountIn uint256.Int
	den.Sub(&effOut, &target)
	if !mulDivCeil(&net, &target, &effIn, &den) {
		return nil, ErrOverflow
	}
	if err := invertFees(&amountIn, s.program, &net); err != nil {
		return nil, err
	}
	q := new(quoteInAlloc)
	if err := refineAmountIn(&q.quoteAlloc, s.program, &amountIn, &target, &effIn, &effOut, tokenInLt); err != nil {
		return nil, err
	}
	// refineAmountIn left the result of amountIn - 1 in q.run: run amountIn itself on the effective balances
	// (no second square root) and settle it exactly like CalcAmountOut does.
	if _, err := forwardOut(&q.quoteAlloc, s.program, &amountIn, &effIn, &effOut, tokenInLt); err != nil {
		return nil, err
	}
	if err := s.settle(&q.quoteAlloc, in, out, &amountIn, params.Limit); err != nil {
		return nil, err
	}
	q.amountIn = pool.TokenAmount{Token: s.Info.Tokens[in], Amount: big256.ToBigBacked(&q.inBig, &q.inWords, &amountIn)}
	q.remaining = pool.TokenAmount{Token: s.Info.Tokens[out], Amount: &q.zeroBig}
	q.result = pool.CalcAmountInResult{
		TokenAmountIn:           &q.amountIn,
		RemainingTokenAmountOut: &q.remaining,
		Fee:                     q.quoteAlloc.result.Fee,
		Gas:                     q.quoteAlloc.result.Gas,
		SwapInfo:                q.quoteAlloc.result.SwapInfo,
	}
	return &q.result, nil
}

// forwardOut runs the program for amountIn on balances that are already effective, into q.run, and returns
// the amountOut. It runs in the buffers of q, so that the search allocates nothing of its own.
func forwardOut(q *quoteAlloc, prog []Instruction, amountIn, effIn, effOut *uint256.Int, tokenInLt bool) (uint256.Int, error) {
	q.run.FeeIns = q.feeBuffer[:0]
	err := runProgramFrom(&q.run, prog, effIn, effOut, amountIn, tokenInLt, true)
	return q.run.AmountOut, err
}

// refineAmountIn moves amountIn to the smallest value whose amountOut reaches target, checking it forward:
// out(amountIn) >= target > out(amountIn-1). The output never falls as amountIn grows, so it walks one wei
// at a time and gives up with ErrNoConvergence after maxRefineSteps steps.
func refineAmountIn(q *quoteAlloc, prog []Instruction, amountIn, target, effIn, effOut *uint256.Int, tokenInLt bool) error {
	for range maxRefineSteps {
		got, err := forwardOut(q, prog, amountIn, effIn, effOut, tokenInLt)
		if err != nil {
			return err
		}
		if got.Lt(target) {
			if _, overflow := amountIn.AddOverflow(amountIn, big256.U1); overflow {
				return ErrOverflow
			}
			continue
		}
		if amountIn.Eq(big256.U1) {
			return nil
		}
		var prev uint256.Int
		prev.SubUint64(amountIn, 1)
		less, err := forwardOut(q, prog, &prev, effIn, effOut, tokenInLt)
		if err != nil {
			return err
		}
		if less.Lt(target) {
			return nil
		}
		*amountIn = prev
	}
	return ErrNoConvergence
}
