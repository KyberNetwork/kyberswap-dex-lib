package prmfun

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func deskToken(chainID valueobject.ChainID) string {
	return strings.ToLower(valueobject.WrappedNativeMap[chainID])
}

var zeroSubjectId [32]byte

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	logger       logger.Logger
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
		logger:       logger.WithFields(logger.Fields{"dex_id": cfg.DexId}),
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	u.logger.Info("started getting new pools")

	var metadata PoolsListUpdaterMetadata
	if len(metadataBytes) > 0 {
		_ = json.Unmarshal(metadataBytes, &metadata)
	}

	var total uint64
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    memeFactoryABI,
			Target: u.config.FactoryAddress,
			Method: memeFactoryMethodMemeCount,
		}, []any{&total}).
		Call(); err != nil {
		u.logger.WithFields(logger.Fields{"error": err}).Error("failed to read memeCount")
		return nil, metadataBytes, err
	}

	limit := u.config.NewPoolLimit
	if limit <= 0 {
		limit = 100
	}
	start := metadata.Offset
	end := min(start+limit, int(total))
	if start >= end {
		newMetadataBytes, _ := json.Marshal(PoolsListUpdaterMetadata{Offset: start})
		return nil, newMetadataBytes, nil
	}

	tokenAddrs := make([]common.Address, end-start)
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i := start; i < end; i++ {
		req.AddCall(&ethrpc.Call{
			ABI:    memeFactoryABI,
			Target: u.config.FactoryAddress,
			Method: memeFactoryMethodMemeTokens,
			Params: []any{big.NewInt(int64(i))},
		}, []any{&tokenAddrs[i-start]})
	}
	if _, err := req.Aggregate(); err != nil {
		u.logger.WithFields(logger.Fields{"error": err}).Error("failed to read memeTokens")
		return nil, metadataBytes, err
	}

	memes := make([]GetMemeResult, len(tokenAddrs))
	req2 := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, addr := range tokenAddrs {
		req2.AddCall(&ethrpc.Call{
			ABI:    memeFactoryABI,
			Target: u.config.FactoryAddress,
			Method: memeFactoryMethodGetMeme,
			Params: []any{addr},
		}, []any{&memes[i]})
	}
	if _, err := req2.Aggregate(); err != nil {
		u.logger.WithFields(logger.Fields{"error": err}).Error("failed to read getMeme")
		return nil, metadataBytes, err
	}

	// Keep only ETH-paired candidates before spending a second round-trip on phase().
	type candidate struct {
		token          common.Address
		curve          common.Address
		graduationDesk string
	}
	var candidates []candidate
	for _, m := range memes {
		if m.Data.SubjectId != zeroSubjectId {
			continue // stock-paired - out of scope for this pool type
		}
		if m.Data.Curve == (common.Address{}) {
			continue // defensive: unresolved/zero curve
		}
		candidates = append(candidates, candidate{
			token:          m.Data.Token,
			curve:          m.Data.Curve,
			graduationDesk: m.Data.GraduationDesk.String(),
		})
	}

	newMetadataBytes, _ := json.Marshal(PoolsListUpdaterMetadata{Offset: end})

	if len(candidates) == 0 {
		return nil, newMetadataBytes, nil
	}

	phases := make([]uint8, len(candidates))
	req3 := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, c := range candidates {
		req3.AddCall(&ethrpc.Call{
			ABI:    memeCurveABI,
			Target: c.curve.Hex(),
			Method: memeCurveMethodPhase,
		}, []any{&phases[i]})
	}
	if _, err := req3.TryAggregate(); err != nil {
		u.logger.WithFields(logger.Fields{"error": err}).Error("failed to read phase")
		return nil, newMetadataBytes, err
	}

	pools := make([]entity.Pool, 0, len(candidates))
	for i, c := range candidates {
		if phases[i] != PhaseTrading {
			continue // graduated or paused at discovery time - belongs to a different pool type, or not yet tradeable
		}

		curveAddr := strings.ToLower(c.curve.Hex())
		tokenAddr := strings.ToLower(c.token.Hex())

		staticExtraBytes, _ := json.Marshal(StaticExtra{
			RouterAddress:  strings.ToLower(u.config.RouterAddress),
			CurveAddress:   curveAddr,
			MemeToken:      tokenAddr,
			GraduationDesk: c.graduationDesk,
		})

		pools = append(pools, entity.Pool{
			Address:     curveAddr,
			Exchange:    u.config.DexId,
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    []string{"0", "0"},
			StaticExtra: string(staticExtraBytes),
			Tokens: []*entity.PoolToken{
				{Address: deskToken(u.config.ChainId), Swappable: true},
				{Address: tokenAddr, Swappable: true},
			},
		})
	}

	u.logger.WithFields(logger.Fields{
		"new_pools": len(pools),
		"scanned":   end - start,
	}).Info("finished getting new pools")

	return pools, newMetadataBytes, nil
}
