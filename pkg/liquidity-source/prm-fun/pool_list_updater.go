package prmfun

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, client *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: client}
}

// Version 2 replays discovery because the earlier ETH-only cursor skipped PRM
// and stock pairs permanently. Pool addresses provide stable deduplication keys.
func discoveryMetadata(data []byte) (PoolsListUpdaterMetadata, error) {
	var m PoolsListUpdaterMetadata
	if len(data) > 0 {
		if err := json.Unmarshal(data, &m); err != nil {
			return m, err
		}
	}
	if m.Offset < 0 {
		return m, ErrInvalidState
	}
	if m.Version != metadataVersion {
		m = PoolsListUpdaterMetadata{Version: metadataVersion}
	}
	return m, nil
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	metadata, err := discoveryMetadata(metadataBytes)
	if err != nil {
		return nil, metadataBytes, err
	}
	// All factory/curve getters used here are public and independent of msg.sender.
	// Strict aggregate makes an incomplete batch retry without advancing its cursor.
	// Robinhood is an Arbitrum chain: Multicall's block.number is the L1 parent,
	// not the L2 JSON-RPC block. Pin using eth_blockNumber and ignore the returned
	// Multicall block number when identifying this snapshot.
	head, err := u.ethrpcClient.GetBlockNumber(ctx)
	if err != nil {
		return nil, metadataBytes, err
	}
	if head == 0 {
		return nil, metadataBytes, ErrInvalidState
	}
	block := new(big.Int).SetUint64(head)
	var total *big.Int
	_, err = u.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(block).AddCall(&ethrpc.Call{
		ABI: memeFactoryABI, Target: u.config.FactoryAddress, Method: memeFactoryMethodMemeCount,
	}, []any{&total}).Aggregate()
	if err != nil {
		return nil, metadataBytes, err
	}
	if total == nil || !total.IsInt64() || total.Sign() < 0 {
		return nil, metadataBytes, ErrInvalidState
	}
	limit := u.config.NewPoolLimit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	start := metadata.Offset
	if int64(start) > total.Int64() {
		return nil, metadataBytes, ErrInvalidState
	}
	end := start + min(limit, int(total.Int64())-start)
	if start == end {
		data, err := json.Marshal(metadata)
		return nil, data, err
	}
	tokens := make([]common.Address, end-start)
	req := u.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(block)
	for i := range tokens {
		req.AddCall(&ethrpc.Call{ABI: memeFactoryABI, Target: u.config.FactoryAddress,
			Method: memeFactoryMethodMemeTokens, Params: []any{big.NewInt(int64(start + i))}}, []any{&tokens[i]})
	}
	if _, err = req.Aggregate(); err != nil {
		return nil, metadataBytes, err
	}
	markets := make([]GetMemeResult, len(tokens))
	req = u.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(block)
	for i, token := range tokens {
		req.AddCall(&ethrpc.Call{ABI: memeFactoryABI, Target: u.config.FactoryAddress,
			Method: memeFactoryMethodGetMeme, Params: []any{token}}, []any{&markets[i]})
	}
	if _, err = req.Aggregate(); err != nil {
		return nil, metadataBytes, err
	}
	pairs := make([]common.Address, len(tokens))
	native := make([]bool, len(tokens))
	phases := make([]uint8, len(tokens))
	req = u.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(block)
	for i, m := range markets {
		if m.Data.Token != tokens[i] || m.Data.Curve == (common.Address{}) || m.Data.Token == (common.Address{}) || m.Data.GraduationDesk == nil || m.Data.GraduationDesk.Sign() <= 0 {
			return nil, metadataBytes, ErrInvalidState
		}
		target := hexutil.Encode(m.Data.Curve[:])
		req.AddCall(&ethrpc.Call{ABI: memeCurveABI, Target: target, Method: "deskToken"}, []any{&pairs[i]}).
			AddCall(&ethrpc.Call{ABI: memeCurveABI, Target: target, Method: "isNativeQuote"}, []any{&native[i]}).
			AddCall(&ethrpc.Call{ABI: memeCurveABI, Target: target, Method: memeCurveMethodPhase}, []any{&phases[i]})
	}
	if _, err = req.Aggregate(); err != nil {
		return nil, metadataBytes, err
	}
	pools := make([]entity.Pool, 0, len(tokens))
	for i, m := range markets {
		p, err := discoveredPool(u.config, m, pairs[i], native[i], phases[i])
		if err != nil {
			return nil, metadataBytes, err
		}
		if p != nil {
			pools = append(pools, *p)
		}
	}
	data, err := json.Marshal(PoolsListUpdaterMetadata{Offset: end, Version: metadataVersion})
	return pools, data, err
}

func discoveredPool(cfg *Config, market GetMemeResult, pair common.Address, native bool, phase uint8) (*entity.Pool, error) {
	m := market.Data
	if phase > PhasePaused || pair == (common.Address{}) || pair == m.Token || native != (m.SubjectId == [32]byte{}) {
		return nil, ErrInvalidState
	}
	pairAddress := hexutil.Encode(pair[:])
	if native && !valueobject.IsWrappedNative(pairAddress, cfg.ChainId) {
		return nil, ErrInvalidToken
	}
	// Paused curves stay discoverable so their ordinary tracker can restore quotes
	// after unpause. Only graduation is terminal.
	if phase == PhaseGraduated {
		return nil, nil
	}
	curveAddress, tokenAddress := hexutil.Encode(m.Curve[:]), hexutil.Encode(m.Token[:])
	static, err := json.Marshal(StaticExtra{
		RouterAddress: hexutil.Encode(common.HexToAddress(cfg.RouterAddress).Bytes()),
		CurveAddress:  curveAddress, MemeToken: tokenAddress,
		GraduationDesk: m.GraduationDesk.String(), IsNativeQuote: native,
	})
	if err != nil {
		return nil, err
	}
	return &entity.Pool{
		Address: curveAddress, Exchange: cfg.DexId, Type: DexType,
		Timestamp: time.Now().Unix(), Reserves: []string{"0", "0"}, StaticExtra: string(static),
		Tokens: []*entity.PoolToken{{Address: pairAddress, Swappable: true}, {Address: tokenAddress, Swappable: true}},
	}, nil
}
