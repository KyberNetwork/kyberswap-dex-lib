package wcm

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	logger       logger.Logger
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	lg := logger.WithFields(logger.Fields{
		"dexId":   config.DexID,
		"dexType": DexType,
	})

	return &PoolsListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
		logger:       lg,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	u.logger.Infof("Start updating pools list ...")
	defer func() {
		u.logger.Infof("Finish updating pools list.")
	}()

	var metadata Metadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			u.logger.Errorf("failed to unmarshal metadata: %v", err)
		}
	}

	tokenMap, err := u.fetchTokenConfigs(ctx)
	if err != nil {
		u.logger.Errorf("failed to fetch highest token id and token configs: %v", err)
		return nil, nil, err
	}
	if len(tokenMap) == 0 {
		return nil, nil, nil
	}

	tradingPairs, err := u.getTradingPairs(ctx, tokenMap)
	if err != nil {
		u.logger.Errorf("failed to get trading pairs: %v", err)
		return nil, nil, err
	}

	seenSet := make(map[uint64]struct{})
	for _, key := range metadata.SeenPairKeys {
		seenSet[key] = struct{}{}
	}
	var newPools []entity.Pool
	var newlySeen []uint64
	for _, pair := range tradingPairs {
		minId, maxId := pair.TokenID1, pair.TokenID2
		if minId > maxId {
			minId, maxId = maxId, minId
		}
		key := (uint64(minId) << 32) | uint64(maxId)
		if _, ok := seenSet[key]; ok {
			continue
		}
		pool, err := u.createPool(pair)
		if err != nil {
			u.logger.Errorf("failed to create pool for pair %s/%s: %v",
				pair.TokenA.Hex(), pair.TokenB.Hex(), err)
			continue
		}
		seenSet[key] = struct{}{}
		newlySeen = append(newlySeen, key)
		newPools = append(newPools, pool)
	}

	metadata.SeenPairKeys = append(metadata.SeenPairKeys, newlySeen...)
	newMetadataBytes, err := json.Marshal(metadata)
	if err != nil {
		u.logger.Errorf("failed to marshal metadata: %v", err)
		return nil, nil, err
	}

	u.logger.Infof("Found %d new pools (seen total %d pairs)", len(newPools), len(metadata.SeenPairKeys))
	return newPools, newMetadataBytes, nil
}

type TradingPair struct {
	TokenA                common.Address
	TokenB                common.Address
	TokenID1              uint32
	TokenID2              uint32
	OrderBookAddr         common.Address
	PositionDecimalsBase  uint8
	PositionDecimalsQuote uint8
}

type tokenInfo struct {
	address          string
	positionDecimals uint8
}

func (u *PoolsListUpdater) fetchTokenConfigs(ctx context.Context) (tokenMap map[uint32]tokenInfo, err error) {
	var raw []*big.Int
	req := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    compositeExchangeABI,
			Target: u.config.ExchangeAddress,
			Method: "bulkReadTokenConfigs_3423260018",
			Params: nil,
		}, []any{&raw})
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	tokenMap = make(map[uint32]tokenInfo)
	for i := 0; i+2 < len(raw); i += 3 {
		vault := raw[i]
		if vault == nil || vault.Sign() == 0 {
			break
		}
		tokenId, posDec, _, addr := UnpackVaultTokenConfig(vault)
		if addr != "" {
			tokenMap[tokenId] = tokenInfo{
				address:          strings.ToLower(addr),
				positionDecimals: posDec,
			}
		}
	}
	return tokenMap, nil
}

func (u *PoolsListUpdater) getTradingPairs(ctx context.Context, tokenMap map[uint32]tokenInfo) ([]TradingPair, error) {
	type orderBookRPC struct {
		OrderBook  common.Address `abi:"orderBook"`
		BuyTokenID uint32         `abi:"buyToken"`
		PayTokenID uint32         `abi:"payToken"`
	}

	type pairKey struct {
		TokenID1 uint32
		TokenID2 uint32
	}

	var keys []pairKey
	for tokenId1 := range tokenMap {
		for tokenId2 := range tokenMap {
			if tokenId1 == tokenId2 {
				continue
			}
			keys = append(keys, pairKey{TokenID1: tokenId1, TokenID2: tokenId2})
		}
	}
	if len(keys) == 0 {
		return nil, nil
	}

	obs := make([]orderBookRPC, len(keys))
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, k := range keys {
		req.AddCall(&ethrpc.Call{
			ABI:    compositeExchangeABI,
			Target: u.config.ExchangeAddress,
			Method: "getSpotOrderBook",
			Params: []any{k.TokenID1, k.TokenID2},
		}, []any{&obs[i]})
	}

	resp, err := req.TryAggregate()
	if err != nil {
		return nil, err
	}

	pairs := make([]TradingPair, 0, len(keys))
	for i, success := range resp.Result {
		ob := obs[i]

		if !success || ob.OrderBook.Cmp(common.Address{}) == 0 {
			continue
		}

		buyInfo, okBuy := tokenMap[ob.BuyTokenID]
		payInfo, okPay := tokenMap[ob.PayTokenID]
		if !okBuy || !okPay {
			continue
		}

		k := keys[i]
		pairs = append(pairs, TradingPair{
			TokenA:                common.HexToAddress(buyInfo.address),
			TokenB:                common.HexToAddress(payInfo.address),
			TokenID1:              k.TokenID1,
			TokenID2:              k.TokenID2,
			OrderBookAddr:         ob.OrderBook,
			PositionDecimalsBase:  buyInfo.positionDecimals,
			PositionDecimalsQuote: payInfo.positionDecimals,
		})
	}

	return pairs, nil
}

func (u *PoolsListUpdater) createPool(pair TradingPair) (entity.Pool, error) {
	poolTokens := []*entity.PoolToken{
		{
			Address:   strings.ToLower(pair.TokenA.Hex()),
			Swappable: true,
		},
		{
			Address:   strings.ToLower(pair.TokenB.Hex()),
			Swappable: true,
		},
	}

	staticExtra := StaticExtra{
		Router:                   u.config.RouterAddress,
		BuyTokenPositionDecimals: pair.PositionDecimalsBase,
		PayTokenPositionDecimals: pair.PositionDecimalsQuote,
	}

	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return entity.Pool{}, err
	}

	pool := entity.Pool{
		Address:     strings.ToLower(pair.OrderBookAddr.Hex()),
		Exchange:    u.config.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    []string{"0", "0"},
		Tokens:      poolTokens,
		StaticExtra: string(staticExtraBytes),
	}

	return pool, nil
}
