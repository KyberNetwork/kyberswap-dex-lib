package mantisswap

import (
	"context"
	"encoding/json"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"strings"
	"time"
)

type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
}

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:         cfg,
		ethrpcClient:   ethrpcClient,
		hasInitialized: false,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if d.hasInitialized {
		return nil, nil, nil
	}

	pools, err := d.init(ctx)
	if err != nil {
		return nil, nil, err
	}

	d.hasInitialized = true

	return pools, nil, nil
}

func (d *PoolsListUpdater) init(ctx context.Context) ([]entity.Pool, error) {
	var (
		lpList = make([]common.Address, 10)
	)

	callLpList := d.ethrpcClient.NewRequest().SetContext(ctx)
	for i := 0; i < 10; i++ {
		callLpList.AddCall(&ethrpc.Call{
			ABI:    MainPoolABI,
			Target: d.config.MainPoolAddress,
			Method: mainPoolMethodLpList,
			Params: []interface{}{big.NewInt(int64(i))},
		}, []interface{}{&lpList[i]})
	}
	if _, err := callLpList.TryAggregate(); err != nil {
		logger.Errorf("failed to aggregate call with error %v", err)
		return nil, err
	}

	lenTokens := 0
	for _, lp := range lpList {
		if lp.Hex() == valueobject.ZeroAddress {
			break
		}
		lenTokens += 1
	}

	var underliers = make([]common.Address, lenTokens)
	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	for i := 0; i < lenTokens; i++ {
		calls.AddCall(&ethrpc.Call{
			ABI:    LPABI,
			Target: lpList[i].Hex(),
			Method: lpMethodUnderlier,
			Params: nil,
		}, []interface{}{&underliers[i]})
	}
	if _, err := calls.Aggregate(); err != nil {
		logger.Errorf("failed to aggregate call with error %v", err)
		return nil, err
	}

	var (
		tokens   = make([]*entity.PoolToken, len(underliers))
		reserves = make([]string, len(underliers))
		lps      = make(map[string]*LP, len(underliers))
	)
	for i, tokenAddress := range underliers {
		tokens[i] = &entity.PoolToken{
			Address:   strings.ToLower(tokenAddress.Hex()),
			Weight:    defaultWeight,
			Swappable: true,
		}
		reserves[i] = zeroString
		lps[tokens[i].Address] = &LP{
			Address: strings.ToLower(lpList[i].Hex()),
		}
	}

	extraBytes, err := json.Marshal(&Extra{
		LPs: lps,
	})
	if err != nil {
		logger.Errorf("failed to marshal lps with error %v", err)
		return nil, err
	}

	var newPool = entity.Pool{
		Address:   strings.ToLower(d.config.MainPoolAddress),
		Exchange:  d.config.DexID,
		Type:      DexTypeMantisSwap,
		Timestamp: time.Now().Unix(),
		Reserves:  reserves,
		Tokens:    tokens,
		Extra:     string(extraBytes),
	}

	logger.Infof("[%s] got pool %v from config", d.config.DexID, newPool.Address)

	return []entity.Pool{newPool}, nil
}
