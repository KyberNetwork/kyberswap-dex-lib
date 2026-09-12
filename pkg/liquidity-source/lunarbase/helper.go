package lunarbase

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type rpcState struct {
	blockNumber uint64
	blockHash   common.Hash
	hasNative   bool
	tokenX      string
	tokenY      string
	reserveX    *big.Int
	reserveY    *big.Int
	extra       Extra
}

var snapshotAggregateABI = func() abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(`[{"name":"tryBlockAndAggregate","type":"function","inputs":[],"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"blockHash","type":"bytes32"},{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}]`))
	if err != nil {
		panic(err)
	}
	return parsed
}()

func fetchRPCState(ctx context.Context, coreAddress string, chainID valueobject.ChainID, ethrpcClient *ethrpc.Client,
	overrides map[common.Address]gethclient.OverrideAccount) (*rpcState, error) {
	if ethrpcClient == nil || !common.IsHexAddress(coreAddress) || common.HexToAddress(coreAddress) == (common.Address{}) {
		return nil, fmt.Errorf("lunarbase snapshot: invalid client or pool address")
	}
	header, err := latestSnapshotHeader(ctx, ethrpcClient)
	if err != nil {
		return nil, err
	}
	ref := snapshotReference{header.Number.Uint64(), header.Hash()}
	return fetchRPCStateAt(ctx, coreAddress, chainID, ethrpcClient, overrides, ref)
}

func latestSnapshotHeader(ctx context.Context, client *ethrpc.Client) (*types.Header, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	header, err := client.GetETHClient().HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}
	if header == nil || header.Number == nil || !header.Number.IsUint64() || header.Number.Sign() <= 0 {
		return nil, fmt.Errorf("lunarbase snapshot: invalid header")
	}
	return header, nil
}

func verifySnapshotHeader(ctx context.Context, client *ethrpc.Client, ref snapshotReference) error {
	header, err := client.GetETHClient().HeaderByNumber(ctx, new(big.Int).SetUint64(ref.number))
	if err != nil {
		return err
	}
	if header == nil || header.Number == nil || !header.Number.IsUint64() || header.Number.Uint64() != ref.number || header.Hash() != ref.hash {
		return fmt.Errorf("lunarbase snapshot: selected block is no longer canonical")
	}
	return ctx.Err()
}

// fetchRPCStateAt reads every getter at one hash and verifies its canonical
// membership after the aggregate has finished.
func fetchRPCStateAt(ctx context.Context, coreAddress string, chainID valueobject.ChainID, ethrpcClient *ethrpc.Client,
	overrides map[common.Address]gethclient.OverrideAccount, ref snapshotReference) (*rpcState, error) {
	return fetchSelectedRPCState(ctx, coreAddress, chainID, ethrpcClient, overrides, &ref)
}

// All getters execute within one EVM call. The returned number identifies its
// height, but no block hash or post-call canonicality guarantee is available.
func fetchLatestRPCState(ctx context.Context, coreAddress string, chainID valueobject.ChainID,
	client *ethrpc.Client) (*rpcState, error) {
	if client == nil || !common.IsHexAddress(coreAddress) || common.HexToAddress(coreAddress) == (common.Address{}) {
		return nil, fmt.Errorf("lunarbase snapshot: invalid client or pool address")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return fetchSelectedRPCState(ctx, coreAddress, chainID, client, nil, nil)
}

func fetchSelectedRPCState(ctx context.Context, coreAddress string, chainID valueobject.ChainID,
	ethrpcClient *ethrpc.Client, overrides map[common.Address]gethclient.OverrideAccount,
	ref *snapshotReference) (*rpcState, error) {
	var blockNumber uint64
	var blockHash common.Hash
	if ref != nil {
		blockNumber, blockHash = ref.number, ref.hash
	}
	var (
		tokenX           common.Address
		tokenY           common.Address
		paused           bool
		blockDelay       uint64
		concentrationK   uint32
		maxPunishmentX24 uint32
		anchorPrice      *big.Int
		reserveX         *big.Int
		reserveY         *big.Int
		state            struct {
			AnchorPrice       *big.Int
			FeeAskX24         uint32
			FeeBidX24         uint32
			LatestUpdateBlock uint64
		}
	)

	req := ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "X",
	}, []any{&tokenX}).AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "Y",
	}, []any{&tokenY}).AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "blockDelay",
	}, []any{&blockDelay}).AddCall(&ethrpc.Call{
		// concentrationK and maxPunishmentX24 are mutually exclusive across
		// contract versions: pools on the old concentration-curve model only
		// have the former; pools upgraded to the punishment model (SDK
		// v0.4.0) only have the latter. Exactly one is expected to revert, so
		// these two calls alone are allowed to fail within the batch below.
		ABI:    coreABI,
		Target: coreAddress,
		Method: "concentrationK",
	}, []any{&concentrationK}).AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "maxPunishmentX24",
	}, []any{&maxPunishmentX24}).AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "getXReserve",
	}, []any{&reserveX}).AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "getYReserve",
	}, []any{&reserveY}).AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "paused",
	}, []any{&paused}).AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "state",
	}, []any{&state}).AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "anchorPrice",
	}, []any{&anchorPrice})

	const (
		idxConcentrationK   = 3
		idxMaxPunishmentX24 = 4
	)
	// The aggregate executes all getter calls against one selected state.
	// ethrpc does not support blockHash with non-empty state overrides, so
	// that path pins the number and verifies its canonical hash afterwards.
	if ref != nil {
		if len(overrides) > 0 {
			req.SetBlockNumber(new(big.Int).SetUint64(blockNumber))
		} else {
			req.SetBlockHash(blockHash)
		}
	}
	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("lunarbase snapshot: missing aggregate response")
	}
	// With RequireSuccess=false, ethrpc may swallow an intermediate getter
	// decode failure while leaving its Result entry true. Independently
	// validate the raw aggregate and every successful getter payload.
	var aggregate ethrpc.TryBlockAndAggregateResult
	if err := snapshotAggregateABI.UnpackIntoInterface(&aggregate, "tryBlockAndAggregate", resp.RawResponse); err != nil {
		return nil, fmt.Errorf("lunarbase snapshot aggregate decode: %w", err)
	}
	if aggregate.BlockNumber == nil || !aggregate.BlockNumber.IsUint64() || aggregate.BlockNumber.Sign() <= 0 ||
		(ref != nil && aggregate.BlockNumber.Uint64() != blockNumber) || len(aggregate.ReturnData) != len(req.Calls) {
		return nil, fmt.Errorf("lunarbase snapshot: aggregate block or result count mismatch")
	}
	blockNumber = aggregate.BlockNumber.Uint64()
	canonicalAggregate, err := snapshotAggregateABI.Methods["tryBlockAndAggregate"].Outputs.Pack(aggregate.BlockNumber, aggregate.BlockHash, aggregate.ReturnData)
	if err != nil || !bytes.Equal(canonicalAggregate, resp.RawResponse) {
		return nil, fmt.Errorf("lunarbase snapshot: noncanonical aggregate encoding")
	}
	for i, item := range aggregate.ReturnData {
		if !item.Success {
			if i != idxConcentrationK && i != idxMaxPunishmentX24 {
				return nil, fmt.Errorf("lunarbase snapshot: required getter %s reverted", req.Calls[i].Method)
			}
			continue
		}
		call := req.Calls[i]
		method := call.ABI.Methods[call.Method]
		values, err := method.Outputs.Unpack(item.ReturnData)
		if err != nil || len(values) != len(method.Outputs) {
			return nil, fmt.Errorf("lunarbase snapshot: getter %s has malformed output", call.Method)
		}
		canonical, err := method.Outputs.Pack(values...)
		if err != nil || !bytes.Equal(canonical, item.ReturnData) {
			return nil, fmt.Errorf("lunarbase snapshot: getter %s has noncanonical output", call.Method)
		}
		if err := call.ABI.UnpackIntoInterface(call.Output[0], call.Method, item.ReturnData); err != nil {
			return nil, fmt.Errorf("lunarbase snapshot getter %s: %w", call.Method, err)
		}
	}
	if aggregate.ReturnData[idxConcentrationK].Success == aggregate.ReturnData[idxMaxPunishmentX24].Success {
		return nil, fmt.Errorf("lunarbase snapshot: exactly one pricing model must be supported")
	}
	if !aggregate.ReturnData[idxConcentrationK].Success {
		concentrationK = 0
	}
	if !aggregate.ReturnData[idxMaxPunishmentX24].Success {
		maxPunishmentX24 = 0
	}
	if tokenX == tokenY || blockDelay == 0 || blockDelay >= 1<<48 ||
		state.AnchorPrice == nil || state.AnchorPrice.Sign() < 0 || state.AnchorPrice.BitLen() > 160 ||
		anchorPrice == nil || state.AnchorPrice.Cmp(anchorPrice) != 0 ||
		state.FeeAskX24 >= 1<<24 || state.FeeBidX24 >= 1<<24 || maxPunishmentX24 >= 1<<24 ||
		state.LatestUpdateBlock > blockNumber ||
		reserveX == nil || reserveX.Sign() < 0 || reserveX.BitLen() > 112 ||
		reserveY == nil || reserveY.Sign() < 0 || reserveY.BitLen() > 112 {
		return nil, fmt.Errorf("lunarbase snapshot: invalid or inconsistent getter values")
	}
	var hashText string
	if ref != nil {
		if err := verifySnapshotHeader(ctx, ethrpcClient, *ref); err != nil {
			return nil, err
		}
		hashText = blockHash.Hex()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	tokenXAddress := valueobject.WrapNativeZeroLower(hexutil.Encode(tokenX[:]), chainID)
	tokenYAddress := valueobject.WrapNativeZeroLower(hexutil.Encode(tokenY[:]), chainID)
	if tokenXAddress == tokenYAddress {
		return nil, fmt.Errorf("lunarbase snapshot: tokens alias after native wrapping")
	}

	sqrtPriceX96 := uint256.MustFromBig(state.AnchorPrice)

	return &rpcState{
		blockNumber: blockNumber,
		blockHash:   blockHash,
		hasNative:   valueobject.IsNativeOrZeroAddr(tokenX) || valueobject.IsNativeOrZeroAddr(tokenY),
		tokenX:      tokenXAddress,
		tokenY:      tokenYAddress,
		reserveX:    reserveX,
		reserveY:    reserveY,
		extra: Extra{
			ConcentrationModel: aggregate.ReturnData[idxConcentrationK].Success,
			BlockHash:          hashText,
			SnapshotComplete:   ref == nil,
			SqrtPriceX96:       sqrtPriceX96,
			FeeAskX24:          state.FeeAskX24,
			FeeBidX24:          state.FeeBidX24,
			LatestUpdateBlock:  state.LatestUpdateBlock,
			Paused:             paused,
			BlockDelay:         blockDelay,
			ConcentrationK:     concentrationK,
			MaxPunishmentX24:   maxPunishmentX24,
		},
	}, nil
}

func buildEntityPool(p *entity.Pool, state *rpcState) (*entity.Pool, error) {
	if p == nil || state == nil || len(p.Tokens) != 2 || p.Tokens[0] == nil || p.Tokens[1] == nil ||
		!strings.EqualFold(p.Tokens[0].Address, state.tokenX) || !strings.EqualFold(p.Tokens[1].Address, state.tokenY) {
		return nil, fmt.Errorf("lunarbase snapshot: token identity or order differs from entity")
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil || staticExtra.HasNative != state.hasNative {
		return nil, fmt.Errorf("lunarbase snapshot: native token metadata differs from entity")
	}
	extraBytes, err := json.Marshal(state.extra)
	if err != nil {
		return nil, err
	}

	// Report the larger of the two directional fees as the entity's nominal
	// SwapFee — routers use it for coarse cost estimation; the simulator
	// applies the precise per-direction value at quote time.
	maxFee := state.extra.FeeAskX24
	if state.extra.FeeBidX24 > maxFee {
		maxFee = state.extra.FeeBidX24
	}
	p.SwapFee = float64(maxFee) / fQ24
	p.BlockNumber = state.blockNumber
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{state.reserveX.String(), state.reserveY.String()}
	p.Extra = string(extraBytes)
	return p, nil
}
