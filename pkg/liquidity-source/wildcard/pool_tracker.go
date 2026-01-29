package wildcard

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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
	samples := make([][][][2]*big.Int, len(pools))
	for i, pool := range pools {
		samples[i] = make([][][2]*big.Int, len(pool.Tokens))
		for j := range pool.Tokens {
			samples[i][j] = make([][2]*big.Int, sampleSize)
			start := lo.Ternary(pool.Tokens[j].Decimals < sampleSize/2, 0, pool.Tokens[j].Decimals-sampleSize/2)
			end := pool.Tokens[j].Decimals + sampleSize/2
			index := 0
			for k := start; k <= end; k++ {
				samples[i][j][index] = [2]*big.Int{bignumber.TenPowInt(k), big.NewInt(0)}
				req.AddCall(&ethrpc.Call{
					ABI:    pairABI,
					Target: pool.Address,
					Method: "getAmountOut",
					Params: []any{j == 0, samples[i][j][index][0]}, // true = 0->1 (getAmountIn(zero_for_one, amount_out))
				}, []any{&samples[i][j][index][1]})
				index++
			}
		}
	}
	_, err = req.TryAggregate()
	if err != nil {
		return nil, err
	}
	buffer := big.NewInt(bps - cfg.PriceTolerance)
	for i := range samples {
		for j := range samples[i] {
			samples[i][j] = lo.Filter(samples[i][j], func(sample [2]*big.Int, _ int) bool {
				ok := sample[0] != nil && sample[1] != nil
				if ok {
					sample[1].Mul(sample[1], buffer).Div(sample[1], bignumber.BasisPoint)
				}
				return ok
			})
		}
	}
	for i := range pools {
		pools[i].Reserves = []string{reserves[i][0].String(), reserves[i][1].String()}
		extras[i].Samples = samples[i]
		extraBytes, err := json.Marshal(extras[i])
		if err != nil {
			return nil, err
		}
		pools[i].Extra = string(extraBytes)
		pools[i].Timestamp = time.Now().Unix()
	}
	return pools, nil
}
