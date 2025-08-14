package bunniv2

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/hooklet"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

func (h *Hook) GetReserves(ctx context.Context, param *uniswapv4.HookParam) (entity.PoolReserves, error) {
	req := param.RpcClient.NewRequest().SetContext(ctx)
	if param.BlockNumber != nil {
		req.SetBlockNumber(param.BlockNumber)
	}

	var poolState PoolStateRPC
	req.AddCall(&ethrpc.Call{
		ABI:    bunniHubABI,
		Target: GetHubAddress(h.hook),
		Method: "poolState",
		Params: []any{common.HexToHash(param.Pool.Address)},
	}, []any{&poolState})

	if _, err := req.Call(); err != nil {
		return nil, err
	}

	return entity.PoolReserves{
		poolState.Data.Reserve0.Add(poolState.Data.Reserve0, poolState.Data.RawBalance0).String(),
		poolState.Data.Reserve1.Add(poolState.Data.Reserve1, poolState.Data.RawBalance1).String(),
	}, nil
}

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (string, error) {
	var hookExtra HookExtra
	if param.HookExtra != "" {
		if err := json.Unmarshal([]byte(param.HookExtra), &hookExtra); err != nil {
			return "", err
		}
	}

	poolId := common.HexToHash(param.Pool.Address)

	var (
		ldfState     [32]byte
		slot0        Slot0RPC
		poolState    PoolStateRPC
		storageSlots [5]common.Hash
		topBid       BidRPC

		poolManagerBalance0, poolManagerBalance1 = big.NewInt(0), big.NewInt(0)
	)

	slotObservationState := crypto.Keccak256Hash(poolId[:], OBSERVATION_STATE_SLOT)
	slotObservationBase := common.BigToHash(new(big.Int).Add(slotObservationState.Big(), bignumber.One))

	slotVaultSharePrices := crypto.Keccak256Hash(poolId[:], VAULT_SHARE_PRICES_SLOT)
	slotCuratorFees := crypto.Keccak256Hash(poolId[:], CURATOR_FEES_SLOT)
	slotHookFee := crypto.Keccak256Hash(poolId[:], HOOK_FEE_SLOT)

	hubAddress := GetHubAddress(h.hook)
	hookAddress := h.hook.Hex()
	token0Address := param.Pool.Tokens[0].Address
	token1Address := param.Pool.Tokens[1].Address
	poolManagerAddress := GetPoolManagerAddress(valueobject.ChainID(param.Cfg.ChainID))

	req1 := param.RpcClient.NewRequest().SetContext(ctx)
	if param.BlockNumber != nil {
		req1.SetBlockNumber(param.BlockNumber)
	}

	req1.AddCall(&ethrpc.Call{
		ABI:    bunniHookABI,
		Target: hookAddress,
		Method: "extsload",
		Params: []any{
			[]common.Hash{
				slotObservationState,
				slotObservationBase,
				slotVaultSharePrices,
				slotCuratorFees,
				slotHookFee,
			},
		},
	}, []any{&storageSlots})
	req1.AddCall(&ethrpc.Call{
		ABI:    bunniHookABI,
		Target: hookAddress,
		Method: "getBid",
		Params: []any{poolId, true},
	}, []any{&topBid})
	req1.AddCall(&ethrpc.Call{
		ABI:    bunniHookABI,
		Target: hookAddress,
		Method: "ldfStates",
		Params: []any{poolId},
	}, []any{&ldfState})
	req1.AddCall(&ethrpc.Call{
		ABI:    bunniHookABI,
		Target: hookAddress,
		Method: "slot0s",
		Params: []any{poolId},
	}, []any{&slot0})
	req1.AddCall(&ethrpc.Call{
		ABI:    bunniHubABI,
		Target: hubAddress,
		Method: "poolState",
		Params: []any{poolId},
	}, []any{&poolState})
	req1.AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: token0Address,
		Method: "balanceOf",
		Params: []any{poolManagerAddress},
	}, []any{&poolManagerBalance0})
	req1.AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: token1Address,
		Method: "balanceOf",
		Params: []any{poolManagerAddress},
	}, []any{&poolManagerBalance1})

	res, err := req1.Aggregate()
	if err != nil {
		return "", err
	}

	hookExtra.Slot0 = Slot0{
		SqrtPriceX96:       uint256.MustFromBig(slot0.SqrtPriceX96),
		Tick:               int(slot0.Tick.Int64()),
		LastSwapTimestamp:  slot0.LastSwapTimestamp,
		LastSurgeTimestamp: slot0.LastSurgeTimestamp,
	}

	hookExtra.BunniState = PoolState{
		TwapSecondsAgo:       uint32(poolState.Data.TwapSecondsAgo.Int64()),
		LdfParams:            poolState.Data.LdfParams,
		HookParams:           poolState.Data.HookParams,
		LdfType:              poolState.Data.LdfType,
		MinRawTokenRatio0:    uint256.MustFromBig(poolState.Data.MinRawTokenRatio0),
		TargetRawTokenRatio0: uint256.MustFromBig(poolState.Data.TargetRawTokenRatio0),
		MaxRawTokenRatio0:    uint256.MustFromBig(poolState.Data.MaxRawTokenRatio0),
		MinRawTokenRatio1:    uint256.MustFromBig(poolState.Data.MinRawTokenRatio1),
		TargetRawTokenRatio1: uint256.MustFromBig(poolState.Data.TargetRawTokenRatio1),
		MaxRawTokenRatio1:    uint256.MustFromBig(poolState.Data.MaxRawTokenRatio1),
		Currency0Decimals:    poolState.Data.Currency0Decimals,
		Currency1Decimals:    poolState.Data.Currency1Decimals,
		RawBalance0:          uint256.MustFromBig(poolState.Data.RawBalance0),
		RawBalance1:          uint256.MustFromBig(poolState.Data.RawBalance1),
		Reserve0:             uint256.MustFromBig(poolState.Data.Reserve0),
		Reserve1:             uint256.MustFromBig(poolState.Data.Reserve1),
		IdleBalance:          poolState.Data.IdleBalance,
	}

	hookExtra.HookletAddress = poolState.Data.Hooklet
	hookExtra.LDFAddress = poolState.Data.LiquidityDensityFunction
	hookExtra.LdfState = ldfState

	hookExtra.PoolManagerReserves = [2]*uint256.Int{
		uint256.MustFromBig(poolManagerBalance0),
		uint256.MustFromBig(poolManagerBalance1),
	}

	hookExtra.HookParams = decodeHookParams(poolState.Data.HookParams)
	hookExtra.ObservationState = decodeObservationState(storageSlots[0:2])
	hookExtra.VaultSharePrices = decodeVaultSharePrices(storageSlots[2])
	hookExtra.CuratorFees = decodeCuratorFees(storageSlots[3])
	hookExtra.HookFee = decodeHookFee(storageSlots[4])
	hookExtra.AmAmm = decodeAmmPayload(topBid.Data.Manager, topBid.Data.Payload)

	var (
		redeemRates = [2]*big.Int{big.NewInt(0), big.NewInt(0)}
		maxDeposits = [2]*big.Int{big.NewInt(0), big.NewInt(0)}
	)
	req2 := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(res.BlockNumber)
	for i, vault := range []common.Address{poolState.Data.Vault0, poolState.Data.Vault1} {
		if valueobject.IsZeroAddress(vault) {
			continue
		}

		req2.AddCall(&ethrpc.Call{
			ABI:    erc4626ABI,
			Target: vault.Hex(),
			Method: "previewRedeem",
			Params: []any{WAD.ToBig()},
		}, []any{&redeemRates[i]})
		req2.AddCall(&ethrpc.Call{
			ABI:    erc4626ABI,
			Target: vault.Hex(),
			Method: "maxDeposit",
			Params: []any{common.HexToAddress(hubAddress)},
		}, []any{&maxDeposits[i]})
	}

	if _, err := req2.TryBlockAndAggregate(); err != nil {
		return "", err
	}

	observationHashes, err := h.fetchObservations(
		ctx, param.RpcClient, res.BlockNumber, poolId, hookExtra.ObservationState.CardinalityNext)
	if err != nil {
		return "", err
	}

	hookExtra.Observations = decodeObservations(observationHashes)

	hookExtra.Vaults = [2]Vault{
		{
			Address:    poolState.Data.Vault0,
			Decimals:   poolState.Data.Vault0Decimals,
			RedeemRate: uint256.MustFromBig(redeemRates[0]),
			MaxDeposit: uint256.MustFromBig(maxDeposits[0]),
		},
		{
			Address:    poolState.Data.Vault1,
			Decimals:   poolState.Data.Vault1Decimals,
			RedeemRate: uint256.MustFromBig(redeemRates[1]),
			MaxDeposit: uint256.MustFromBig(maxDeposits[1]),
		},
	}

	if hookExtra.HookletExtra == "" {
		h.hooklet = InitHooklet(hookExtra.HookletAddress, "")
	}

	hookletExtra, err := h.hooklet.Track(ctx, hooklet.HookletParams{
		RpcClient:      param.RpcClient,
		HookletAddress: hookExtra.HookletAddress,
		HookletExtra:   hookExtra.HookletExtra,
		PoolId:         poolId,
	})
	if err != nil {
		return "", err
	}

	hookExtra.HookletExtra = hookletExtra

	newHookExtra, err := json.Marshal(&hookExtra)
	if err != nil {
		return "", err
	}

	return string(newHookExtra), nil
}

func (h *Hook) fetchObservations(
	ctx context.Context,
	rpcClient *ethrpc.Client,
	blockNumber *big.Int,
	poolId common.Hash,
	cardinalityNext uint32,
) ([]common.Hash, error) {
	observationsBaseSlot := crypto.Keccak256Hash(poolId[:], OBSERVATION_BASE_SLOT)

	var slotObservations = make([]common.Hash, 0, cardinalityNext)
	for i := range cardinalityNext {
		slotBig := big.NewInt(int64(i))
		slotBig.Add(slotBig, observationsBaseSlot.Big())
		slotObservations = append(slotObservations, common.BigToHash(slotBig))
	}

	var observationHashes = make([]common.Hash, 0, len(slotObservations))
	for start := 0; start < len(slotObservations); start += _MAX_OBSERVATION_BATCH_SIZE {
		end := min(start+_MAX_OBSERVATION_BATCH_SIZE, len(slotObservations))

		batchSlots := slotObservations[start:end]
		var batchResult = make([]common.Hash, len(batchSlots))

		reqBatch := rpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
		reqBatch.AddCall(&ethrpc.Call{
			ABI:    bunniHookABI,
			Target: h.hook.Hex(),
			Method: "extsload",
			Params: []any{batchSlots},
		}, []any{&batchResult})

		if _, err := reqBatch.TryBlockAndAggregate(); err != nil {
			return nil, err
		}

		observationHashes = append(observationHashes, batchResult...)
	}

	return observationHashes, nil
}
