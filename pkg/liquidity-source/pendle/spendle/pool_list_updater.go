package spendle

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client

	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolListUpdater)

func NewPoolListUpdater(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolListUpdater {
	return &PoolListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, nil, nil
	}
	l := log.Ctx(ctx).With().Str("dexID", u.config.DexID).Logger()
	l.Info().Msg("start getting new pools")

	startTime := time.Now()
	u.hasInitialized = true

	sPendle := strings.ToLower(u.config.Address)
	var pendle common.Address
	if _, err := u.ethrpcClient.R().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    StakedPendleABI,
		Target: sPendle,
		Method: "PENDLE",
	}, []any{&pendle}).Call(); err != nil {
		return nil, nil, err
	}

	logger.
		WithFields(
			logger.Fields{
				"dex_id":      DexType,
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pools")

	return []entity.Pool{
		{
			Address:   sPendle,
			Exchange:  valueobject.ExchangeSPendle,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: entity.PoolTokens{
				{Address: hexutil.Encode(pendle[:]), Swappable: true},
				{Address: sPendle, Swappable: true},
			},
		},
	}, nil, nil
}
