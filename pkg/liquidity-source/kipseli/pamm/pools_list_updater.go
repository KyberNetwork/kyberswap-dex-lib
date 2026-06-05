package pamm

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type (
	PoolsListUpdater struct {
		cfg          *Config
		ethrpcClient *ethrpc.Client
	}

	PoolsListUpdaterMetadata struct {
		Offset int `json:"offset"`
	}
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{cfg: cfg, ethrpcClient: ethrpcClient}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	log := logger.WithFields(logger.Fields{"dexId": u.cfg.DexID})

	metadata, err := u.getMetadata(metadataBytes)
	if err != nil {
		log.Warnf("getMetadata failed: %v", err)
	}

	var quoteToken common.Address
	var listed []common.Address
	quoteTarget := common.HexToAddress(u.cfg.LensAddress)

	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: lensABI, Target: quoteTarget.Hex(), Method: "QUOTE_TOKEN"}, []any{&quoteToken}).
		AddCall(&ethrpc.Call{ABI: lensABI, Target: quoteTarget.Hex(), Method: "getListedTokens"}, []any{&listed}).
		TryAggregate(); err != nil {
		log.Errorf("quote target calls failed: %v", err)
		return nil, metadataBytes, err
	}

	tokens := make([]common.Address, 0, len(listed))
	for _, t := range listed {
		if !strings.EqualFold(t.Hex(), quoteToken.Hex()) {
			tokens = append(tokens, t)
		}
	}

	if metadata.Offset > len(tokens) {
		metadata.Offset = 0
	}
	if metadata.Offset == len(tokens) {
		return nil, metadataBytes, nil
	}
	tokens = tokens[metadata.Offset:]

	staticExtraBytes, _ := json.Marshal(StaticExtra{RouterAddress: strings.ToLower(u.cfg.RouterAddress)})

	pools := make([]entity.Pool, 0, len(tokens))
	for _, token := range tokens {
		pools = append(pools, entity.Pool{
			Address:   syntheticPoolAddress(u.cfg.DexID, token, quoteToken),
			Exchange:  u.cfg.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(token[:]), Swappable: true},
				{Address: hexutil.Encode(quoteToken[:]), Swappable: true},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
		})
	}

	newMetadata, err := u.newMetadata(metadata.Offset + len(tokens))
	if err != nil {
		log.Warnf("newMetadata failed: %v", err)
		return pools, metadataBytes, nil
	}
	return pools, newMetadata, nil
}

func (u *PoolsListUpdater) getMetadata(metadataBytes []byte) (PoolsListUpdaterMetadata, error) {
	if len(metadataBytes) == 0 {
		return PoolsListUpdaterMetadata{}, nil
	}
	var metadata PoolsListUpdaterMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return PoolsListUpdaterMetadata{}, err
	}
	return metadata, nil
}

func (u *PoolsListUpdater) newMetadata(offset int) ([]byte, error) {
	return json.Marshal(PoolsListUpdaterMetadata{Offset: offset})
}

func syntheticPoolAddress(dexID string, token, quote common.Address) string {
	return strings.ToLower(dexID + "_" + token.Hex() + "_" + quote.Hex())
}
