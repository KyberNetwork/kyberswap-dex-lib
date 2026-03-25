package feltir

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	utilabi "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{cfg: cfg, ethrpcClient: ethrpcClient}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	log := logger.WithFields(logger.Fields{"dexId": t.cfg.DexID, "pool": p.Address})

	samples, reserves, blockNumber, err := t.fetchState(ctx, p)
	if err != nil {
		log.Errorf("feltir: fetchState failed: %v", err)
		return p, err
	}

	samples = filterSamples(samples)

	extraBytes, err := json.Marshal(Extra{Samples: samples})
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{reserves[0].String(), reserves[1].String()}
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = blockNumber.Uint64()

	return p, nil
}

func (t *PoolTracker) fetchState(ctx context.Context, p entity.Pool) ([2][][2]*big.Int, [2]*big.Int, *big.Int, error) {
	req := t.ethrpcClient.NewRequest().SetContext(ctx)

	var samples [2][][2]*big.Int
	var reserves [2]*big.Int

	for i := range p.Tokens {
		samples[i] = make([][2]*big.Int, sampleSize)
		tokenIn := common.HexToAddress(p.Tokens[i].Address)
		tokenOut := common.HexToAddress(p.Tokens[1-i].Address)

		dec := int(p.Tokens[i].Decimals)
		start := max(0, dec-sampleSize/2)

		for idx, k := 0, start; idx < sampleSize; idx, k = idx+1, k+1 {
			samples[i][idx] = [2]*big.Int{bignumber.TenPowInt(k), big.NewInt(0)}
			req.AddCall(&ethrpc.Call{
				ABI:    feltirABI,
				Target: t.cfg.FeltirAddress,
				Method: "getAmountOut",
				Params: []any{tokenIn, tokenOut, samples[i][idx][0]},
			}, []any{&samples[i][idx][1]})
		}
	}

	vaultAddr := common.HexToAddress(t.cfg.MakerAddress)
	for i, tok := range p.Tokens {
		reserves[i] = new(big.Int)
		req.AddCall(&ethrpc.Call{
			ABI:    utilabi.Erc20ABI,
			Target: tok.Address,
			Method: utilabi.Erc20BalanceOfMethod,
			Params: []any{vaultAddr},
		}, []any{&reserves[i]})
	}

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return [2][][2]*big.Int{}, [2]*big.Int{}, nil, err
	}

	return samples, reserves, res.BlockNumber, nil
}

func filterSamples(samples [2][][2]*big.Int) [2][][2]*big.Int {
	for dir := range samples {
		valid := samples[dir][:0]
		for _, s := range samples[dir] {
			if s[0] == nil || s[1] == nil || s[1].Sign() <= 0 {
				continue
			}
			valid = append(valid, s)
		}
		samples[dir] = valid
	}
	return samples
}
