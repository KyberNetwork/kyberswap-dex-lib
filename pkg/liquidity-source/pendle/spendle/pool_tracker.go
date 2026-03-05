package spendle

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

type PoolTracker struct {
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryE0(DexType, NewPoolTracker)

func NewPoolTracker(ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	var extra Extra
	var reserves [2]*big.Int
	if _, err := t.ethrpcClient.R().SetContext(ctx).SetOverrides(overrides).AddCall(&ethrpc.Call{
		ABI:    StakedPendleABI,
		Target: p.Address,
		Method: "instantUnstakeFeeRate",
	}, []any{&extra.InstantUnstakeFeeRate}).AddCall(&ethrpc.Call{
		ABI:    abi.Erc20ABI,
		Target: p.Tokens[0].Address, // PENDLE
		Method: abi.Erc20BalanceOfMethod,
		Params: []any{common.HexToAddress(p.Tokens[1].Address)}, // sPENDLE
	}, []any{&reserves[0]}).AddCall(&ethrpc.Call{
		ABI:    abi.Erc20ABI,
		Target: p.Tokens[0].Address, // PENDLE
		Method: abi.Erc20TotalSupplyMethod,
	}, []any{&reserves[1]}).Aggregate(); err != nil {
		return p, err
	}

	extraBytes, _ := json.Marshal(extra)
	p.Reserves = entity.PoolReserves{reserves[0].String(), reserves[1].String()}
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	return p, nil
}
