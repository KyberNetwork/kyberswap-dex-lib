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

	if t.config.WrapAddress == "" {
		p.Reserves = entity.PoolReserves{
			extra.NETReserve.ToBig().String(),
			extra.SNETStakingReserve.ToBig().String(),
		}
	} else {
		p.Reserves = entity.PoolReserves{
			extra.NETReserve.ToBig().String(),
			extra.SNETStakingReserve.ToBig().String(),
			defaultReserve,
		}
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
	hasWrap := wrapAddress != ""

	var (
		indexBig    *big.Int
		blockNumber = big.NewInt(0)
	)

	// index() is only ever consumed by the wrap-side math (sNetToWs/wsToSNet), which
	// never runs when there's no wrap contract. Some staking forks' staked token (e.g.
	// NUKE's sNUKE) don't even expose an index() view, so skip the call entirely.
	if hasWrap {
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
		if resp1.BlockNumber != nil {
			blockNumber = resp1.BlockNumber
		}
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
	}, []any{&sNetStakingReserveBig})
	if hasWrap {
		req2.AddCall(&ethrpc.Call{
			ABI:    utilabi.Erc20ABI,
			Target: sNetAddr,
			Method: utilabi.Erc20BalanceOfMethod,
			Params: []any{gethcommon.HexToAddress(wrapAddress)},
		}, []any{&sNetWrapReserveBig})
	}
	resp2, err := req2.Aggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}
	if !hasWrap && resp2.BlockNumber != nil {
		blockNumber = resp2.BlockNumber
	}

	extra := PoolExtra{
		NETReserve:         uint256.MustFromBig(netReserveBig),
		SNETStakingReserve: uint256.MustFromBig(sNetStakingReserveBig),
	}
	if hasWrap {
		extra.Index = uint256.MustFromBig(indexBig)
		extra.SNETWrapReserve = uint256.MustFromBig(sNetWrapReserveBig)
	}

	return extra, blockNumber.Uint64(), nil
}
