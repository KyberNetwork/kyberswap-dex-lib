package makerpsm

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	cfg       *Config
	psmReader *PSMReader
	vatReader *VatReader
}

var _ = pooltrack.RegisterFactoryCE0(DexTypeMakerPSM, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		cfg:       cfg,
		psmReader: NewPSMReader(ethrpcClient),
		vatReader: NewVatReader(ethrpcClient),
	}
}

func (d *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params sourcePool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return d.getNewPoolState(ctx, p, sourcePool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params sourcePool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return d.getNewPoolState(ctx, p, params, nil)
}

func (d *PoolTracker) getNewPoolState(
	ctx context.Context,
	pool entity.Pool,
	_ sourcePool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	defer func(startTime time.Time) {
		logger.
			WithFields(logger.Fields{
				"dexID":             d.cfg.DexID,
				"poolsUpdatedCount": "1",
				"duration":          time.Since(startTime).Milliseconds(),
			}).
			Info("finished GetNewPoolState")
	}(time.Now())

	psm, err := d.getPSM(ctx, pool.Address, overrides)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("get psm error")
		return entity.Pool{}, err
	}

	extra := struct {
		PSM *PSM `json:"psm"`
	}{
		PSM: psm,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not marshal extra")
		return entity.Pool{}, err
	}

	reserve0 := new(big.Int).Sub(
		new(big.Int).Div(
			psm.Vat.ILK.Line,
			psm.Vat.ILK.Rate,
		),
		psm.Vat.ILK.Art,
	)
	reserve1 := psm.Vat.ILK.Art

	pool.Reserves = []string{reserve0.String(), reserve1.String()}
	pool.Extra = string(extraBytes)
	pool.Timestamp = time.Now().Unix()

	return pool, nil
}

func (d *PoolTracker) getPSM(
	ctx context.Context,
	address string,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*PSM, error) {
	psm, err := d.psmReader.Read(ctx, address, overrides)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("psm read error")
		return nil, err
	}

	vat, err := d.vatReader.Read(ctx, psm.VatAddress.String(), psm.ILK, overrides)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("vat read error")
		return nil, err
	}
	psm.Vat = vat

	return psm, nil
}
