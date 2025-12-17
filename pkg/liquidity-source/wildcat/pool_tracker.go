package wildcat

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		config:       config,
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
	_ map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	pools, err := TrackPools(ctx, []entity.Pool{p}, t.ethrpcClient, t.config)
	if err != nil {
		return p, err
	}
	return pools[0], nil
}

func TrackPools(ctx context.Context, pools []entity.Pool, rpcClient *ethrpc.Client, cfg *Config) ([]entity.Pool, error) {
	req := rpcClient.NewRequest().SetContext(ctx)
	rates := make([][]*big.Int, len(pools))
	reserves := make([][]*big.Int, len(pools))
	extras := make([]Extra, len(pools))
	for i, pool := range pools {
		err := json.Unmarshal([]byte(pool.Extra), &extras[i])
		if err != nil {
			return nil, err
		}
		reserves[i] = make([]*big.Int, 2)
		for j, token := range pool.Tokens {
			if !extras[i].IsNative[j] {
				req.AddCall(&ethrpc.Call{
					ABI:    erc20ABI,
					Target: token.Address,
					Method: "balanceOf",
					Params: []any{common.HexToAddress(pool.Address)},
				}, []any{&reserves[i][j]})
			} else {
				req.AddCall(&ethrpc.Call{
					ABI:    multicallABI,
					Target: cfg.MulticallAddress,
					Method: "getEthBalance",
					Params: []any{common.HexToAddress(pool.Address)},
				}, []any{&reserves[i][j]})
			}
		}
	}
	_, err := req.Aggregate()
	if err != nil {
		return nil, err
	}

	req = rpcClient.NewRequest().SetContext(ctx)
	for i, pool := range pools {
		rates[i] = make([]*big.Int, 2)
		for j := range pool.Tokens {
			if reserves[i][(j+1)%2].Sign() == 0 {
				rates[i][j] = big.NewInt(0)
				continue
			}
			req.AddCall(&ethrpc.Call{
				ABI:    pairABI,
				Target: pool.Address,
				Method: "getAmountIn",
				Params: []any{j == 0, reserves[i][(j+1)%2]}, // true = 0->1 (getAmountIn(zero_for_one, amount_out))
			}, []any{&rates[i][j]})
		}
	}
	_, err = req.Aggregate()
	if err != nil {
		return nil, err
	}

	for i := range pools {
		pools[i].Reserves = []string{reserves[i][0].String(), reserves[i][1].String()}
		extras[i].Rates = rates[i]
		extraBytes, err := json.Marshal(extras[i])
		if err != nil {
			return nil, err
		}
		pools[i].Extra = string(extraBytes)
		pools[i].Timestamp = time.Now().Unix()
	}
	return pools, nil
}
