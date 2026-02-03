package stabull

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kutils"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/go-resty/resty/v2"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	client       *resty.Client
}

type PoolsListUpdaterMetadata struct {
	Offset int `json:"offset"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	client := resty.NewWithClient(http.DefaultClient).
		SetBaseURL(config.HTTPConfig.BaseURL).
		SetTimeout(config.HTTPConfig.Timeout.Duration).
		SetRetryCount(config.HTTPConfig.RetryCount)

	return &PoolsListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
		client:       client,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	l := logger.WithFields(logger.Fields{"dexID": u.config.DexID})
	l.Info("Start getting new pools")

	var metadata Metadata
	if len(metadataBytes) != 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}
	}

	// Stabull uses factory.getCurve(base, quote) to discover pools
	// We query its api to get all the curves
	curves, err := u.listCurvesFromApi(ctx)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Errorf("listCurvesFromApi failed")
		return nil, nil, err
	}

	var poolsChecksum common.Address
	for _, curve := range curves {
		poolAddr := common.HexToAddress(curve)
		for i := range common.AddressLength {
			poolsChecksum[i] ^= poolAddr[i]
		}
	}
	if metadata.LastCount == len(curves) && metadata.LastPoolsChecksum == poolsChecksum {
		return nil, metadataBytes, nil
	}
	metadata.LastCount, metadata.LastPoolsChecksum = len(curves), poolsChecksum
	l.Infof("got %v curves", metadata.LastCount)

	pools, err := u.initPools(ctx, curves)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Errorf("initPools failed")
		return nil, metadataBytes, err
	}

	l.WithFields(logger.Fields{
		"pools_len": len(pools),
	}).Info("Finished getting new pools")
	return pools, metadataBytes, nil
}

// listCurvesFromApi queries API to discover pools
func (u *PoolsListUpdater) listCurvesFromApi(ctx context.Context) ([]string, error) {
	var resp struct {
		Data struct {
			Tokens []struct {
				Curve *string `json:"curve"`
			} `json:"tokens"`
		} `json:"data"`
	}
	if _, err := u.client.R().SetContext(ctx).SetResult(&resp).
		SetQueryString("env=PROD&chainId=" + kutils.Utoa(u.config.ChainID)).
		Get("api/v1/tokenData/get-tokens-info"); err != nil {
		return nil, err
	}

	curves := make([]string, 0, len(resp.Data.Tokens)-1)
	for _, token := range resp.Data.Tokens {
		if token.Curve != nil {
			curves = append(curves, strings.ToLower(*token.Curve))
		}
	}

	return curves, nil
}

const MaxBatchSize = 64

func (u *PoolsListUpdater) initPools(ctx context.Context, poolStrs []string) ([]entity.Pool, error) {
	if len(poolStrs) > MaxBatchSize {
		pools := make([]entity.Pool, 0, len(poolStrs))
		for poolStrsChunk := range slices.Chunk(poolStrs, MaxBatchSize) {
			poolsChunk, err := u.initPools(ctx, poolStrsChunk)
			if err != nil {
				return nil, err
			}
			pools = append(pools, poolsChunk...)
		}
		return pools, nil
	}

	tokens := make([][2]common.Address, len(poolStrs))
	req := u.ethrpcClient.R().SetContext(ctx)
	for i, pool := range poolStrs {
		req.AddCall(&ethrpc.Call{
			ABI:    stabullPoolABI,
			Target: pool,
			Method: poolMethodNumeraires,
			Params: []any{bignumber.ZeroBI},
		}, []any{&tokens[i][0]}).AddCall(&ethrpc.Call{
			ABI:    stabullPoolABI,
			Target: pool,
			Method: poolMethodNumeraires,
			Params: []any{bignumber.One},
		}, []any{&tokens[i][1]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, errors.New("failed to fetch tokens")
	}

	assimilators := make([][2]common.Address, len(poolStrs))
	req = u.ethrpcClient.R().SetContext(ctx)
	for i, pool := range poolStrs {
		req.AddCall(&ethrpc.Call{
			ABI:    stabullPoolABI,
			Target: pool,
			Method: poolMethodAssimilator,
			Params: []any{tokens[i][0]},
		}, []any{&assimilators[i][0]}).AddCall(&ethrpc.Call{
			ABI:    stabullPoolABI,
			Target: pool,
			Method: poolMethodAssimilator,
			Params: []any{tokens[i][1]},
		}, []any{&assimilators[i][1]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, errors.New("failed to fetch assimilators")
	}

	oracles := make([][2]common.Address, len(poolStrs))
	req = u.ethrpcClient.R().SetContext(ctx)
	for i, assimilator := range assimilators {
		req.AddCall(&ethrpc.Call{
			ABI:    assimilatorABI,
			Target: hexutil.Encode(assimilator[0][:]),
			Method: assimilatorMethodOracle,
		}, []any{&oracles[i][0]}).AddCall(&ethrpc.Call{
			ABI:    assimilatorABI,
			Target: hexutil.Encode(assimilator[1][:]),
			Method: assimilatorMethodOracle,
		}, []any{&oracles[i][1]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, errors.New("failed to fetch oracles")
	}

	pools := make([]entity.Pool, len(poolStrs))
	for i, poolStr := range poolStrs {
		staticExtraBytes, _ := json.Marshal(StaticExtra{Oracles: oracles[i]})

		pools[i] = entity.Pool{
			Address:   poolStr,
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(tokens[i][0][:]), Swappable: true},
				{Address: hexutil.Encode(tokens[i][1][:]), Swappable: true},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
		}
	}

	return pools, nil
}
