package umbraedamm

import (
	"context"
	"math/big"
	"strings"
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

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{config: config, ethrpcClient: ethrpcClient}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("getting new pool state")

	var (
		reserves struct {
			ReserveX *big.Int
			ReserveY *big.Int
		}
		feeBps   uint16
		feeToken common.Address
	)

	// Pin all reads to one block so reserves and the fee snapshot are consistent.
	resp, err := t.ethrpcClient.R().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: pairABI, Target: p.Address, Method: pairMethodGetReserves}, []any{&reserves}).
		AddCall(&ethrpc.Call{ABI: pairABI, Target: p.Address, Method: pairMethodCurrentFeeBps}, []any{&feeBps}).
		AddCall(&ethrpc.Call{ABI: pairABI, Target: p.Address, Method: pairMethodFeeToken}, []any{&feeToken}).
		TryBlockAndAggregate()
	if err != nil {
		return p, err
	}

	extraBytes, err := json.Marshal(Extra{
		FeeBps:   uint64(feeBps),
		FeeToken: strings.ToLower(feeToken.Hex()),
	})
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{reserves.ReserveX.String(), reserves.ReserveY.String()}
	p.BlockNumber = resp.BlockNumber.Uint64()
	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("finished getting new pool state")
	return p, nil
}
