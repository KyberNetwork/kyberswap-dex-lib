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
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
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
	pool, err := u.createPool(ctx)
	if err != nil {
		return nil, nil, err
	}
	pools := []entity.Pool{pool}

	metadata.LastSyncTime = uint64(time.Now().Unix())
	newMetadataBytes, _ := json.Marshal(metadata)

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) createPool(ctx context.Context) (entity.Pool, error) {
	// Fetch token addresses from contract helper methods
	var aWETH, aWstETH, aWeETH, aEzETH common.Address

	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req = req.AddCall(&ethrpc.Call{
		ABI:    aTokenSwapABI,
		Target: u.config.ATokenSwapAddr,
		Method: "aWETH",
	}, []any{&aWETH}).AddCall(&ethrpc.Call{
		ABI:    aTokenSwapABI,
		Target: u.config.ATokenSwapAddr,
		Method: "aWstETH",
	}, []any{&aWstETH}).AddCall(&ethrpc.Call{
		ABI:    aTokenSwapABI,
		Target: u.config.ATokenSwapAddr,
		Method: "aWeETH",
	}, []any{&aWeETH}).AddCall(&ethrpc.Call{
		ABI:    aTokenSwapABI,
		Target: u.config.ATokenSwapAddr,
		Method: "aEzETH",
	}, []any{&aEzETH})

	if _, err := req.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("failed to fetch token addresses from contract")
		return entity.Pool{}, err
	}

	// Build output tokens list (skip zero addresses for unsupported tokens)
	var tokens []*entity.PoolToken
	if valueobject.IsZeroAddress(aWETH) {
		for _, token := range []string{AEthWETH, AEthwstETH, AEthweETH} {
			tokens = append(tokens, &entity.PoolToken{Address: token, Swappable: true})
		}
	} else {
		for _, token := range []common.Address{aWETH, aWstETH, aWeETH, aEzETH} {
			if !valueobject.IsZeroAddress(token) {
				tokens = append(tokens, &entity.PoolToken{Address: hexutil.Encode(token[:]), Swappable: true})
			}
		}
	}

	// Create reserves array: input + each output
	return entity.Pool{
		Address:  strings.ToLower(u.config.ATokenSwapAddr),
		Exchange: u.config.DexID,
		Type:     DexType,
		Reserves: lo.Map(tokens, func(token *entity.PoolToken, _ int) string { return "0" }),
		Tokens:   tokens,
		Extra:    "{}",
	}, nil
}
