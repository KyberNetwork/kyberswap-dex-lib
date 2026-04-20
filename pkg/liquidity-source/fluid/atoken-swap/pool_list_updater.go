package atokenswap

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       Config
	ethrpcClient *ethrpc.Client
}

type Metadata struct {
	LastSyncTime uint64 `json:"lastSyncTime"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       *config,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	logger.WithFields(logger.Fields{
		"dexType": DexType,
	}).Infof("Start updating pools list ...")
	defer func() {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
		}).Infof("Finish updating pools list.")
	}()

	var metadata Metadata
	_ = json.Unmarshal(metadataBytes, &metadata)

	// Single pool with n+1 tokens: aEthWETH (input) + all output tokens
	pools := []entity.Pool{
		u.createPool(),
	}

	metadata.LastSyncTime = uint64(time.Now().Unix())
	newMetadataBytes, _ := json.Marshal(metadata)

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) createPool() entity.Pool {
	// Hardcoded output tokens
	outputTokens := []common.Address{AEthwstETH, AEthweETH}
	outputTokenStrs := make([]string, len(outputTokens))
	for i, token := range outputTokens {
		outputTokenStrs[i] = hexutil.Encode(token[:])
	}

	tokens := []*entity.PoolToken{
		{Address: hexutil.Encode(AEthWETH[:]), Swappable: true},
	}
	for _, outputTokenStr := range outputTokenStrs {
		tokens = append(tokens, &entity.PoolToken{Address: outputTokenStr, Swappable: true})
	}

	// Create reserves array: input + each output
	reserves := make(entity.PoolReserves, len(outputTokens)+1)
	for i := range reserves {
		reserves[i] = "0"
	}

	return entity.Pool{
		Address:  strings.ToLower(u.config.ATokenSwapAddr),
		Exchange: u.config.DexID,
		Type:     DexType,
		Reserves: reserves,
		Tokens:   tokens,
		Extra:    "{}",
	}
}
