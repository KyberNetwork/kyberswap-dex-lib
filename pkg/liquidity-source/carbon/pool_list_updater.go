package carbon

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolsListUpdater(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	logger.WithFields(logger.Fields{
		"dexId": u.config.DexId,
	}).Info("start getting new pools")

	pairs, err := getPairs(ctx, u.ethrpcClient, u.config.Controller)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId": u.config.DexId,
			"error": err,
		}).Error("failed to get pairs")
		return nil, nil, err
	}

	pools := make([]entity.Pool, 0, len(pairs))

	for _, pair := range pairs {
		token0 := hexutil.Encode(pair[0][:])
		token1 := hexutil.Encode(pair[1][:])

		staticExtra := StaticExtra{
			Token0:     token0,
			Token1:     token1,
			Controller: u.config.Controller.String(),
		}
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			continue
		}

		pool := entity.Pool{
			Address:   generatePoolAddress(u.config.Controller, pair[0], pair[1]),
			Exchange:  string(u.config.DexId),
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: valueobject.WrapNativeLower(token0, u.config.ChainId), Swappable: true},
				{Address: valueobject.WrapNativeLower(token1, u.config.ChainId), Swappable: true},
			},
			StaticExtra: string(staticExtraBytes),
		}

		pools = append(pools, pool)
	}

	logger.WithFields(logger.Fields{
		"dexId":    u.config.DexId,
		"numPools": len(pools),
	}).Info("finished getting new pools")

	return pools, nil, nil
}

func getPairs(ctx context.Context, ethrpcClient *ethrpc.Client, controller common.Address) ([][2]common.Address, error) {
	var pairs [][2]common.Address
	if _, err := ethrpcClient.R().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    controllerABI,
		Target: controller.String(),
		Method: "pairs",
	}, []any{&pairs}).Call(); err != nil {
		return nil, err
	}

	return pairs, nil
}

func generatePoolAddress(controller, token0, token1 common.Address) string {
	if bytes.Compare(token0.Bytes(), token1.Bytes()) > 0 {
		token0, token1 = token1, token0
	}

	hash := sha256.Sum256([]byte(strings.ToLower(controller.String() + "_" + token0.String() + "_" + token1.String())))

	return "0x" + hex.EncodeToString(hash[:])
}
