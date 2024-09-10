package erc20balanceslot

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/pkg/jsonrpc"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

const (
	maxDoublingIterations = 256
)

var (
	// 2^127
	balanceThreshold = new(big.Int).Exp(big.NewInt(2), big.NewInt(127), nil)
)

type DoubleFromSourceStrategyExtraParams struct {
	Source common.Address
}

func (*DoubleFromSourceStrategyExtraParams) ProbeStrategyExtraParams() {}

// DoubleFromSourceStrategy From a known source whose balance > 0, try to clone its balance to another two wallets,
// then transfer from the two wallets to a third wallet to double the source balance.
// Keep doubling balance until could not double anymore or reach a threshold.
type DoubleFromSourceStrategy struct {
	rpcClient *rpc.Client
	ethClient *ethclient.Client
}

func NewDoubleFromSourceStrategy(rpcClient *rpc.Client) *DoubleFromSourceStrategy {
	return &DoubleFromSourceStrategy{
		rpcClient: rpcClient,
		ethClient: ethclient.NewClient(rpcClient),
	}
}

func (*DoubleFromSourceStrategy) Name(extraParams ProbeStrategyExtraParams) string {
	_extraParams := extraParams.(*DoubleFromSourceStrategyExtraParams)
	return fmt.Sprintf("double_from_source,source=%s", strings.ToLower(_extraParams.Source.String()))
}

func (p *DoubleFromSourceStrategy) ProbeBalanceSlot(ctx context.Context, token common.Address, extraParams ProbeStrategyExtraParams) (*types.ERC20BalanceSlot, error) {
	logger.Infof(ctx, "[%s] probing balance slot for token %s", p.Name(extraParams), token)

	_extraParams, ok := extraParams.(*DoubleFromSourceStrategyExtraParams)
	if !ok || _extraParams == nil {
		return nil, fmt.Errorf("extraParams must be DoubleFromSourceStrategyExtraParams")
	}

	blockNumber, err := p.ethClient.BlockNumber(context.Background())
	if err != nil {
		return nil, fmt.Errorf("could not get latest block number %w", err)
	}
	blockNumberHex := hexutil.EncodeUint64(blockNumber)

	var (
		source       = _extraParams.Source
		srcOverrides map[common.Hash]common.Hash
		balance      = big.NewInt(0)
	)
	// check source balance > 0
	balanceOfSourceCall, _ := abis.ERC20.Pack("balanceOf", source)
	balanceOfSourceResut, err := jsonrpc.EthCall(p.rpcClient, &jsonrpc.EthCallCalldataParam{
		From: source.String(),
		To:   token.String(),
		Gas:  gasLimit,
		Data: hexutil.Encode(balanceOfSourceCall),
	}, blockNumberHex, nil)
	if err != nil {
		return nil, fmt.Errorf("could not balanceOf(source): %w", err)
	}
	sourceBalance := new(big.Int).SetBytes(common.HexToHash(*balanceOfSourceResut).Bytes())
	if sourceBalance.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("source balance must > 0")
	}

	for i := 0; i < maxDoublingIterations; i++ {
		var nextSource *common.Address
		nextBalance, nextSource, nextOverrides, err := p.doubleBalance(ctx, blockNumberHex, token, source, srcOverrides)
		// stop if err while doubling
		if err != nil {
			logger.Warnf(ctx, "could not double balance: %s", err)
			break
		}
		// stop if balance stop doubling
		if nextBalance.Cmp(balance) <= 0 {
			break
		}
		balance = nextBalance
		srcOverrides = nextOverrides
		source = *nextSource
		// stop if balance reaches balanceThreshold
		if balance.Cmp(balanceThreshold) >= 0 {
			break
		}
	}

	if balance.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("could not double balance")
	}

	bl := &types.ERC20BalanceSlot{
		Token:          strings.ToLower(token.String()),
		Wallet:         strings.ToLower(source.String()),
		Found:          true,
		ExtraOverrides: make(map[string]string),
	}
	for slot, value := range srcOverrides {
		bl.ExtraOverrides[slot.String()] = value.String()
	}
	return bl, nil
}

/*
See the diagram by pasting the following code into https://edotor.net.

	digraph {
	    node [shape=box];

	    source [label="known holder"];
	    wallet1 [label="wallet 1"];
	    wallet2 [label="wallet 2"];
	    wallet3 [label="wallet 3"];
	    wallet1_state [label="wallet 1's new slots" shape=note]
	    wallet2_state [label="wallet 2's new slots" shape=note]
	    wallet3_state [label="wallet 3's new slots" shape=note]
	    tx1 [label="transfer(amount=A)" shape=parallelogram];
	    tx2 [label="transfer(amount=A)" shape=parallelogram];
	    tx3 [label="transfer(amount=A)\ntransfer(amount=A)" shape=parallelogram];

	    source -> tx1 -> wallet1
	    wallet1 -> wallet1_state;

	    source -> tx2 -> wallet2;
	    wallet2 -> wallet2_state;

	    wallet1_state -> tx3;
	    wallet1 -> tx3;
	    wallet2_state -> tx3;
		wallet2 -> tx3 -> wallet3;
	    wallet3 -> wallet3_state;
	}
*/
func (p *DoubleFromSourceStrategy) doubleBalance(ctx context.Context, blockNumberHex string, token, source common.Address, srcOverrides map[common.Hash]common.Hash) (
	balance *big.Int, nextSource *common.Address, nextOverrides map[common.Hash]common.Hash, err error) {
	// transfer from source to randomized wallet1
	wallet1 := randomizeAddress()
	overrides1, err := p.doTransfersAndExtractNextStateOverrides(
		blockNumberHex, token, source, wallet1, srcOverrides, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not do transfer and extract new slots %w", err)
	}

	// transfer from source to randomized wallet2
	wallet2 := randomizeAddress()
	overrides2, err := p.doTransfersAndExtractNextStateOverrides(
		blockNumberHex, token, source, wallet2, srcOverrides, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not do transfer and extract new slots %w", err)
	}

	// transfer from wallet1 and wallet2 to randonmized wallet3
	wallet3 := randomizeAddress()
	overrides31, err := p.doTransfersAndExtractNextStateOverrides(
		blockNumberHex, token, wallet1, wallet3, overrides1, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not do transfer and extract new slots %w", err)
	}
	includedSlots := make(map[common.Hash]struct{})
	for slot := range overrides31 {
		includedSlots[slot] = struct{}{}
	}
	overrides32, err := p.doTransfersAndExtractNextStateOverrides(
		blockNumberHex, token, wallet2, wallet3, mergeStateMap(overrides2, overrides31), includedSlots)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not do transfer and extract new slots %w", err)
	}
	// we have to merge these two since there are some slots unchanged
	overrides32 = mergeStateMap(overrides31, overrides32)

	// get the doubled balance
	balanceOfCall, _ := abis.ERC20.Pack("balanceOf", wallet3)
	result, err := jsonrpc.EthCall(p.rpcClient, &jsonrpc.EthCallCalldataParam{
		From: wallet3.String(),
		To:   token.String(),
		Gas:  gasLimit,
		Data: hexutil.Encode(balanceOfCall),
	}, blockNumberHex, jsonrpc.StateOverride{token: jsonrpc.OverrideAccount{StateDiff: overrides32}})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not balanceOf() after doubling: %w", err)
	}
	balance = new(big.Int).SetBytes(common.HexToHash(*result).Bytes())
	logger.Debugf(ctx, "balance after doubling = %s", balance)

	// check if transfer success
	transferCall, _ := abis.ERC20.Pack("transfer", randomizeAddress(), balance)
	result, err = jsonrpc.EthCall(p.rpcClient, &jsonrpc.EthCallCalldataParam{
		From: wallet3.String(),
		To:   token.String(),
		Gas:  gasLimit,
		Data: hexutil.Encode(transferCall),
	}, blockNumberHex, jsonrpc.StateOverride{token: jsonrpc.OverrideAccount{StateDiff: overrides32}})
	if err != nil || common.HexToHash(*result) != common.HexToHash("0x1") {
		return nil, nil, nil, fmt.Errorf("could not transfer() after doubling: %w", err)
	}

	// wallet3 is the next source to double
	nextSource = &wallet3
	nextOverrides = overrides32

	return balance, nextSource, nextOverrides, nil
}

// Do transfer() and extract the newly-populated slots or slots in "includedSlots".
// These slots store only the balance of "to" address.
func (p *DoubleFromSourceStrategy) doTransfersAndExtractNextStateOverrides(
	blockNumber string,
	token common.Address, from, to common.Address,
	overrides map[common.Hash]common.Hash,
	includedSlots map[common.Hash]struct{},
) (map[common.Hash]common.Hash, error) {
	config := prestateTracerConfig{
		DiffMode: true, // set diffMode to true to get the post state
	}
	configEncoded, _ := json.Marshal(config)
	params := &jsonrpc.DebugTraceCallTracerConfigParam{
		Tracer:       "prestateTracer",
		TracerConfig: configEncoded,
	}
	if len(overrides) > 0 {
		params.StateOverrides = jsonrpc.StateOverride{
			token: jsonrpc.OverrideAccount{StateDiff: overrides},
		}
	}

	// first determine source balance
	balanceOfCall, _ := abis.ERC20.Pack("balanceOf", from)
	balanceOfResult, err := jsonrpc.EthCall(p.rpcClient, &jsonrpc.EthCallCalldataParam{
		From: from.String(),
		To:   token.String(),
		Gas:  gasLimit,
		Data: hexutil.Encode(balanceOfCall),
	}, blockNumber, params.StateOverrides)
	if err != nil {
		return nil, fmt.Errorf("could not eth_call balanceOf() %w", err)
	}
	sourceBalance := new(big.Int).SetBytes(common.HexToHash(*balanceOfResult).Bytes())

	call, _ := abis.ERC20.Pack("transfer", to, sourceBalance)
	// make sure transfer is success
	transferResult, err := jsonrpc.EthCall(p.rpcClient, &jsonrpc.EthCallCalldataParam{
		From: from.String(),
		To:   token.String(),
		Gas:  gasLimit,
		Data: hexutil.Encode(call),
	}, blockNumber, params.StateOverrides)
	if err != nil || common.HexToHash(*transferResult) != common.HexToHash("0x1") {
		return nil, fmt.Errorf("could not transfer")
	}

	// then extract new slots after trasferring
	result := new(prestateTracerResult)
	err = jsonrpc.DebugTraceCall(
		p.rpcClient,
		&jsonrpc.DebugTraceCallCalldataParam{
			From: from.String(),
			To:   token.String(),
			Gas:  gasLimit,
			Data: hexutil.Encode(call),
		},
		blockNumber,
		params,
		result,
	)
	if err != nil {
		return nil, fmt.Errorf("could not debug_traceCall transfer() %w", err)
	}

	pre := result.Pre[token].Storage
	post := result.Post[token].Storage
	nextOverrides := make(map[common.Hash]common.Hash)
	for slot, value := range post {
		if _, ok := includedSlots[slot]; ok {
			// slots in "includedSlots"
			nextOverrides[slot] = value
		} else if preValue, ok := pre[slot]; !ok || preValue == (common.Hash{}) {
			// newly-populated slots
			nextOverrides[slot] = value
		}
	}
	return nextOverrides, nil
}

// merge two state map a and b, preferred b over a
func mergeStateMap(a, b map[common.Hash]common.Hash) map[common.Hash]common.Hash {
	m := make(map[common.Hash]common.Hash)
	for slot, value := range a {
		m[slot] = value
	}
	for slot, value := range b {
		m[slot] = value
	}
	return m
}
