package synthetix

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

type ExchangerWithFeeRecAlternativesReader struct {
	abi          abi.ABI
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

func NewExchangerWithFeeRecAlternativesReader(cfg *Config, ethrpcClient *ethrpc.Client) *ExchangerWithFeeRecAlternativesReader {
	return &ExchangerWithFeeRecAlternativesReader{
		abi:          exchangerWithFeeRecAlternatives,
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (r *ExchangerWithFeeRecAlternativesReader) Read(ctx context.Context, poolState *PoolState) (*PoolState, error) {
	if err := r.readData(ctx, poolState); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read")
		return nil, err
	}

	return poolState, nil

}

// readData reads data which required no parameters, included:
// - AtomicMaxVolumePerBlock
// - LastAtomicVolume
func (r *ExchangerWithFeeRecAlternativesReader) readData(ctx context.Context, poolState *PoolState) error {
	var (
		address          = poolState.Addresses.Exchanger
		lastAtomicVolume ExchangeVolumeAtPeriod
	)

	req := r.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: ExchangerWithFeeRecAlternativesMethodAtomicMaxVolumePerBlock,
			Params: nil,
		}, []interface{}{&poolState.AtomicMaxVolumePerBlock}).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: ExchangerWithFeeRecAlternativesMethodLastAtomicVolume,
			Params: nil,
		}, []interface{}{&lastAtomicVolume})

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read data")
		return err
	}

	poolState.LastAtomicVolume = &lastAtomicVolume

	return nil
}
