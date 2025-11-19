package genericarm

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
)

func fetchAssetAndState(ctx context.Context, ethrpcClient *ethrpc.Client, armAddr string, armCfg ArmCfg) (*PoolState, error) {
	var poolState PoolState

	calls := ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    lidoArmABI,
		Target: armAddr,
		Method: "token0",
	}, []any{&poolState.Token0})
	calls.AddCall(&ethrpc.Call{
		ABI:    lidoArmABI,
		Target: armAddr,
		Method: "token1",
	}, []any{&poolState.Token1})

	if armCfg.ArmType == Pricable {
		calls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: armAddr,
			Method: "traderate0",
		}, []any{&poolState.TradeRate0})
		calls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: armAddr,
			Method: "traderate1",
		}, []any{&poolState.TradeRate1})
		calls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: armAddr,
			Method: "PRICE_SCALE",
		}, []any{&poolState.PriceScale})
	}
	if armCfg.HasWithdrawalQueue {
		calls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: armAddr,
			Method: "liquidityAsset",
		}, []any{&poolState.LiquidityAsset})
		calls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: armAddr,
			Method: "withdrawsQueued",
		}, []any{&poolState.WithdrawsQueued})
		calls.AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: armAddr,
			Method: "withdrawsClaimed",
		}, []any{&poolState.WithdrawsClaimed})
	}
	_, err := calls.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to initPool")
		return nil, err
	}

	calls.AddCall(&ethrpc.Call{
		ABI:    lidoArmABI,
		Target: poolState.Token0.Hex(),
		Method: "balanceOf",
		Params: []any{common.HexToAddress(armAddr)},
	}, []any{&poolState.Reserve0})
	calls.AddCall(&ethrpc.Call{
		ABI:    lidoArmABI,
		Target: poolState.Token1.Hex(),
		Method: "balanceOf",
		Params: []any{common.HexToAddress(armAddr)},
	}, []any{&poolState.Reserve1})
	_, err = calls.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to initPool")
		return nil, err
	}

	return &poolState, nil
}
