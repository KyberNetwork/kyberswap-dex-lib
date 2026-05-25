package gohm

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
	ohmAddr := p.Tokens[0].Address
	sohmAddr := p.Tokens[1].Address

	extra, blockNumber, err := fetchDynamic(ctx, t.ethrpcClient, p.Address, ohmAddr, sohmAddr, overrides)
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
		extra.OHMReserve.ToBig().String(),
		extra.SOHMReserve.ToBig().String(),
		defaultReserve,
	}

	return p, nil
}

func fetchDynamic(
	ctx context.Context,
	ethrpcClient *ethrpc.Client,
	stakingAddress string,
	ohmAddr string,
	sohmAddr string,
	overrides map[gethcommon.Address]gethclient.OverrideAccount,
) (PoolExtra, uint64, error) {
	var (
		indexBig     *big.Int
		warmupPeriod *big.Int
	)

	req1 := ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req1.SetOverrides(overrides)
	}
	req1.AddCall(&ethrpc.Call{
		ABI:    olympusStakingABI,
		Target: stakingAddress,
		Method: "index",
	}, []any{&indexBig}).AddCall(&ethrpc.Call{
		ABI:    olympusStakingABI,
		Target: stakingAddress,
		Method: "warmupPeriod",
	}, []any{&warmupPeriod})

	resp1, err := req1.Aggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}
	if resp1.BlockNumber == nil {
		resp1.BlockNumber = big.NewInt(0)
	}

	var (
		ohmReserveBig  *big.Int
		sohmReserveBig *big.Int
	)
	req2 := ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req2.SetOverrides(overrides)
	}
	req2.AddCall(&ethrpc.Call{
		ABI:    utilabi.Erc20ABI,
		Target: ohmAddr,
		Method: utilabi.Erc20BalanceOfMethod,
		Params: []any{gethcommon.HexToAddress(stakingAddress)},
	}, []any{&ohmReserveBig}).AddCall(&ethrpc.Call{
		ABI:    utilabi.Erc20ABI,
		Target: sohmAddr,
		Method: utilabi.Erc20BalanceOfMethod,
		Params: []any{gethcommon.HexToAddress(stakingAddress)},
	}, []any{&sohmReserveBig})
	if _, err = req2.Aggregate(); err != nil {
		return PoolExtra{}, 0, err
	}

	return PoolExtra{
		Index:        uint256.MustFromBig(indexBig),
		WarmupPeriod: warmupPeriod.Uint64(),
		OHMReserve:   uint256.MustFromBig(ohmReserveBig),
		SOHMReserve:  uint256.MustFromBig(sohmReserveBig),
	}, resp1.BlockNumber.Uint64(), nil
}
