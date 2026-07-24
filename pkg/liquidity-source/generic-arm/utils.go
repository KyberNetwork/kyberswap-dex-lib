package genericarm

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// uint256FromBigOrNil converts a *big.Int to *uint256.Int, returning nil when v is nil. Used for
// fields that are only populated for some ArmType values.
func uint256FromBigOrNil(v *big.Int) *uint256.Int {
	if v == nil {
		return nil
	}
	return uint256.MustFromBig(v)
}

// buildExtra converts fetched on-chain state into the persisted Extra, shared by the pool lister and
// tracker so the two never drift on which fields are nil-safe for a given ArmType.
func buildExtra(poolState *PoolState, armCfg ArmCfg) Extra {
	return Extra{
		TradeRate0:             uint256FromBigOrNil(poolState.TradeRate0),
		TradeRate1:             uint256FromBigOrNil(poolState.TradeRate1),
		PriceScale:             uint256FromBigOrNil(poolState.PriceScale),
		WithdrawsQueued:        uint256FromBigOrNil(poolState.WithdrawsQueued),
		WithdrawsClaimed:       uint256FromBigOrNil(poolState.WithdrawsClaimed),
		LiquidityAsset:         poolState.LiquidityAsset,
		LiquidityAssetDecimals: poolState.LiquidityAssetDecimals,
		SwapTypes:              armCfg.SwapType,
		ArmType:                armCfg.ArmType,
		HasWithdrawalQueue:     armCfg.HasWithdrawalQueue,
		Gas:                    Gas(armCfg.Gas),
		BaseAssets:             toBaseAssetInfos(poolState.BaseAssets),
	}
}

func toBaseAssetInfos(baseAssets []PoolStateBaseAsset) []BaseAssetInfo {
	if baseAssets == nil {
		return nil
	}
	out := make([]BaseAssetInfo, len(baseAssets))
	for i, ba := range baseAssets {
		out[i] = BaseAssetInfo{
			Decimals:                  ba.Decimals,
			PeggedToLiquidityAsset:    ba.PeggedToLiquidityAsset,
			BuyPrice:                  uint256FromBigOrNil(ba.BuyPrice),
			SellPrice:                 uint256FromBigOrNil(ba.SellPrice),
			BuyLiquidityRemaining:     uint256FromBigOrNil(ba.BuyLiquidityRemaining),
			SellLiquidityRemaining:    uint256FromBigOrNil(ba.SellLiquidityRemaining),
			ConvertRateAssetsPerShare: uint256FromBigOrNil(ba.ConvertRateAssetsPerShare),
			ConvertRateSharesPerAsset: uint256FromBigOrNil(ba.ConvertRateSharesPerAsset),
		}
	}
	return out
}

// buildTokensAndReserves builds the entity.Pool token list and matching reserves. For ArmType
// Pricable4626 the pool has N+1 tokens: [liquidityAsset, baseAssets...] (star topology); for other
// ArmTypes it stays the original fixed [token0, token1] pair.
func buildTokensAndReserves(poolState *PoolState, armCfg ArmCfg) ([]*entity.PoolToken, entity.PoolReserves) {
	if armCfg.ArmType != Pricable4626 {
		return []*entity.PoolToken{
				{Address: hexutil.Encode(poolState.Token0[:]), Swappable: true},
				{Address: hexutil.Encode(poolState.Token1[:]), Swappable: true},
			}, entity.PoolReserves{
				poolState.Reserve0.String(),
				poolState.Reserve1.String(),
			}
	}

	tokens := make([]*entity.PoolToken, 0, len(poolState.BaseAssets)+1)
	reserves := make(entity.PoolReserves, 0, len(poolState.BaseAssets)+1)
	tokens = append(tokens, &entity.PoolToken{Address: hexutil.Encode(poolState.Token0[:]), Swappable: true})
	reserves = append(reserves, poolState.Reserve0.String())
	for i, ba := range poolState.BaseAssets {
		tokens = append(tokens, &entity.PoolToken{Address: hexutil.Encode(ba.Address[:]), Swappable: true})
		reserves = append(reserves, poolState.BaseAssetReserves[i].String())
	}
	return tokens, reserves
}

func fetchAssetAndState(ctx context.Context, ethrpcClient *ethrpc.Client, armAddr string, armCfg ArmCfg) (*PoolState, error) {
	var poolState PoolState
	var withdrawsQueued, withdrawsClaimed, withdrawsQueuedShares, withdrawsClaimedShares *big.Int

	calls := ethrpcClient.NewRequest().SetContext(ctx)

	type requiredCall struct {
		idx    int
		method string
	}
	var requiredCalls []requiredCall
	idx := func() int { return len(calls.Calls) }
	addRequired := func(method string) { requiredCalls = append(requiredCalls, requiredCall{idx(), method}) }

	var baseAssetAddrs []common.Address
	switch armCfg.ArmType {
	case Pricable4626:
		// The upgraded ARM contract (AbstractARM) dropped the fixed token0()/token1() pair in favor of
		// liquidityAsset() (the quote asset) plus getBaseAssets() (one or more tradeable base assets
		// against it, e.g. EthenaARM has 1, WETH_ARM has 4: stETH/wstETH/eETH/weETH).
		addRequired("liquidityAsset")
		calls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: armAddr,
			Method: "liquidityAsset",
		}, []any{&poolState.Token0})
		addRequired("getBaseAssets")
		calls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: armAddr,
			Method: "getBaseAssets",
		}, []any{&baseAssetAddrs})
	default:
		addRequired("token0")
		calls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: armAddr,
			Method: "token0",
		}, []any{&poolState.Token0})
		addRequired("token1")
		calls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: armAddr,
			Method: "token1",
		}, []any{&poolState.Token1})
	}

	if armCfg.ArmType == Pricable {
		addRequired("traderate0")
		calls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: armAddr,
			Method: "traderate0",
		}, []any{&poolState.TradeRate0})
		addRequired("traderate1")
		calls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: armAddr,
			Method: "traderate1",
		}, []any{&poolState.TradeRate1})
		addRequired("PRICE_SCALE")
		calls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: armAddr,
			Method: "PRICE_SCALE",
		}, []any{&poolState.PriceScale})
	}

	var idxWithdrawsQueued, idxWithdrawsClaimed, idxWithdrawsQueuedShares, idxWithdrawsClaimedShares int
	if armCfg.HasWithdrawalQueue {
		addRequired("liquidityAsset")
		calls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: armAddr,
			Method: "liquidityAsset",
		}, []any{&poolState.LiquidityAsset})

		// withdrawsQueued/withdrawsClaimed are asset-denominated getters used by non-upgraded ARMs.
		// withdrawsQueuedShares/withdrawsClaimedShares are the share-denominated replacements used by
		// upgraded ARMs (e.g. EthenaARM). Both pairs are requested via TryAggregate so an ARM missing
		// either pair doesn't fail the whole batch; whichever pair succeeds is used below.
		idxWithdrawsQueued = idx()
		calls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: armAddr,
			Method: "withdrawsQueued",
		}, []any{&withdrawsQueued})
		idxWithdrawsClaimed = idx()
		calls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: armAddr,
			Method: "withdrawsClaimed",
		}, []any{&withdrawsClaimed})
		idxWithdrawsQueuedShares = idx()
		calls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: armAddr,
			Method: "withdrawsQueuedShares",
		}, []any{&withdrawsQueuedShares})
		idxWithdrawsClaimedShares = idx()
		calls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: armAddr,
			Method: "withdrawsClaimedShares",
		}, []any{&withdrawsClaimedShares})
	}

	res, err := calls.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to initPool")
		return nil, err
	}
	for _, rc := range requiredCalls {
		if !res.Result[rc.idx] {
			logger.WithFields(logger.Fields{
				"armAddr": armAddr,
				"method":  rc.method,
			}).Errorf("failed to initPool: required call reverted")
			return nil, ErrFailedToFetchPoolState
		}
	}

	if armCfg.ArmType == Pricable4626 {
		if len(baseAssetAddrs) == 0 {
			logger.WithFields(logger.Fields{
				"armAddr": armAddr,
			}).Errorf("failed to initPool: getBaseAssets returned no base asset")
			return nil, ErrFailedToFetchPoolState
		}
		poolState.PriceScale = bignumber.NewBig(priceScale4626)
		poolState.BaseAssets = make([]PoolStateBaseAsset, len(baseAssetAddrs))
		for i, addr := range baseAssetAddrs {
			poolState.BaseAssets[i].Address = addr
		}
	}

	if armCfg.HasWithdrawalQueue {
		switch {
		case res.Result[idxWithdrawsQueued] && res.Result[idxWithdrawsClaimed]:
			poolState.WithdrawsQueued = withdrawsQueued
			poolState.WithdrawsClaimed = withdrawsClaimed
		case res.Result[idxWithdrawsQueuedShares] && res.Result[idxWithdrawsClaimedShares]:
			convertCalls := ethrpcClient.NewRequest().SetContext(ctx)
			convertCalls.AddCall(&ethrpc.Call{
				ABI:    lidoArmABI,
				Target: armAddr,
				Method: "convertToAssets",
				Params: []any{withdrawsQueuedShares},
			}, []any{&poolState.WithdrawsQueued})
			convertCalls.AddCall(&ethrpc.Call{
				ABI:    lidoArmABI,
				Target: armAddr,
				Method: "convertToAssets",
				Params: []any{withdrawsClaimedShares},
			}, []any{&poolState.WithdrawsClaimed})
			if _, err := convertCalls.Aggregate(); err != nil {
				logger.WithFields(logger.Fields{
					"error": err,
				}).Errorf("failed to convert withdrawal queue shares to assets")
				return nil, err
			}
		default:
			return nil, ErrWithdrawalQueueState
		}
	}

	// Second round trip: reserves for every ArmType, plus (Pricable4626 only) each base asset's
	// decimals and baseAssetConfigs() pricing/adapter. All independent of each other, so they can be
	// aggregated together. baseAssetConfigs() tries two tuple shapes (see baseAssetConfigsV2ABI), which
	// requires TryAggregate; a genuine on-chain revert on any other (very standard, already-confirmed-live
	// target) call here is still treated as a hard failure via requiredCalls.
	balanceCalls := ethrpcClient.NewRequest().SetContext(ctx)
	requiredCalls = requiredCalls[:0]
	balanceIdx := func() int { return len(balanceCalls.Calls) }
	addBalanceRequired := func(method string) { requiredCalls = append(requiredCalls, requiredCall{balanceIdx(), method}) }

	addBalanceRequired("balanceOf(token0)")
	balanceCalls.AddCall(&ethrpc.Call{
		ABI:    lidoArmABI,
		Target: poolState.Token0.Hex(),
		Method: "balanceOf",
		Params: []any{common.HexToAddress(armAddr)},
	}, []any{&poolState.Reserve0})
	if armCfg.ArmType != Pricable4626 {
		addBalanceRequired("balanceOf(token1)")
		balanceCalls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: poolState.Token1.Hex(),
			Method: "balanceOf",
			Params: []any{common.HexToAddress(armAddr)},
		}, []any{&poolState.Reserve1})
	}

	baseAssetConfigs := make([]BaseAssetConfig, len(poolState.BaseAssets))
	if armCfg.ArmType == Pricable4626 {
		addBalanceRequired("decimals(liquidityAsset)")
		balanceCalls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: poolState.Token0.Hex(),
			Method: "decimals",
		}, []any{&poolState.LiquidityAssetDecimals})

		poolState.BaseAssetReserves = make([]*big.Int, len(poolState.BaseAssets))
		for i := range poolState.BaseAssets {
			baseAssetAddr := poolState.BaseAssets[i].Address
			addBalanceRequired("balanceOf(baseAsset)")
			balanceCalls.AddCall(&ethrpc.Call{
				ABI:    lidoArmABI,
				Target: baseAssetAddr.Hex(),
				Method: "balanceOf",
				Params: []any{common.HexToAddress(armAddr)},
			}, []any{&poolState.BaseAssetReserves[i]})
			addBalanceRequired("decimals(baseAsset)")
			balanceCalls.AddCall(&ethrpc.Call{
				ABI:    lidoArmABI,
				Target: baseAssetAddr.Hex(),
				Method: "decimals",
			}, []any{&poolState.BaseAssets[i].Decimals})
			addBalanceRequired("baseAssetConfigs")
			balanceCalls.AddCall(&ethrpc.Call{
				ABI:       lidoArmABI,
				Target:    armAddr,
				Method:    "baseAssetConfigs",
				Params:    []any{baseAssetAddr},
				UnpackABI: []abi.ABI{baseAssetConfigsV2ABI, lidoArmABI},
			}, []any{&baseAssetConfigs[i], &baseAssetConfigs[i]})
		}
	}
	balanceRes, err := balanceCalls.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to initPool")
		return nil, err
	}
	for _, rc := range requiredCalls {
		if !balanceRes.Result[rc.idx] {
			logger.WithFields(logger.Fields{
				"armAddr": armAddr,
				"method":  rc.method,
			}).Errorf("failed to initPool: required call reverted")
			return nil, ErrFailedToFetchPoolState
		}
	}

	if armCfg.ArmType != Pricable4626 {
		return &poolState, nil
	}

	for i := range poolState.BaseAssets {
		cfg := baseAssetConfigs[i]
		if cfg.BuyPrice == nil {
			logger.WithFields(logger.Fields{
				"armAddr":   armAddr,
				"baseAsset": poolState.BaseAssets[i].Address,
			}).Errorf("failed to initPool: baseAssetConfigs did not match either known tuple shape")
			return nil, ErrFailedToFetchPoolState
		}
		poolState.BaseAssets[i].PeggedToLiquidityAsset = cfg.PeggedToLiquidityAsset
		poolState.BaseAssets[i].BuyPrice = cfg.BuyPrice
		poolState.BaseAssets[i].SellPrice = cfg.SellPrice
		poolState.BaseAssets[i].BuyLiquidityRemaining = cfg.BuyLiquidityRemaining
		poolState.BaseAssets[i].SellLiquidityRemaining = cfg.SellLiquidityRemaining
	}

	// Third round trip (only for non-pegged base assets): snapshot each base asset's adapter
	// conversion rate. Every adapter implements the same convertToAssets(shares)/convertToShares(assets)
	// view functions (IAssetAdapter), regardless of the underlying mechanism (ERC4626 vault ratio for
	// sUSDe, Lido share math for stETH-family adapters, wstETH's own rate, etc), so this works uniformly
	// without protocol-specific logic on our side.
	rateCalls := ethrpcClient.NewRequest().SetContext(ctx)
	var hasRateCalls bool
	for i := range poolState.BaseAssets {
		ba := &poolState.BaseAssets[i]
		if ba.PeggedToLiquidityAsset {
			continue
		}
		hasRateCalls = true
		refShares := bignumber.TenPowInt(ba.Decimals)
		refAssets := bignumber.TenPowInt(poolState.LiquidityAssetDecimals)
		rateCalls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: baseAssetConfigs[i].Adapter.Hex(),
			Method: "convertToAssets",
			Params: []any{refShares},
		}, []any{&ba.ConvertRateAssetsPerShare})
		rateCalls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: baseAssetConfigs[i].Adapter.Hex(),
			Method: "convertToShares",
			Params: []any{refAssets},
		}, []any{&ba.ConvertRateSharesPerAsset})
	}
	if hasRateCalls {
		if _, err := rateCalls.Aggregate(); err != nil {
			logger.WithFields(logger.Fields{
				"error": err,
			}).Errorf("failed to fetch base asset adapter conversion rates")
			return nil, err
		}
	}

	return &poolState, nil
}
