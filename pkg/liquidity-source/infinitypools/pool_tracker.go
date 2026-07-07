package infinitypools

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
)

type (
	PoolTracker struct {
		ethrpcClient *ethrpc.Client
	}
)

var _ = pooltrack.RegisterFactoryE0(DexType, NewPoolTracker)

func NewPoolTracker(ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{ethrpcClient: ethrpcClient}
}

func (u *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	var balanceToken0 *big.Int
	var balanceToken1 *big.Int

	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: p.Tokens[0].Address,
		Method: "balanceOf",
		Params: []any{common.HexToAddress(p.Address)},
	}, []any{&balanceToken0})

	req.AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: p.Tokens[1].Address,
		Method: "balanceOf",
		Params: []any{common.HexToAddress(p.Address)},
	}, []any{&balanceToken1})

	if _, err := req.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"err":         err,
		}).Errorf("[%s] failed to get pool balance", DexType)
		return entity.Pool{}, err
	}

	p.Reserves = entity.PoolReserves{balanceToken0.String(), balanceToken1.String()}
	p.Timestamp = time.Now().Unix()

	return p, nil
}
