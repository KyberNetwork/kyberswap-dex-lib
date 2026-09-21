package everlongflamm

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
)

// Reads go through Multicall3.tryBlockAndAggregate(false, calls) as one eth_call, decoded here rather than by
// ethrpc's Request: the tracker needs each call's revert data (a probe must revert with the port's error, not
// just revert) and must fail a word that does not decode instead of leaving it zero. The block the aggregate ran
// at comes back with it; every later read of the same refresh is pinned to that block *number*.
//
// The aggregate's second output, the block hash, is not a pin and is ignored: Multicall3 returns
// blockhash(block.number), which the EVM defines as zero for the block being executed, so an eth_call at any block
// -- latest or historical, on Base and on anvil -- answers 0x00..00 there. Pinning a refresh's rounds to one block
// hash would need EIP-1898 block parameters on every later eth_call and eth_getStorageAt, which not every node
// serves. What a same-height reorg between two rounds of one refresh would therefore have to survive is the
// attestation: the probe aggregate runs after the storage reads and compares the deployed previews against a state
// built from the view round, so a block that differs in any word those previews read fails the refresh closed. Only
// a replacement that moves no probed answer could pass, and it is the same class as the "not bound by attestation"
// bucket the fork tamper test measures.

// multicall3 is Multicall3's canonical address, deployed on every supported chain.
var multicall3 = common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")

// callOpts are the eth_call fields a round may need beyond the block: an explicit gas limit, and the block
// timestamp the EVM sees. The gas limit is sent only by the probe aggregate, whose cost is state-dependent and
// large (attest.go probeAggregateGas). The timestamp override is sent by the forward rounds, which read at a later
// clock inside the snapshot window what a time-dependent oracle will answer (tracker_reads.go readOracleAhead) and
// what the views a deadline word decides will answer (readClock); they need an eth_call that accepts
// `blockOverrides`, which every node the refresh runs against must support -- and which every such round proves the
// node applied rather than assumes (aggregate, clockProof).
type callOpts struct {
	gas  uint64
	time uint64
}

// A read either fails in transport (the node, the batch or the aggregate envelope did not answer usably: errRPC)
// or is answered by the chain at the refresh block and does not fit the registry -- a required view reverted
// (errReadFailed) or returned data that does not decode (errReadDecode). Only the second kind is the pool's own
// answer: the tracker publishes it as a refusing refresh and the lister skips the pool; the first is returned.
var (
	errRPC        = errors.New("everlong-flamm: rpc answer unusable")
	errReadFailed = errors.New("everlong-flamm: read failed")
	errReadDecode = errors.New("everlong-flamm: read did not decode")
)

// chainAnswered reports whether err is the chain's own answer at the refresh block (see errRPC).
func chainAnswered(err error) bool {
	return !errors.Is(err, errRPC) && (errors.Is(err, errReadFailed) || errors.Is(err, errReadDecode))
}

// mcCall is one call of an aggregate; Out, when set, receives the decoded outputs (Required calls only).
type mcCall struct {
	Name     string
	Target   common.Address
	ABI      *abi.ABI
	Method   string
	Args     []any
	Optional bool // a revert is an answer (e.g. an aggregator that reverts), not a read failure
}

type mcResult struct {
	Ok   bool
	Data []byte
}

// mcRPC is the RPC surface a refresh uses: eth_call (optionally with state overrides) and batched JSON-RPC.
type mcRPC struct {
	client    *ethrpc.Client
	overrides map[common.Address]gethclient.OverrideAccount
}

func (r *mcRPC) call(ctx context.Context, msg ethereum.CallMsg, block *big.Int, opts callOpts) ([]byte, error) {
	msg.Gas = opts.gas
	var state *map[common.Address]gethclient.OverrideAccount
	if len(r.overrides) != 0 {
		state = &r.overrides
	}
	if opts.time != 0 {
		return gethclient.New(r.client.GetETHClient().Client()).CallContractWithBlockOverrides(ctx, msg, block,
			state, gethclient.BlockOverrides{Time: opts.time})
	}
	if state != nil {
		return gethclient.New(r.client.GetETHClient().Client()).CallContract(ctx, msg, block, state)
	}
	return r.client.GetETHClient().CallContract(ctx, msg, block)
}

// reverted reports whether the node answered an eth_call with a revert rather than with an answer or with a
// failure of its own. A revert is a JSON-RPC error of code 3 (geth's revertError), which carries the revert data
// where there is any; anything else -- an out-of-gas, a transport failure -- is not the contract's answer.
func reverted(err error) bool {
	if err == nil {
		return false
	}
	var data rpc.DataError
	if errors.As(err, &data) && data.ErrorData() != nil {
		return true
	}
	var coded rpc.Error
	return errors.As(err, &coded) && coded.ErrorCode() == 3
}

// execErrorCodes are the JSON-RPC codes a node answers an eth_call it ran and could not finish with: geth's generic
// server error and the code Base answers an out-of-gas with (a revert is code 3 or carries revert data, reverted).
// Base does not use -32000 for it: mainnet.base.org answers USDC.balanceOf sent with 30,000 gas -- above that
// call's intrinsic cost, so the EVM ran it -- {"code":-32003,"message":"out of gas: gas required exceeds: 30000"}.
// The class is matched rather than one code of it.
var execErrorCodes = [...]int{-32000, -32003}

// executed reports whether err is the node's report of having run a call and not finished it, rather than a failure
// of the request itself. A provider's limit, an internal error, a context that expired and a connection that
// dropped all carry another code or no JSON-RPC error at all (mainnet.base.org answers its own rate limit with
// HTTP 429 and code -32016).
func executed(err error) bool {
	var coded rpc.Error
	if !errors.As(err, &coded) {
		return false
	}
	for _, code := range execErrorCodes {
		if coded.ErrorCode() == code {
			return true
		}
	}
	return false
}

// emptyRevert reports whether err is a revert that carries no data: the answer the port reads as OpenZeppelin
// mulDiv's own empty revert. A revert that carries data is a different answer than the aggregate's empty one, and
// an EVM that ran out of gas inside the call frame is reported as an empty revert too -- on Base,
// previewSwap(true,100000) sent with 200,000 gas answers {"code":3,"message":"execution reverted"} with no data and
// with 500,000 gas answers the quote -- which is why the confirmation is sent more gas than the subcall had
// (confirmRevert).
func emptyRevert(err error) bool {
	if !reverted(err) {
		return false
	}
	var carrier rpc.DataError
	if !errors.As(err, &carrier) {
		return true
	}
	switch data := carrier.ErrorData().(type) {
	case nil:
		return true
	case string:
		return len(strings.TrimPrefix(data, "0x")) == 0
	default:
		return false // an answer the port cannot read is not a confirmation
	}
}

// confirmRevert re-runs one call of an aggregate on its own, with an explicit gas limit, and reports whether the
// node answers it with the same empty revert. Multicall3 reports a subcall that ran out of gas exactly as it
// reports a revert with no data -- empty returndata on an unsuccessful call -- which revertError maps to
// OpenZeppelin mulDiv's own empty revert, so an aggregate that outgrew the node's gas cap could otherwise be read
// as a probe the port reproduced. The call of its own is sent the whole aggregate's limit for one call, which is
// strictly more than Multicall3 could have forwarded it inside the aggregate (63/64 of what was left there), so a
// subcall starved by the aggregate is not starved here.
//
// Its outcomes are the two kinds a read has (errRPC). An empty revert confirms the aggregate's empty answer. An
// answer, a revert carrying data -- which the aggregate's empty returndata is not -- or the node reporting that it
// ran the call and could not finish it at that gas, is the chain's own answer that the port's refusal is wrong
// (errReadFailed), which the refresh publishes. A request the node did not run -- a provider limit, an internal
// error, an expired context, a dropped connection -- is transport (errRPC) and is returned, exactly as the probe
// aggregate's own failure is, so pool-service keeps the last entity instead of dropping the pool for the cycle.
//
// The one answer it cannot tell apart is a node whose own eth_call ceiling is below what a single probe costs: the
// confirmation is then starved as well and answers the same empty revert. That is the gas cap the refresh states
// as an operator requirement (attest.go probeAggregateGas, README "Known limitations").
func (r *mcRPC) confirmRevert(ctx context.Context, block *big.Int, c *mcCall, gas uint64) error {
	data, err := c.ABI.Pack(c.Method, c.Args...)
	if err != nil {
		return fmt.Errorf("%w: pack %s: %v", errRPC, c.Name, err)
	}
	target := c.Target
	_, err = r.call(ctx, ethereum.CallMsg{To: &target, Data: data}, block, callOpts{gas: gas})
	switch {
	case emptyRevert(err):
		return nil
	case err == nil:
		return fmt.Errorf("%w: %s answered on its own", errReadFailed, c.Name)
	case reverted(err):
		return fmt.Errorf("%w: %s reverted with data on its own: %v", errReadFailed, c.Name, err)
	case executed(err):
		return fmt.Errorf("%w: %s did not revert on its own: %v", errReadFailed, c.Name, err)
	}
	return fmt.Errorf("%w: %s unanswered on its own: %v", errRPC, c.Name, err)
}

// clockProof is what a round sent at an overridden clock reads its own block.timestamp back with: Multicall3 is
// the aggregate's own target, so the proof costs no round trip and no extra call frame of consequence. Without it a
// node that accepts `blockOverrides` and ignores it -- as opposed to one that rejects it, which fails the eth_call
// -- would answer every forward round at the snapshot's own clock, which is a silent no-op: readOracleAhead would
// publish the snapshot's own oracle answers, the simulator would find nothing moved (pool_simulator.go
// oracleShifted), and the refresh would attest while the window guard it advertises could never fire.
func clockProof() mcCall {
	return mcCall{Name: "multicall3.getCurrentBlockTimestamp", Target: multicall3, ABI: &multicallABI,
		Method: "getCurrentBlockTimestamp", Optional: true}
}

// checkClock is the clock the node ran the aggregate at against the one the round asked for. A node that did not
// apply the override is a property of the node and not of the pool, so it fails the refresh in transport (errRPC)
// and leaves pool-service with the last entity, rather than publishing a pool as refusing.
func checkClock(res mcResult, want uint64) error {
	if !res.Ok || len(res.Data) != 32 {
		return fmt.Errorf("%w: multicall clock: no answer", errRPC)
	}
	var got uint256.Int
	if got.SetBytes(res.Data); !got.IsUint64() || got.Uint64() != want {
		return fmt.Errorf("%w: eth_call ran at block timestamp %s, not the %d it was sent with: the node does not "+
			"apply blockOverrides", errRPC, got.Dec(), want)
	}
	return nil
}

// aggregate runs calls in one tryBlockAndAggregate at block (nil: latest), with the eth_call options of opts, and
// returns the block it ran at, also alongside a required call's revert, which is that block's answer. A round sent
// at an overridden clock carries clockProof and fails unless the node answers at that clock.
func (r *mcRPC) aggregate(ctx context.Context, block *big.Int, opts callOpts,
	calls []mcCall) (uint64, []mcResult, error) {
	want := len(calls)
	if opts.time != 0 {
		calls = append(append(make([]mcCall, 0, want+1), calls...), clockProof())
	}
	type mcIn struct {
		Target   common.Address
		CallData []byte
	}
	in := make([]mcIn, len(calls))
	for i, c := range calls {
		data, err := c.ABI.Pack(c.Method, c.Args...)
		if err != nil {
			return 0, nil, fmt.Errorf("%w: pack %s: %v", errRPC, c.Name, err)
		}
		in[i] = mcIn{Target: c.Target, CallData: data}
	}
	data, err := multicallABI.Pack("tryBlockAndAggregate", false, in)
	if err != nil {
		return 0, nil, err
	}
	raw, err := r.call(ctx, ethereum.CallMsg{To: &multicall3, Data: data}, block, opts)
	if err != nil {
		return 0, nil, err
	}
	vals, err := multicallABI.Methods["tryBlockAndAggregate"].Outputs.Unpack(raw)
	if err != nil || len(vals) != 3 {
		return 0, nil, fmt.Errorf("%w: multicall: %v", errRPC, err)
	}
	blockNumber, ok := vals[0].(*big.Int)
	if !ok || !blockNumber.IsUint64() {
		return 0, nil, fmt.Errorf("%w: multicall block", errRPC)
	}
	if block != nil && blockNumber.Cmp(block) != 0 {
		return 0, nil, fmt.Errorf("%w: multicall ran at block %s, not %s", errRPC, blockNumber, block)
	}
	type mcOut struct {
		Success    bool
		ReturnData []byte
	}
	results, ok := abi.ConvertType(vals[2], new([]mcOut)).(*[]mcOut)
	if !ok || len(*results) != len(calls) {
		return 0, nil, fmt.Errorf("%w: multicall results", errRPC)
	}
	out := make([]mcResult, len(calls))
	for i, res := range *results {
		out[i] = mcResult{Ok: res.Success, Data: res.ReturnData}
		if !res.Success && !calls[i].Optional {
			return blockNumber.Uint64(), nil, fmt.Errorf("%w: %s reverted %s", errReadFailed, calls[i].Name,
				hexutil.Encode(res.ReturnData))
		}
	}
	if opts.time != 0 {
		if err := checkClock(out[want], opts.time); err != nil {
			return blockNumber.Uint64(), nil, err
		}
		out = out[:want]
	}
	return blockNumber.Uint64(), out, nil
}

// rpcBatchSize caps the calls of one JSON-RPC batch. Providers cap batches (mainnet.base.org answers a batch of
// more than ten calls with a single error object, -32014 "maximum 10 calls in 1 batch"), so every batch is sent in
// chunks of at most ten.
const rpcBatchSize = 10

// batchCall sends batch in consecutive chunks of at most rpcBatchSize calls. A chunk that fails in transport fails
// the call; per-call errors stay on their elements.
func batchCall(ctx context.Context, client *ethrpc.Client, batch []rpc.BatchElem) error {
	rc := client.GetETHClient().Client()
	for i := 0; i < len(batch); i += rpcBatchSize {
		if err := rc.BatchCallContext(ctx, batch[i:min(i+rpcBatchSize, len(batch))]); err != nil {
			return err
		}
	}
	return nil
}

// storageAt reads raw storage words at block in JSON-RPC batches (batchCall), applying the state overrides the way an
// eth_call would see them (a full `state` replaces the account's storage, a `stateDiff` patches it).
func (r *mcRPC) storageAt(ctx context.Context, block *big.Int, reads []storageRead) ([]common.Hash, error) {
	out := make([]common.Hash, len(reads))
	batch := make([]rpc.BatchElem, 0, len(reads))
	idx := make([]int, 0, len(reads))
	for i, rd := range reads {
		if o, ok := r.overrides[rd.Account]; ok {
			if o.State != nil {
				out[i] = o.State[rd.Slot]
				continue
			}
			if v, ok := o.StateDiff[rd.Slot]; ok {
				out[i] = v
				continue
			}
		}
		batch = append(batch, rpc.BatchElem{Method: "eth_getStorageAt",
			Args: []any{rd.Account, rd.Slot, hexutil.EncodeBig(block)}, Result: &out[i]})
		idx = append(idx, i)
	}
	if len(batch) == 0 {
		return out, nil
	}
	if err := batchCall(ctx, r.client, batch); err != nil {
		return nil, err
	}
	for i := range batch {
		if batch[i].Error != nil {
			return nil, fmt.Errorf("%w: storage %s[%s]: %v", errRPC, reads[idx[i]].Account, reads[idx[i]].Slot,
				batch[i].Error)
		}
	}
	return out, nil
}

type storageRead struct {
	Account common.Address
	Slot    common.Hash
}

// unpack decodes a successful call's outputs.
func unpack(c *mcCall, res mcResult) ([]any, error) {
	if !res.Ok {
		return nil, fmt.Errorf("%w: %s reverted", errReadFailed, c.Name)
	}
	vals, err := c.ABI.Methods[c.Method].Outputs.Unpack(res.Data)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", errReadDecode, c.Name, err)
	}
	return vals, nil
}

// wordOf converts a decoded unsigned ABI value into a word; a nil or negative big.Int does not decode.
func wordOf(v any) (uint256.Int, error) {
	var z uint256.Int
	switch x := v.(type) {
	case *big.Int:
		if x == nil || x.Sign() < 0 || z.SetFromBig(x) {
			return z, errReadDecode
		}
	case uint64:
		z.SetUint64(x)
	case uint32:
		z.SetUint64(uint64(x))
	case uint16:
		z.SetUint64(uint64(x))
	case uint8:
		z.SetUint64(uint64(x))
	default:
		return z, errReadDecode
	}
	return z, nil
}

// signedOf converts a decoded int256 into its two's-complement word.
func signedOf(v any) (int256.Int, error) {
	var z int256.Int
	x, ok := v.(*big.Int)
	if !ok || x == nil || z.SetFromBig(x) {
		return z, errReadDecode
	}
	return z, nil
}

// wordsOf converts a decoded []*big.Int.
func wordsOf(v any) ([]uint256.Int, error) {
	xs, ok := v.([]*big.Int)
	if !ok {
		return nil, errReadDecode
	}
	out := make([]uint256.Int, len(xs))
	for i := range xs {
		w, err := wordOf(xs[i])
		if err != nil {
			return nil, err
		}
		out[i] = w
	}
	return out, nil
}
