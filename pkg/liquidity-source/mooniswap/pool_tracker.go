package mooniswap

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
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

	var staticExtra StaticExtra
	if len(p.StaticExtra) > 0 {
		_ = json.Unmarshal([]byte(p.StaticExtra), &staticExtra)
	}

	extra, blockNumber, err := t.getPoolState(ctx, p.Address, p.Tokens, staticExtra, nil)
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

	p.Reserves = entity.PoolReserves{
		u256.Min(extra.BalAdd0, extra.BalRem0).Dec(),
		u256.Min(extra.BalAdd1, extra.BalRem1).Dec(),
	}

	logger.WithFields(logger.Fields{"pool": p.Address}).Info("finished getting new pool state")

	return p, nil
}

func (t *PoolTracker) getPoolState(
	ctx context.Context,
	poolAddress string,
	tokens []*entity.PoolToken,
	staticExtra StaticExtra,
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
	if staticExtra.IsNativeToken0 {
		token0Addr = valueobject.AddrZero
	}
	if staticExtra.IsNativeToken1 {
		token1Addr = valueobject.AddrZero
	}

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
		Fee:         uint256.MustFromBig(fee),
		SlippageFee: uint256.MustFromBig(slippageFee),
		BalAdd0:     uint256.MustFromBig(balAdd0),
		BalAdd1:     uint256.MustFromBig(balAdd1),
		BalRem0:     uint256.MustFromBig(balRem0),
		BalRem1:     uint256.MustFromBig(balRem1),
	}, resp.BlockNumber.Uint64(), nil
}
