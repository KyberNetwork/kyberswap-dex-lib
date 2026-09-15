package netstaking

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	utilabi "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
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
	overrides map[gethcommon.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	netAddr := p.Tokens[idxNET].Address
	sNetAddr := p.Tokens[idxSNET].Address

	extra, blockNumber, err := fetchDynamic(ctx, t.ethrpcClient, t.config.StakingAddress, t.config.WrapAddress, netAddr, sNetAddr, overrides)
	if err != nil {
		return p, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()

	p.Reserves = entity.PoolReserves{
		extra.NETReserve.ToBig().String(),
		extra.SNETStakingReserve.ToBig().String(),
		defaultReserve,
	}

	return p, nil
}

func fetchDynamic(
	ctx context.Context,
	ethrpcClient *ethrpc.Client,
	stakingAddress string,
	wrapAddress string,
	netAddr string,
	sNetAddr string,
	overrides map[gethcommon.Address]gethclient.OverrideAccount,
) (PoolExtra, uint64, error) {
	var indexBig *big.Int

	req1 := ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req1.SetOverrides(overrides)
	}
	req1.AddCall(&ethrpc.Call{
		ABI:    stakedNETABI,
		Target: sNetAddr,
		Method: "index",
	}, []any{&indexBig})

	resp1, err := req1.Aggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}
	if resp1.BlockNumber == nil {
		resp1.BlockNumber = big.NewInt(0)
	}

	var (
		netReserveBig         *big.Int
		sNetStakingReserveBig *big.Int
		sNetWrapReserveBig    *big.Int
	)
	req2 := ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req2.SetOverrides(overrides)
	}
	req2.AddCall(&ethrpc.Call{
		ABI:    utilabi.Erc20ABI,
		Target: netAddr,
		Method: utilabi.Erc20BalanceOfMethod,
		Params: []any{gethcommon.HexToAddress(stakingAddress)},
	}, []any{&netReserveBig}).AddCall(&ethrpc.Call{
		ABI:    utilabi.Erc20ABI,
		Target: sNetAddr,
		Method: utilabi.Erc20BalanceOfMethod,
		Params: []any{gethcommon.HexToAddress(stakingAddress)},
	}, []any{&sNetStakingReserveBig}).AddCall(&ethrpc.Call{
		ABI:    utilabi.Erc20ABI,
		Target: sNetAddr,
		Method: utilabi.Erc20BalanceOfMethod,
		Params: []any{gethcommon.HexToAddress(wrapAddress)},
	}, []any{&sNetWrapReserveBig})
	if _, err = req2.Aggregate(); err != nil {
		return PoolExtra{}, 0, err
	}

	return PoolExtra{
		Index:              uint256.MustFromBig(indexBig),
		NETReserve:         uint256.MustFromBig(netReserveBig),
		SNETStakingReserve: uint256.MustFromBig(sNetStakingReserveBig),
		SNETWrapReserve:    uint256.MustFromBig(sNetWrapReserveBig),
	}, resp1.BlockNumber.Uint64(), nil
}
