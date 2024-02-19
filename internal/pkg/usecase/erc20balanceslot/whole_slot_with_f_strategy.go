package erc20balanceslot

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"strings"

	"github.com/ALTree/bigfloat"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/jsonrpc"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

var (
	maxUint256 = new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil), big.NewInt(1))
)

func randomizeAddress() common.Address {
	a := common.Address{}
	for i := range a {
		a[i] = byte(rand.Intn(256))
	}
	return a
}

// WholeSlotWithFStrategy For a ERC20 token and a wallet, find a slot so that
//
// * balanceOf() returns a number which is large enough
//
// * and transfer() executes successfully
//
// This strategy assumes that the real balance is calculated from a whole slot value.
type WholeSlotWithFStrategy struct {
	rpcClient *rpc.Client
	ethClient *ethclient.Client
	wallet    common.Address
}

func NewWholeSlotWithFStrategy(rpcClient *rpc.Client, wallet common.Address) *WholeSlotWithFStrategy {
	return &WholeSlotWithFStrategy{
		rpcClient: rpcClient,
		ethClient: ethclient.NewClient(rpcClient),
		wallet:    wallet,
	}
}

func (*WholeSlotWithFStrategy) Name(_ ProbeStrategyExtraParams) string {
	return "whole_slot_with_f"
}

func (p *WholeSlotWithFStrategy) ProbeBalanceSlot(ctx context.Context, token common.Address, _ ProbeStrategyExtraParams) (*entity.ERC20BalanceSlot, error) {
	logger.Infof(ctx, "[%s] probing balance slot for wallet %s in token %s", p.Name(nil), p.wallet, token)

	blockNumber, err := p.ethClient.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get latest block number %w", err)
	}
	blockNumberHex := hexutil.EncodeUint64(blockNumber)

	/*
		Step 1: Assume that there is exactly one slot where wallet's balance is stored.
				Isolate the slot by subtracting the set of SLOAD slots when balanceOf(wallet)
				with the set of SLOAD slots when balanceOf(anotherWallet).
	*/
	sload, err := p.isolateExactOneBalanceSlot(blockNumberHex, token)
	if err != nil {
		logger.Debugf(ctx, "could not isolate exact 1 balance slot %s", err)
		return nil, fmt.Errorf("could not isolate exact 1 balance slot %w", err)
	}
	logger.Debugf(ctx, "slot = %s", sload.Slot)

	/*
		Step 2: Analyze the relationship between the slot value and the output of balanceOf().
				We assume that
					* the token contract handled overflow and underflow arithmetic
					* the relationship between the slot value and the output of balanceOf() is an increasing or decreasing function.

				Let f(v) is the relationship between the slot value and the output of balanceOf()
					* f(v) > 0 when balanceOf() with overrided value v returns successfully and f(v) is the output itself
					* f(v) = 0 when balanceOf() with overrided value v returns an error

				So when evaluating f(v) at N increasing values, there is atmost 1 continous subsequence whose elements are >0 and the sequence is either increasing or decreasing.
				The output of this step is the continous subsequence, or nothing at all.
	*/
	const (
		subsequenceMinLength = 4
	)
	var (
		overrideValues       []*big.Int
		balances             []*big.Int
		incOrDec, ok         bool
		sequenceBegin        int
		sequenceEndInclusive int
	)
	for _, incOrDec = range []bool{true, false} {
		for _, n := range []int{4, 8, 16, 32} {
			overrideValues = geometricSequenceN(n, maxUint256)
			balances = p.balanceOfWithOverridedSlot(ctx, blockNumberHex, token, sload, overrideValues)
			begin, length, err := longestMonotonicAndPositiveSequence(balances, incOrDec)
			if err != nil {
				// the pattern is broken a n[i], don't analyze with n[i+1:]
				break
			}
			if length > subsequenceMinLength {
				ok = true
				sequenceBegin = begin
				sequenceEndInclusive = begin + length - 1
				break
			}
		}
		if ok {
			break
		}
	}
	if !ok {
		return nil, fmt.Errorf("could not analyze the relationship between slot value and balanceOf()")
	}

	/*
		Step 3: From the continous subsequence, find the value which yields maximum balance by doing binary search.
	*/
	var (
		prevBalance    = balances[sequenceBegin]
		lower, upper   *big.Int
		values         []*big.Int
		largerBalances []*big.Int
		anotherWallet  = randomizeAddress()
	)
	if incOrDec {
		lower = overrideValues[sequenceBegin]
		if sequenceEndInclusive+1 < len(overrideValues) {
			upper = overrideValues[sequenceEndInclusive+1]
		} else {
			upper = maxUint256
		}
	} else {
		upper = overrideValues[sequenceEndInclusive]
		if sequenceBegin > 0 {
			lower = overrideValues[sequenceBegin-1]
		} else {
			lower = big.NewInt(0)
		}
	}
	for i := 0; lower.Cmp(upper) < 0 && i < 256; i++ {
		// mid = (lower + upper)/2
		mid := new(big.Int).Add(lower, upper)
		mid = mid.Rsh(mid, 1)

		balance := p.balanceOfWithOverridedSlot(ctx, blockNumberHex, token, sload, []*big.Int{mid})[0]

		if balance.Cmp(prevBalance) > 0 && p.canTransferAmountWithOverride(ctx, blockNumberHex, token, anotherWallet, balance, sload, mid) {
			// stop if could not improve
			if len(largerBalances) > 0 && balance.Cmp(largerBalances[len(largerBalances)-1]) == 0 {
				break
			}
			prevBalance = balance
			values = append(values, mid)
			largerBalances = append(largerBalances, balance)
			if incOrDec {
				lower = mid
			} else {
				upper = mid
			}
		} else {
			if incOrDec {
				upper = mid
			} else {
				lower = mid
			}
		}
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("could not probe token %s", token)
	}

	return &entity.ERC20BalanceSlot{
		Token:          strings.ToLower(token.String()),
		Wallet:         strings.ToLower(p.wallet.String()),
		Found:          true,
		BalanceSlot:    strings.ToLower(sload.Slot),
		PreferredValue: strings.ToLower(common.BigToHash(values[len(values)-1]).String()),
	}, nil
}

func (p *WholeSlotWithFStrategy) isolateExactOneBalanceSlot(blockNumber string, token common.Address) (*tracingResultSload, error) {
	_, sloads, err := sloadTrace(p.rpcClient, blockNumber, token, p.wallet)
	if err != nil {
		return nil, fmt.Errorf("could not trace sloads of the call balanceOf(wallet) %w", err)
	}
	anotherWallet := randomizeAddress()
	_, anotherSloads, err := sloadTrace(p.rpcClient, blockNumber, token, anotherWallet)
	if err != nil {
		return nil, fmt.Errorf("could not trace sloads of the call balanceOf(anotherWallet) %w", err)
	}
	for slot := range anotherSloads {
		delete(sloads, slot)
	}

	if len(sloads) != 1 {
		return nil, fmt.Errorf("expected 1 sload")
	}

	var sload tracingResultSload
	for _, s := range sloads {
		sload = s
		break
	}

	return &sload, nil
}

func (p *WholeSlotWithFStrategy) balanceOfWithOverridedSlot(ctx context.Context, blockNumberHex string, token common.Address, sload *tracingResultSload, overrideValues []*big.Int) []*big.Int {
	data, err := abis.ERC20.Pack("balanceOf", p.wallet)
	if err != nil {
		panic("must pack balanceOf()")
	}
	dataEncoded := hexutil.Encode(data)
	balances := make([]*big.Int, 0, len(overrideValues))
	for _, value := range overrideValues {
		result, err := jsonrpc.EthCall(
			p.rpcClient,
			&jsonrpc.EthCallCalldataParam{
				From: common.Address{}.String(),
				To:   token.String(),
				Gas:  gasLimit,
				Data: dataEncoded,
			},
			blockNumberHex,
			map[common.Address]jsonrpc.OverrideAccount{
				common.HexToAddress(sload.Address): {
					StateDiff: map[common.Hash]common.Hash{
						common.HexToHash(sload.Slot): common.BigToHash(value),
					},
				},
			},
		)
		if err != nil {
			logger.Debugf(ctx, "    could not eth_call: %s", err)
			balances = append(balances, big.NewInt(0))
			continue
		}
		balance := new(big.Int).SetBytes(common.HexToHash(*result).Bytes())
		balances = append(balances, balance)
		logger.Debugf(ctx, "slot value = %s, balance = %s", common.BigToHash(value), balance)
	}
	return balances
}

func (p *WholeSlotWithFStrategy) canTransferAmountWithOverride(ctx context.Context, blockNumber string, token, to common.Address, amount *big.Int, sload *tracingResultSload, overrideValue *big.Int) bool {
	transferData, err := abis.ERC20.Pack("transfer", to, amount)
	if err != nil {
		panic("must pack transfer()")
	}
	result, err := jsonrpc.EthCall(
		p.rpcClient,
		&jsonrpc.EthCallCalldataParam{
			From: p.wallet.String(),
			To:   token.String(),
			Gas:  gasLimit,
			Data: hexutil.Encode(transferData),
		},
		blockNumber,
		map[common.Address]jsonrpc.OverrideAccount{
			common.HexToAddress(sload.Address): {
				StateDiff: map[common.Hash]common.Hash{
					common.HexToHash(sload.Slot): common.BigToHash(overrideValue),
				},
			},
		},
	)
	ok := err == nil && common.HexToHash(*result) == common.HexToHash("0x1")
	if !ok {
		logger.Debugf(ctx, "err = %v", err)
	} else {
		logger.Debugf(ctx, "result = %s err = %v", *result, err)
	}
	return ok
}

func sloadTrace(rpcClient *rpc.Client, blockNumber string, token, wallet common.Address) (*tracingResult, map[common.Hash]tracingResultSload, error) {
	data, err := abis.ERC20.Pack("balanceOf", wallet)
	if err != nil {
		return nil, nil, err
	}
	tracingResult := new(tracingResult)
	err = jsonrpc.DebugTraceCall(
		rpcClient,
		&jsonrpc.DebugTraceCallCalldataParam{
			From: common.Address{}.String(),
			To:   token.String(),
			Gas:  gasLimit,
			Data: hexutil.Encode(data),
		},
		blockNumber,
		&jsonrpc.DebugTraceCallTracerConfigParam{
			Tracer: string(sloadTracerMinified),
		},
		tracingResult,
	)
	if err != nil {
		return nil, nil, err
	}
	// only takes unique sload slot whose address = address
	uniqSloads := make(map[common.Hash]tracingResultSload)
	for _, sload := range tracingResult.Sloads {
		if common.HexToAddress(sload.Address) == token {
			if _, ok := uniqSloads[common.HexToHash(sload.Slot)]; !ok {
				uniqSloads[common.HexToHash(sload.Slot)] = sload
			}
		}
	}
	return tracingResult, uniqSloads, nil
}

// Generate an N-element sequence [1, c^1, c^2, ..., c^(n-1)] where c^(n-1) ≈ endInclusive.
func geometricSequenceN(n int, endInclusive *big.Int) []*big.Int {
	if n < 2 {
		panic("n must be >= 2")
	}
	if n == 2 {
		// multiplication factor c = endInclusive
		return []*big.Int{
			big.NewInt(1),
			new(big.Int).Set(endInclusive),
		}
	}

	// exp = 1/(n - 1)
	exp := big.NewFloat(1)
	exp.Quo(exp, big.NewFloat(float64(n-1)))

	// multiplication factor c = endInclusive^(1/(n-1))
	c := new(big.Float).SetInt(endInclusive)
	c = bigfloat.Pow(c, exp)

	seq := make([]*big.Int, 0, n)
	seq = append(seq, big.NewInt(1))

	acc := big.NewFloat(1)
	for i := 1; i < n; i++ {
		acc.Mul(acc, c)
		ni, _ := acc.Int(new(big.Int))
		seq = append(seq, ni)
	}

	return seq
}

// Find longest increasing or decreasing subsequence. Returns error if there is more than 1 subsequence.
func longestMonotonicAndPositiveSequence(S []*big.Int, incOrDec bool) (begin int, length int, err error) {
	if len(S) == 0 {
		return 0, 0, nil
	}

	zero := big.NewInt(0)

	L := make([]int, len(S))
	for i := range S {
		if S[i].Cmp(zero) > 0 {
			L[i] = 1
		}
	}

	var cmp int
	if incOrDec {
		cmp = 1
	} else {
		cmp = -1
	}

	for i := 1; i < len(S); i++ {
		if S[i].Cmp(zero) > 0 && S[i].Cmp(S[i-1]) == cmp {
			L[i] = L[i-1] + 1
		}
	}

	end := -1
	for i := range L {
		if L[i] > length {
			length = L[i]
			end = i
		}
	}
	if end == -1 {
		return 0, 0, nil
	}

	begin = end - L[end] + 1
	for i := range L {
		if (i < begin || i > end) && L[i] > 0 {
			return 0, 0, fmt.Errorf("expected only 1 subsequence")
		}
	}

	return begin, length, nil
}
