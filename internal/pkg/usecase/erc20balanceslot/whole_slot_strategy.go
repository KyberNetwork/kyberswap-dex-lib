package erc20balanceslot

import (
	"context"
	"errors"
	"math/rand"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/pkg/jsonrpc"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

func randomizeHash() common.Hash {
	h := common.Hash{}
	for i := range h {
		h[i] = byte(rand.Intn(256))
	}
	return h
}

const (
	gasLimit = "0x7a120"
)

// WholeSlotStrategy For a ERC20 token and a wallet, find the storage slot of the token that contains the wallet's balance of the token.
// This strategy only works if the ERC20 token's contract reads and writes balances directly from and to a mapping (e.g `mapping(address => uint256) _balances`).
type WholeSlotStrategy struct {
	rpcClient *rpc.Client
	wallet    common.Address
}

func NewWholeSlotStrategy(rpcClient *rpc.Client, wallet common.Address) *WholeSlotStrategy {
	return &WholeSlotStrategy{
		rpcClient: rpcClient,
		wallet:    wallet,
	}
}

func (*WholeSlotStrategy) Name(_ ProbeStrategyExtraParams) string {
	return "whole_slot"
}

// ProbeBalanceSlot For a ERC20 token and a wallet, find the storage slot of the token that contains the wallet's balance of the token.
// This approach only works if the ERC20 token's contract reads and writes balances directly from and to a mapping.
func (p *WholeSlotStrategy) ProbeBalanceSlot(ctx context.Context, token common.Address, _ ProbeStrategyExtraParams) (*types.ERC20BalanceSlot, error) {
	logger.Infof(ctx, "probing balance slot for wallet %s in token %s\n", p.wallet, token)

	/*
		Step 1: Trace all SLOAD instructions after calling balanceOf(wallet)
	*/
	data, err := abis.ERC20.Pack("balanceOf", p.wallet)
	if err != nil {
		return nil, err
	}
	tracingResult := new(tracingResult)
	err = jsonrpc.DebugTraceCall(
		p.rpcClient,
		&jsonrpc.DebugTraceCallCalldataParam{
			From: common.Address{}.String(),
			To:   token.String(),
			Gas:  gasLimit,
			Data: hexutil.Encode(data),
		},
		"latest",
		&jsonrpc.DebugTraceCallTracerConfigParam{
			Tracer: string(sloadTracerMinified),
		},
		tracingResult,
	)
	if err != nil {
		return nil, err
	}

	// encoded, _ := json.MarshalIndent(tracingResult, "", "  ")
	// fmt.Printf("tracing result = %s\n", string(encoded))

	/*
		Step 2:
			For each SLOAD instruction, if its value is the same as output, its slot might be the slot we are finding.
			There might be many of them so we need to check each of them for sure.
			For each SLOAD instruction whose value is the same as output, override its slot with a randomized value v then call balanceOf(wallet) again.
			If the output of balanceOf(wallet) is the same as v, the slot is the slot we are finding with high possibility.
			If there is only 1 instruction whose output of balanceOf(wallet) is the same as v, its slot is the slot we are finding.
			Otherwise, we could not find the slot we are finding.
	*/
	var possibleSlots []common.Hash
	for _, sload := range tracingResult.Sloads {
		if common.HexToHash(sload.Value) != common.HexToHash(tracingResult.Output) {
			continue
		}

		testValue := randomizeHash()
		logger.Debugf(ctx, "    probing slot %s with test value %s\n", common.HexToHash(sload.Slot), testValue)
		result, err := jsonrpc.EthCall(
			p.rpcClient,
			&jsonrpc.EthCallCalldataParam{
				From: common.Address{}.String(),
				To:   token.String(),
				Gas:  gasLimit,
				Data: hexutil.Encode(data),
			},
			"latest",
			map[common.Address]jsonrpc.OverrideAccount{
				token: {
					StateDiff: map[common.Hash]common.Hash{
						common.HexToHash(sload.Slot): testValue,
					},
				},
			},
		)
		if err != nil {
			return nil, err
		}
		logger.Debugf(ctx, "    result = %+v\n", *result)
		if common.HexToHash(*result) == testValue {
			logger.Debugf(ctx, "        slot %s is a candidate\n", common.HexToHash(sload.Slot))
			possibleSlots = append(possibleSlots, common.HexToHash(sload.Slot))
		}
	}

	if len(possibleSlots) != 1 {
		logger.Debugf(ctx, "    EXPECTED 1 CANDIDATE, GOT %v\n", len(possibleSlots))
		return nil, errors.New("could not probe")
	}

	return &types.ERC20BalanceSlot{
		Token:       strings.ToLower(token.String()),
		Wallet:      strings.ToLower(p.wallet.String()),
		Found:       true,
		BalanceSlot: possibleSlots[0].String(),
	}, nil
}
