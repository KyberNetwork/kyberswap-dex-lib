package kipseliprop

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

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
	return &PoolsListUpdater{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	log := logger.WithFields(logger.Fields{"dexId": u.cfg.DexID})
	log.Info("kipseli-prop: start get pools")

	metadata, err := u.getMetadata(metadataBytes)
	if err != nil {
		log.Warnf("kipseli-prop: getMetadata failed: %v", err)
	}

	req := u.ethrpcClient.NewRequest().SetContext(ctx)

	var quoteToken common.Address
	var listed []common.Address

	req.AddCall(&ethrpc.Call{
		ABI:    lensABI,
		Target: u.cfg.LensAddress,
		Method: "getQuoteToken",
		Params: nil,
	}, []any{&quoteToken})

	req.AddCall(&ethrpc.Call{
		ABI:    lensABI,
		Target: u.cfg.LensAddress,
		Method: "getListedTokens",
		Params: nil,
	}, []any{&listed})

	if _, err := req.TryAggregate(); err != nil {
		log.Errorf("kipseli-prop: lens calls failed: %v", err)
		return nil, metadataBytes, err
	}

	tokens := make([]common.Address, 0, len(listed))
	for _, t := range listed {
		if strings.EqualFold(t.Hex(), quoteToken.Hex()) {
			continue
		}
		tokens = append(tokens, t)
	}

	if metadata.Offset > len(tokens) {
		metadata.Offset = 0
	}
	if metadata.Offset == len(tokens) {
		return nil, metadataBytes, nil
	}

	tokens = tokens[metadata.Offset:]

	staticExtraBytes, _ := json.Marshal(StaticExtra{
		RouterAddress: strings.ToLower(u.cfg.RouterAddress),
	})

	pools := make([]entity.Pool, 0, len(tokens))
	now := time.Now().Unix()

	for _, token := range tokens {
		p := entity.Pool{
			Address:   syntheticPoolAddress(u.cfg.DexID, token, quoteToken),
			Exchange:  u.cfg.DexID,
			Type:      DexType,
			Timestamp: now,
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: strings.ToLower(token.Hex()), Swappable: true},
				{Address: strings.ToLower(quoteToken.Hex()), Swappable: true},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
		}
		pools = append(pools, p)
	}

	// Lưu offset tích lũy cho lần chạy sau (đã xử lý metadata.Offset + len(tokens))
	newMetadata, err := u.newMetadata(metadata.Offset + len(tokens))
	if err != nil {
		log.Warnf("kipseli-prop: newMetadata failed: %v", err)
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
	metadata := PoolsListUpdaterMetadata{Offset: offset}
	return json.Marshal(metadata)
}

func syntheticPoolAddress(dexID string, token common.Address, quote common.Address) string {
	return strings.ToLower(dexID + "_" + token.Hex() + "_" + quote.Hex())
}
