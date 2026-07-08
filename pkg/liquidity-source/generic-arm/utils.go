package genericarm

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// uint256FromBigOrNil converts a *big.Int to *uint256.Int, returning nil when v is nil. Used for
// fields like BuyPrice/SellPrice that are only populated for some ArmType values.
func uint256FromBigOrNil(v *big.Int) *uint256.Int {
	if v == nil {
		return nil
	}
	return uint256.MustFromBig(v)
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

	var baseAssets []common.Address
	switch armCfg.ArmType {
	case Pricable4626:
		// The upgraded ARM contract (AbstractARM) dropped the fixed token0()/token1() pair in favor of
		// liquidityAsset() (the quote asset) plus getBaseAssets() (tradeable base assets against it).
		// Pricable4626 only ever traded a single base asset, so Token0/Token1 keep working as before,
		// just sourced from the new getters.
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
		}, []any{&baseAssets})
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
		if len(baseAssets) == 0 {
			logger.WithFields(logger.Fields{
				"armAddr": armAddr,
			}).Errorf("failed to initPool: getBaseAssets returned no base asset")
			return nil, ErrFailedToFetchPoolState
		}
		poolState.Token1 = baseAssets[0]
		poolState.Vault.BaseAsset = baseAssets[0]
		poolState.PriceScale = bignumber.NewBig(priceScale4626)
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

	balanceCalls := ethrpcClient.NewRequest().SetContext(ctx)
	balanceCalls.AddCall(&ethrpc.Call{
		ABI:    lidoArmABI,
		Target: poolState.Token0.Hex(),
		Method: "balanceOf",
		Params: []any{common.HexToAddress(armAddr)},
	}, []any{&poolState.Reserve0})
	balanceCalls.AddCall(&ethrpc.Call{
		ABI:    lidoArmABI,
		Target: poolState.Token1.Hex(),
		Method: "balanceOf",
		Params: []any{common.HexToAddress(armAddr)},
	}, []any{&poolState.Reserve1})
	var baseAssetConfig BaseAssetConfig
	if armCfg.ArmType == Pricable4626 {
		balanceCalls.AddCall(&ethrpc.Call{
			ABI:    ERC626ABI,
			Target: poolState.Vault.BaseAsset.Hex(),
			Method: "totalAssets",
		}, []any{&poolState.Vault.TotalAssets})
		balanceCalls.AddCall(&ethrpc.Call{
			ABI:    ERC626ABI,
			Target: poolState.Vault.BaseAsset.Hex(),
			Method: "totalSupply",
		}, []any{&poolState.Vault.TotalSupply})
		balanceCalls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: armAddr,
			Method: "baseAssetConfigs",
			Params: []any{poolState.Vault.BaseAsset},
		}, []any{&baseAssetConfig})
	}
	if _, err := balanceCalls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to initPool")
		return nil, err
	}
	if armCfg.ArmType == Pricable4626 {
		poolState.Vault.BuyPrice = baseAssetConfig.BuyPrice
		poolState.Vault.SellPrice = baseAssetConfig.SellPrice
	}

	return &poolState, nil
}
