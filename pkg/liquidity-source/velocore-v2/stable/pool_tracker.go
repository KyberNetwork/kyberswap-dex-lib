package stable

import (
	"context"
	"encoding/json"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"dexID":   d.cfg.DexID,
		"dexType": DexTypeVelocoreV2Stable,
		"address": p.Address,
	}).Infof("Start getting new state of pool")

	// query lens
	var poolDataResp poolDataResp
	req := d.ethrpcClient.R()
	req.AddCall(&ethrpc.Call{
		ABI:    lensABI,
		Target: d.cfg.LensAddress,
		Method: lensMethodQueryPool,
		Params: []interface{}{common.HexToAddress(p.Address)},
	}, []interface{}{&poolDataResp})
	if _, err := req.Call(); err != nil {
		logger.Error(err.Error())
		return p, err
	}

	// query pool
	tokenInfos := make([]tokenInfo, len(poolDataResp.Data.ListedTokens))
	req = d.ethrpcClient.R()
	for i, tokenBytes32 := range poolDataResp.Data.ListedTokens {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: poolMethodTokenInfo,
			Params: []interface{}{tokenBytes32},
		}, []interface{}{&tokenInfos[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		logger.Error(err.Error())
		return p, err
	}

	// transform
	poolDat := newPoolData(poolDataResp)
	tokenInfoMap := make(map[string]tokenInfo)
	for i, tokenInfo := range tokenInfos {
		tokenInfoMap[poolDat.Tokens[i].Address] = tokenInfo
	}

	extra := Extra{
		Amp:             poolDat.Amp,
		Fee1e18:         poolDat.Fee1e18,
		LpTokenBalances: poolDat.LpTokenBalances,
		TokenInfo:       tokenInfoMap,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.Errorf("failed to marshal extra data, err: %v", err)
		return p, err
	}

	p.Reserves = poolDat.PoolReserves
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	return p, nil
}
