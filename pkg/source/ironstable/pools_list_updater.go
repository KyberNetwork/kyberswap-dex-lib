package ironstable

import (
	"context"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/timer"
)

type PoolsListUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client

	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexTypeIronStable, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	finish := timer.Start(fmt.Sprintf("[%s] get new pools", d.cfg.DexID))
	defer finish()

	if d.hasInitialized {
		return nil, nil, nil
	}

	pools, err := d.loadPools()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not load pools")
		return nil, nil, err
	}

	ret := make([]entity.Pool, 0, len(pools))
	for _, p := range pools {
		var (
			multipliers []*big.Int
			swapStorage SwapStorage
		)

		req := d.ethrpcClient.
			NewRequest().
			SetContext(ctx).
			AddCall(&ethrpc.Call{
				ABI:    ironSwap,
				Target: p.ID,
				Method: ironSwapMethodGetTokenPrecisionMultipliers,
				Params: nil,
			}, []interface{}{&multipliers}).
			AddCall(&ethrpc.Call{
				ABI:    ironSwap,
				Target: p.ID,
				Method: ironSwapMethodSwapStorage,
				Params: nil,
			}, []interface{}{&swapStorage})

		_, err := req.Aggregate()
		if err != nil {
			logger.WithFields(logger.Fields{
				"dexID": d.cfg.DexID,
				"error": err,
			}).Error("can not call IronSwap contract")
			return nil, nil, err
		}

		var (
			tokens   = make([]*entity.PoolToken, 0, len(p.Tokens))
			reserves = make(entity.PoolReserves, 0, len(p.Tokens))

			staticExtra = PoolStaticExtra{
				LpToken: hexutil.Encode(swapStorage.LpToken[:]),
			}
		)

		for j, t := range p.Tokens {
			newToken := entity.PoolToken{
				Address:   t.Address,
				Swappable: true,
			}
			tokens = append(tokens, &newToken)
			reserves = append(reserves, poolTokenDefaultReserve)
			staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, multipliers[j].String())
		}

		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			logger.WithFields(logger.Fields{
				"dexID": d.cfg.DexID,
				"error": err,
			}).Error("can not marshal static extra")
			return nil, nil, err
		}

		newPool := entity.Pool{
			Address:     p.ID,
			Exchange:    d.cfg.DexID,
			Type:        string(DexTypeIronStable),
			Reserves:    reserves,
			Tokens:      tokens,
			StaticExtra: string(staticExtraBytes),
		}
		ret = append(ret, newPool)
	}

	d.hasInitialized = true

	return ret, nil, nil
}

func (d *PoolsListUpdater) loadPools() ([]Pool, error) {
	poolsBytes, ok := bytesByPath[d.cfg.PoolPath]
	if !ok {
		err := fmt.Errorf("key %s not found", d.cfg.PoolPath)
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not find pools")
		return nil, err
	}

	// unmarshal data
	var pools []Pool
	err := json.Unmarshal(poolsBytes, &pools)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not unmarshal pool data")
		return nil, err
	}

	return pools, nil
}
