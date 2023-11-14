package aevm

import (
	"fmt"
	"math"
	"math/big"
	"strings"

	aevmclient "github.com/KyberNetwork/aevm/client"
	aevmcommon "github.com/KyberNetwork/aevm/common"
	aevmtypes "github.com/KyberNetwork/aevm/types"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	routerentity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/common"
)

const (
	maxNumberOfHoldersToTransfer = 10
)

// AEVMSwapInfo holds related data after a simulation. These data are used for the next simulation of the same pool.
type AEVMSwapInfo struct {
	StateAfter *aevmtypes.StateOverrides `json:"-"`
}

// AEVMPool a AEVM-integrated pool
type AEVMPool struct {
	// Pool address
	Address gethcommon.Address
	// Client to communicate with AEVM server to do simulation
	AEVMClient common.NoDeepClone // aevmclient.Client
	// The state root to ensure that all simulations are consistent
	StateRoot gethcommon.Hash
	// NextSwapInfo state changed after simulation which is use for the next simulation
	NextSwapInfo *AEVMSwapInfo
	// TokenBalanceSlots balance slots needed for the swap simulation
	TokenBalanceSlots common.NoDeepClone // entity.TokenBalanceSlots
}

// AEVMSwapCalls a list of contract calls required for a swap
type AEVMSwapCalls struct {
	// Contract calls before swapping such as approving call
	PreCalls []aevmtypes.SingleCall
	// The swap call itself
	SwapCall aevmtypes.SingleCall
	// Contract calls after swapping such as calling balanceOf to tokenOut to retrieve amountOut
	PostCalls []aevmtypes.SingleCall
}

// Len returns the number of contract calls
func (c *AEVMSwapCalls) Len() int {
	return len(c.PreCalls) + len(c.PostCalls) + 1
}

// List converts contract calls to a list
func (c *AEVMSwapCalls) List() []aevmtypes.SingleCall {
	var calls []aevmtypes.SingleCall
	calls = append(calls, c.PreCalls...)
	calls = append(calls, c.SwapCall)
	calls = append(calls, c.PostCalls...)
	return calls
}

// GetSwapCallResult get the simulation result of the swap call from the list of simulation results
func (c *AEVMSwapCalls) GetSwapCallResult(results []*aevmtypes.CallResult) (*aevmtypes.CallResult, error) {
	if len(results) != c.Len() {
		return nil, fmt.Errorf("an error occurred in call sequence")
	}
	return results[len(c.PreCalls)], nil
}

// AmountOutGetter the strategy to retrieve amount out from a swap
type AmountOutGetter int

const (
	// AmountOutGetterSwapOutput Amount out is the return value of swap function. For example:
	// function exactInputSingle(ExactInputSingleParams calldata params) returns (uint256 amountOut)
	AmountOutGetterSwapOutput AmountOutGetter = iota
	// AmountOutGetterSwapOutputTuple Amount out is a tuple element of the return value of swap function. For example:
	// function swap(bool _swapForY, address _to) returns (uint256 amountXOut, uint256 amountYOut)
	AmountOutGetterSwapOutputTuple
	// AmountOutGetterSwapOutputPackedBytes32 Amount out is one of two int128s packed into a bytes32
	AmountOutGetterSwapOutputPackedBytes32
	// AmountOutGetterDelta Amount out is calculated from the different before and after swapping.
	AmountOutGetterDelta
)

// AmountOutGetterSwapOutputTupleArgs arguments of AmountOutGetterSwapOutputTuple strategy
type AmountOutGetterSwapOutputTupleArgs struct {
	ElementIndex int
}

// AmountOutGetterSwapOutputPackedBytes32Args arguments of AmountOutGetterSwapOutputPackedBytes32 strategy
type AmountOutGetterSwapOutputPackedBytes32Args struct {
	ElementIndex int
}

// AmountOutGetterDeltaArgs arguments of AmountOutGetterDelta strategy
type AmountOutGetterDeltaArgs struct {
	BalanceOfBeforeIndex int
	BalanceOfAfterIndex  int
}

// AEVMSwapStrategy generalizes swapping via AEVM
type AEVMSwapStrategy struct {
	// Check before swapping such as check if pool address is calculated correctly from pool's parameters
	Precheck func() error
	// Contract calls required for swapping
	SwapCalls *AEVMSwapCalls
	// Which way to retrieve amount out
	AmountOutGetter AmountOutGetter
	// How to retrieve amount out
	AmountOutGetterArgs interface{}
}

func (s *AEVMSwapStrategy) getAmountOut(swapResult *aevmtypes.CallResult, results []*aevmtypes.CallResult) (*big.Int, error) {
	var amountOut *big.Int
	switch s.AmountOutGetter {
	case AmountOutGetterSwapOutput:
		if len(swapResult.Data) != 32 {
			return nil, fmt.Errorf("expect output to be uint256")
		}
		amountOut = new(big.Int).SetBytes(swapResult.Data)
	case AmountOutGetterSwapOutputTuple:
		args, ok := s.AmountOutGetterArgs.(AmountOutGetterSwapOutputTupleArgs)
		if !ok {
			return nil, fmt.Errorf("args must be a AmountOutGetterSwapOutputTupleArgs")
		}
		if len(swapResult.Data) < 32*(args.ElementIndex+1) {
			return nil, fmt.Errorf("expect output to be a tuple whose length is atleast %d", 32*(args.ElementIndex+1))
		}
		amountOut = new(big.Int).SetBytes(swapResult.Data[32*args.ElementIndex:][:32])
	case AmountOutGetterSwapOutputPackedBytes32:
		if len(swapResult.Data) != 32 {
			return nil, fmt.Errorf("expect output to be bytes32")
		}
		args, ok := s.AmountOutGetterArgs.(AmountOutGetterSwapOutputPackedBytes32Args)
		if !ok {
			return nil, fmt.Errorf("args must be a AmountOutGetterSwapOutputPackedBytes32Args")
		}
		amountOut = new(big.Int).SetBytes(swapResult.Data[16*args.ElementIndex:][:16])
	case AmountOutGetterDelta:
		args, ok := s.AmountOutGetterArgs.(AmountOutGetterDeltaArgs)
		if !ok {
			return nil, fmt.Errorf("args must be a AmountOutGetterDeltaArgs")
		}
		if len(results) < args.BalanceOfBeforeIndex+1 {
			return nil, fmt.Errorf("results must have atleast %d elements", args.BalanceOfBeforeIndex+1)
		}
		if len(results) < args.BalanceOfAfterIndex+1 {
			return nil, fmt.Errorf("results must have atleast %d elements", args.BalanceOfAfterIndex+1)
		}
		balanceOutBefore := new(big.Int).SetBytes(results[args.BalanceOfBeforeIndex].Data)
		balanceOutAfter := new(big.Int).SetBytes(results[args.BalanceOfAfterIndex].Data)
		amountOut = balanceOutAfter.Sub(balanceOutAfter, balanceOutBefore)
	}
	return amountOut, nil
}

// CalcAmountOutAEVM calculates amount out after swapping through pool p by strategy s
func CalcAmountOutAEVM(
	p *AEVMPool,
	s *AEVMSwapStrategy,
	amountIn *big.Int,
	tokenIn, tokenOut gethcommon.Address,
) (*pool.CalcAmountOutResult, error) {
	if s.Precheck != nil {
		if err := s.Precheck(); err != nil {
			return nil, err
		}
	}

	blIn, ok := p.TokenBalanceSlots.Get().(routerentity.TokenBalanceSlots)[tokenIn]
	if !ok {
		return nil, fmt.Errorf("expected token balance slot for token %s", tokenIn)
	}

	wallet := aevmcommon.HexToAddress(blIn.Wallet)

	calls := s.SwapCalls

	var overrides *aevmtypes.StateOverrides
	if p.NextSwapInfo != nil {
		// inherit current state
		overrides = p.NextSwapInfo.StateAfter.Clone()
	} else {
		overrides = new(aevmtypes.StateOverrides)
	}
	// make sure wallet have abundant native tokens
	overrides.OverrideBalance(wallet, new(uint256.Int).SetUint64(math.MaxUint64))
	// if have to use holders
	var numHolderTransfers int
	if len(blIn.Holders) > 0 {
		var sources []gethcommon.Address
		for _, holder := range blIn.Holders {
			addr := gethcommon.HexToAddress(holder)
			// ignore the pool address
			if addr != p.Address {
				sources = append(sources, addr)
			}
			if len(sources) == maxNumberOfHoldersToTransfer {
				break
			}
		}
		if len(sources) == 0 {
			return nil, fmt.Errorf("there is no usable holder from holders list %v", blIn.Holders)
		}
		// exaggeratedAmountIn = 1.5 * amountIn to overcome transfer fee
		exaggeratedAmountIn := new(big.Int).Mul(amountIn, big.NewInt(3))
		exaggeratedAmountIn.Rsh(exaggeratedAmountIn, 1)
		var transferCalls []aevmtypes.SingleCall
		for _, source := range sources {
			transferInput, _ := abis.ERC20.Pack("transfer", gethcommon.HexToAddress(blIn.Wallet), exaggeratedAmountIn)
			transferCalls = append(transferCalls, aevmtypes.SingleCall{
				From:  aevmcommon.Address(source),
				To:    aevmcommon.Address(tokenIn),
				Value: uint256.NewInt(0),
				Data:  transferInput,
			})
		}
		numHolderTransfers = len(transferCalls)
		calls.PreCalls = append(transferCalls, calls.PreCalls...)
	} else {
		// overriding amountIn if BalanceSlot is specified
		if blIn.BalanceSlot != "" {
			var overridedBalance aevmcommon.Hash
			if blIn.PreferredValue != "" {
				// Some token stores balance in specical way that we coud only handle case-by-case.
				overridedBalance = aevmcommon.HexToHash(blIn.PreferredValue)
			} else {
				overridedBalance = uint256.MustFromBig(amountIn).Bytes32()
			}
			// make sure wallet have enough amountIn
			overrides.OverrideState(aevmcommon.Address(tokenIn), aevmcommon.HexToHash(blIn.BalanceSlot), overridedBalance)
		}
		// override extra if needed
		for slot, val := range blIn.ExtraOverrides {
			overrides.OverrideState(aevmcommon.Address(tokenIn), aevmcommon.HexToHash(slot), aevmcommon.HexToHash(val))
		}
	}

	results, err := p.AEVMClient.Get().(aevmclient.Client).MultipleCall(&aevmtypes.MultipleCallParams{
		StateRoot: aevmcommon.Hash(p.StateRoot),
		Calls:     calls.List(),
		Overrides: overrides,
	})
	if err != nil {
		return nil, err
	}
	if len(results.Results) != calls.Len() {
		return nil, fmt.Errorf("an error occurred in call sequence")
	}

	swapResult, err := calls.GetSwapCallResult(results.Results)
	if err != nil {
		return nil, err
	}

	// make sure all calls are success
	for i, result := range results.Results {
		if i >= numHolderTransfers && !result.Success {
			return nil, fmt.Errorf("simulation call %d/%d returns error: %s", i+1, len(results.Results), result.Error)
		}
	}

	amountOut, err := s.getAmountOut(swapResult, results.Results)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  strings.ToLower(tokenOut.String()),
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  strings.ToLower(tokenIn.String()),
			Amount: nil,
		},
		SwapInfo: &AEVMSwapInfo{
			StateAfter: swapResult.StateAfter,
		},
		Gas: int64(swapResult.GasUsed),
	}, nil
}
