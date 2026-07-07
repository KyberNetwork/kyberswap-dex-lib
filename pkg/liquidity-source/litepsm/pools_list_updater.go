package litepsm

import (
	"context"
	"encoding/binary"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	cfg            *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexTypeLitePSM, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if d.hasInitialized {
		return nil, nil, nil
	}
	logger.WithFields(logger.Fields{"dexID": d.cfg.DexID}).Info("get new pools")
	d.hasInitialized = true

	pools := make([]entity.Pool, 0, len(d.cfg.PSMs))
	for psm, psmCfg := range d.cfg.PSMs {
		newPool, err := d.newPool(ctx, psm, psmCfg)
		if err != nil {
			logger.WithFields(logger.Fields{
				"dexID": d.cfg.DexID,
				"error": err,
			}).Error("can not create new pool")
			d.hasInitialized = false
			continue
		}
		pools = append(pools, newPool)
	}

	logger.WithFields(logger.Fields{"dexID": d.cfg.DexID}).Info("get new pools successfully")
	return pools, nil, nil
}

func (d *PoolsListUpdater) newPool(ctx context.Context, psmStr string, psmCfg PSMConfig) (entity.Pool, error) {
	var pocket, dai, innerPsm, innerDai, gemJoin, gem common.Address
	genericPsmAbi := abi.ABI{
		Methods: map[string]abi.Method{
			genericMethodDai: {
				ID: binary.BigEndian.AppendUint32(make([]byte, 0, 4), psmCfg.DaiSelector),
				Outputs: abi.Arguments{
					{Type: abi.Type{T: abi.AddressTy}},
				},
			},
		},
	}

	resp, err := d.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    LitePSMABI,
		Target: psmStr,
		Method: litePSMMethodPocket,
	}, []any{&pocket}).AddCall(&ethrpc.Call{
		ABI:    genericPsmAbi,
		Target: psmStr,
		Method: genericMethodDai,
	}, []any{&dai}).AddCall(&ethrpc.Call{
		ABI:    LitePSMABI,
		Target: psmStr,
		Method: litePSMMethodPsm,
	}, []any{&innerPsm}).TryAggregate()
	if err != nil {
		return entity.Pool{}, err
	} else if !resp.Result[1] {
		return entity.Pool{}, ErrInvalidToken
	}

	staticExtra := StaticExtra{IsMint: psmCfg.IsMint}
	if resp.Result[0] {
		staticExtra.Pocket = &pocket
	}
	innerPsmStr := psmStr
	if resp.Result[2] {
		innerPsmStr = hexutil.Encode(innerPsm[:])
	}

	psm := common.HexToAddress(psmStr)
	if _, err = d.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    LitePSMABI,
		Target: innerPsmStr,
		Method: litePSMMethodGemJoin,
	}, []any{&gemJoin}).Call(); err != nil {
		return entity.Pool{}, err
	} else if psm != gemJoin {
		staticExtra.GemJoin = &gemJoin
	}

	req := d.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    LitePSMABI,
		Target: hexutil.Encode(gemJoin[:]),
		Method: litePSMMethodGem,
	}, []any{&gem})
	if innerPsmStr != psmStr {
		req = req.AddCall(&ethrpc.Call{
			ABI:    genericPsmAbi,
			Target: innerPsmStr,
			Method: genericMethodDai,
		}, []any{&innerDai})
	}
	if _, err = req.TryAggregate(); err != nil {
		return entity.Pool{}, err
	} else if innerDai != valueobject.AddrZero {
		staticExtra.Dai = &innerDai
	}

	staticExtraBytes, _ := json.Marshal(staticExtra)

	return entity.Pool{
		Address:  strings.ToLower(psmStr),
		Exchange: d.cfg.DexID,
		Type:     DexTypeLitePSM,
		Tokens: []*entity.PoolToken{
			{Address: hexutil.Encode(dai[:]), Swappable: true},
			{Address: hexutil.Encode(gem[:]), Swappable: true},
		},
		Reserves:    entity.PoolReserves{"0", "0"},
		Timestamp:   time.Now().Unix(),
		StaticExtra: string(staticExtraBytes),
	}, nil
}
