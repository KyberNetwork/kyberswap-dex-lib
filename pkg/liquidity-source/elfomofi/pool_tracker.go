package elfomofi

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
	req := t.ethrpcClient.NewRequest().SetContext(ctx)

	samples := make([][][2]*big.Int, len(p.Tokens))
	for i := range p.Tokens {
		samples[i] = make([][2]*big.Int, sampleSize)
		start := lo.Ternary(p.Tokens[i].Decimals < sampleSize/2, 0, p.Tokens[i].Decimals-sampleSize/2)
		index := 0
		for k := start; k <= start+sampleSize-1; k++ {
			samples[i][index] = [2]*big.Int{bignumber.TenPowInt(k), big.NewInt(0)}
			req.AddCall(&ethrpc.Call{
				ABI:    factoryABI,
				Target: t.config.FactoryAddress,
				Method: "getAmountOut",
				Params: []any{
					common.HexToAddress(p.Tokens[i].Address),
					common.HexToAddress(p.Tokens[1-i].Address),
					samples[i][index][0],
				},
			}, []any{&samples[i][index][1]})
			index++
		}
	}

	_, err := req.TryAggregate()
	if err != nil {
		return entity.Pool{}, err
	}

	// Scale samples with buffer
	buffer := big.NewInt(t.config.Buffer)
	for i := range samples {
		for j := range samples[i] {
			if samples[i][j][1] != nil {
				samples[i][j][1].Mul(samples[i][j][1], buffer).Div(samples[i][j][1], bignumber.BasisPoint)
			}
		}
	}

	// Use the maximum amountOut as the token reserves
	var reserves [2]big.Int

	for i := range samples {
		for _, sample := range samples[i] {
			if sample[1] != nil && reserves[i].Cmp(sample[1]) < 0 {
				reserves[1-i].Set(sample[1])
			}
		}
	}

	extra := Extra{Samples: samples, FactoryAddress: t.config.FactoryAddress}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = []string{reserves[0].String(), reserves[1].String()}
	p.Timestamp = time.Now().Unix()

	return p, nil
}
