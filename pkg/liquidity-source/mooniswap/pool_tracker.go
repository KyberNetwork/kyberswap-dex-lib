package mooniswap

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
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{"pool": p.Address}).Info("started getting new pool state")

	extra, blockNumber, err := t.getPoolState(ctx, p.Address, p.Tokens, nil)
	if err != nil {
		return p, err
	}

	if p.BlockNumber > blockNumber {
		return p, nil
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()

	token0Addr := common.HexToAddress(p.Tokens[0].Address)
	token1Addr := common.HexToAddress(p.Tokens[1].Address)
	p.Reserves = entity.PoolReserves{
		getMinStr(extra.BalAdd0, extra.BalRem0, token0Addr),
		getMinStr(extra.BalAdd1, extra.BalRem1, token1Addr),
	}

	logger.WithFields(logger.Fields{"pool": p.Address}).Info("finished getting new pool state")

	return p, nil
}

func (t *PoolTracker) getPoolState(
	ctx context.Context,
	poolAddress string,
	tokens []*entity.PoolToken,
	blockNumber *big.Int,
) (*Extra, uint64, error) {
	var (
		fee         *big.Int
		slippageFee *big.Int
		balAdd0     *big.Int
		balAdd1     *big.Int
		balRem0     *big.Int
		balRem1     *big.Int
	)

	token0Addr := common.HexToAddress(tokens[0].Address)
	token1Addr := common.HexToAddress(tokens[1].Address)

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber != nil {
		req.SetBlockNumber(blockNumber)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodFee,
	}, []any{&fee})
	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodSlippageFee,
	}, []any{&slippageFee})
	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetBalanceForAddition,
		Params: []any{token0Addr},
	}, []any{&balAdd0})
	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetBalanceForAddition,
		Params: []any{token1Addr},
	}, []any{&balAdd1})
	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetBalanceForRemoval,
		Params: []any{token0Addr},
	}, []any{&balRem0})
	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetBalanceForRemoval,
		Params: []any{token1Addr},
	}, []any{&balRem1})

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, 0, err
	}

	return &Extra{
		Fee:         fee.String(),
		SlippageFee: slippageFee.String(),
		BalAdd0:     balAdd0.String(),
		BalAdd1:     balAdd1.String(),
		BalRem0:     balRem0.String(),
		BalRem1:     balRem1.String(),
	}, resp.BlockNumber.Uint64(), nil
}

func getMinStr(a, b string, _ common.Address) string {
	aBig, ok1 := new(big.Int).SetString(a, 10)
	bBig, ok2 := new(big.Int).SetString(b, 10)
	if !ok1 || !ok2 {
		return "0"
	}
	if aBig.Cmp(bBig) < 0 {
		return a
	}
	return b
}
