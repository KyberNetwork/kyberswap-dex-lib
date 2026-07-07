package altfun

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

type EventParserConfig struct {
	BondingAddress       string `json:"bondingAddress"`
	ZapAddress           string `json:"zapAddress"`
	FactoryAddress       string `json:"factoryAddress"`
	GlobalStorageAddress string `json:"globalStorageAddress"`
}

type protocolParams struct {
	usdc                   string
	buyFeeBps              uint64
	sellFeeBps             uint64
	graduationThresholdUsd *uint256.Int
}

type EventParser struct {
	config       *EventParserConfig
	ethrpcClient *ethrpc.Client
	params       *protocolParams // fetched once at construction, immutable after
}

var _ = poolfactory.RegisterFactoryCE(DexType, NewPoolFactory)

func NewPoolFactory(cfg *EventParserConfig, ethrpcClient *ethrpc.Client) *EventParser {
	ep := &EventParser{config: cfg, ethrpcClient: ethrpcClient}
	// Fetch immutable protocol constants eagerly so DecodePoolCreated is fast.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if params, err := ep.fetchProtocolParams(ctx); err == nil {
		ep.params = params
	}
	return ep
}

// DecodePoolAddressesFromFactoryLog returns the pool address for state-changing events.
// topic[1] = token address for Trade / TokenGraduating / TokenGraduated.
func (p *EventParser) DecodePoolAddressesFromFactoryLog(_ context.Context, log types.Log) ([]string, error) {
	if !strings.EqualFold(log.Address.Hex(), p.config.BondingAddress) {
		return nil, nil
	}
	if len(log.Topics) < 2 {
		return nil, nil
	}
	switch log.Topics[0] {
	case bondingABI.Events["Trade"].ID,
		bondingABI.Events["TokenGraduating"].ID,
		bondingABI.Events["TokenGraduated"].ID:
		token := strings.ToLower(common.HexToAddress(log.Topics[1].Hex()).Hex())
		return []string{token}, nil
	}
	return nil, nil
}

// DecodePoolCreated handles the TokenLaunched event and creates a new pool entity.
//
// TokenLaunched topics:
//
//	[0] = event ID
//	[1] = token  (meme token address)
//	[2] = creator
//	[3] = ltAddress (BounceTech LT)
func (ep *EventParser) DecodePoolCreated(log types.Log) (*entity.Pool, error) {
	if !strings.EqualFold(log.Address.Hex(), ep.config.BondingAddress) {
		return nil, nil
	}
	if len(log.Topics) < 4 {
		return nil, nil
	}
	if log.Topics[0] != bondingABI.Events["TokenLaunched"].ID {
		return nil, nil
	}

	tokenAddr := strings.ToLower(common.HexToAddress(log.Topics[1].Hex()).Hex())
	ltAddr := strings.ToLower(common.HexToAddress(log.Topics[3].Hex()).Hex())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Fetch pairAddress for this specific token.
	pairAddr, err := ep.fetchPairAddress(ctx, tokenAddr)
	if err != nil {
		return nil, err
	}

	// Fetch protocol-level constants (cached after first call).
	params := ep.params
	if params == nil {
		var perr error
		if params, perr = ep.fetchProtocolParams(ctx); perr != nil {
			return nil, perr
		}
	}

	staticExtra := StaticExtra{
		PairAddress:            pairAddr,
		LTAddress:              ltAddr,
		USDC:                   params.usdc,
		ZapAddress:             ep.config.ZapAddress,
		BuyFeeBps:              params.buyFeeBps,
		SellFeeBps:             params.sellFeeBps,
		BasePool:               ltAddr,
		GraduationThresholdUsd: params.graduationThresholdUsd,
	}
	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return nil, err
	}

	return &entity.Pool{
		Address:   tokenAddr,
		Exchange:  DexType,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: params.usdc, Swappable: true},
			{Address: tokenAddr, Swappable: true},
		},
		StaticExtra: string(staticExtraBytes),
		Extra:       "{}",
		BlockNumber: log.BlockNumber,
	}, nil
}

// IsEventSupported returns true for TokenLaunched — enables real-time pool creation.
func (ep *EventParser) IsEventSupported(event common.Hash) bool {
	return event == bondingABI.Events["TokenLaunched"].ID
}

func (ep *EventParser) fetchPairAddress(ctx context.Context, tokenAddr string) (string, error) {
	var pairAddr common.Address
	req := ep.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: ep.config.FactoryAddress,
		Method: "pairFor",
		Params: []any{common.HexToAddress(tokenAddr)},
	}, []any{&pairAddr})
	if _, err := req.Aggregate(); err != nil {
		return "", err
	}
	return strings.ToLower(pairAddr.Hex()), nil
}

// fetchProtocolParams fetches immutable on-chain constants in one multicall.
func (ep *EventParser) fetchProtocolParams(ctx context.Context) (*protocolParams, error) {
	var (
		buyFee     = new(big.Int)
		sellFee    = new(big.Int)
		baseAsset  common.Address
		gradThresh = new(big.Int)
	)
	req := ep.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    zapABI,
		Target: ep.config.ZapAddress,
		Method: "buyFeeBps",
	}, []any{&buyFee}).
		AddCall(&ethrpc.Call{
			ABI:    zapABI,
			Target: ep.config.ZapAddress,
			Method: "sellFeeBps",
		}, []any{&sellFee}).
		AddCall(&ethrpc.Call{
			ABI:    globalStorageABI,
			Target: ep.config.GlobalStorageAddress,
			Method: "baseAsset",
		}, []any{&baseAsset}).
		AddCall(&ethrpc.Call{
			ABI:    bondingABI,
			Target: ep.config.BondingAddress,
			Method: "graduationThresholdUsd",
		}, []any{&gradThresh})

	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}
	return &protocolParams{
		usdc:                   strings.ToLower(baseAsset.Hex()),
		buyFeeBps:              buyFee.Uint64(),
		sellFeeBps:             sellFee.Uint64(),
		graduationThresholdUsd: uint256.MustFromBig(gradThresh),
	}, nil
}
