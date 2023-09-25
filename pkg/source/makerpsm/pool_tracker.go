package makerpsm

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolTracker struct {
	cfg       *Config
	psmReader *PSMReader
	vatReader *VatReader
}

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		cfg:       cfg,
		psmReader: NewPSMReader(ethrpcClient),
		vatReader: NewVatReader(ethrpcClient),
	}
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	pool entity.Pool,
	_ sourcePool.GetNewPoolStateParams,
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

	psm, err := d.getPSM(ctx, pool.Address)
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

func (d *PoolTracker) getPSM(ctx context.Context, address string) (*PSM, error) {
	psm, err := d.psmReader.Read(ctx, address)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("psm read error")
		return nil, err
	}

	vat, err := d.vatReader.Read(ctx, psm.VatAddress.String(), psm.ILK)
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
