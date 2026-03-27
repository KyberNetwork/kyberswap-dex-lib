package lunarbase

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type rpcState struct {
	blockNumber   uint64
	peripheryAddr string
	hasNative     bool
	tokenX        string
	tokenY        string
	reserveX      *big.Int
	reserveY      *big.Int
	extra         Extra
}

func normalizeAddress(address string) string {
	return strings.ToLower(address)
}

func defaultCore(cfg *Config) string {
	if cfg != nil && cfg.CoreAddress != "" {
		return normalizeAddress(cfg.CoreAddress)
	}
	return normalizeAddress(defaultCoreAddress)
}

func defaultPeriphery(cfg *Config) string {
	if cfg != nil && cfg.PeripheryAddress != "" {
		return normalizeAddress(cfg.PeripheryAddress)
	}
	return normalizeAddress(defaultPeripheryAddress)
}

func fetchRPCState(ctx context.Context, p *entity.Pool, cfg *Config, ethrpcClient *ethrpc.Client,
	overrides map[common.Address]gethclient.OverrideAccount) (*rpcState, error) {
	coreAddress := defaultCore(cfg)
	peripheryAddress := defaultPeriphery(cfg)
	if p != nil {
		coreAddress = p.Address
	}

	var (
		tokenX         common.Address
		tokenY         common.Address
		paused         bool
		blockDelay     uint64
		concentrationK uint32
		reserveX       *big.Int
		reserveY       *big.Int
		state          struct {
			PX96              *big.Int
			Fee               uint64
			LatestUpdateBlock uint64
		}
	)

	resp, err := ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "X",
	}, []any{&tokenX}).AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "Y",
	}, []any{&tokenY}).AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "blockDelay",
	}, []any{&blockDelay}).AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "concentrationK",
	}, []any{&concentrationK}).AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "getXReserve",
	}, []any{&reserveX}).AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "getYReserve",
	}, []any{&reserveY}).AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "paused",
	}, []any{&paused}).AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "state",
	}, []any{&state}).Aggregate()
	if err != nil {
		return nil, err
	}
	blockNumber := resp.BlockNumber.Uint64()

	tokenXAddress := valueobject.WrapNativeZeroLower(hexutil.Encode(tokenX[:]), cfg.ChainID)
	tokenYAddress := valueobject.WrapNativeZeroLower(hexutil.Encode(tokenY[:]), cfg.ChainID)

	pX96 := lo.CoalesceOrEmpty(uint256.MustFromBig(state.PX96), big256.U0)
	if reserveX == nil {
		reserveX = bignumber.ZeroBI
	}
	if reserveY == nil {
		reserveY = bignumber.ZeroBI
	}

	return &rpcState{
		blockNumber:   blockNumber,
		peripheryAddr: peripheryAddress,
		hasNative:     valueobject.IsNativeOrZeroAddr(tokenX) || valueobject.IsNativeOrZeroAddr(tokenY),
		tokenX:        tokenXAddress,
		tokenY:        tokenYAddress,
		reserveX:      reserveX,
		reserveY:      reserveY,
		extra: Extra{
			PriceX96:          pX96,
			FeeQ48:            state.Fee,
			LatestUpdateBlock: state.LatestUpdateBlock,
			Paused:            paused,
			BlockDelay:        blockDelay,
			ConcentrationK:    concentrationK,
		},
	}, nil
}

func buildEntityPool(p *entity.Pool, cfg *Config, state *rpcState) (*entity.Pool, error) {
	if p == nil {
		staticExtraBytes, _ := json.Marshal(StaticExtra{
			PeripheryAddress: state.peripheryAddr,
			HasNative:        state.hasNative,
		})
		p = &entity.Pool{
			Address:  defaultCore(cfg),
			Exchange: cfg.DexID,
			Type:     DexType,
			Tokens: []*entity.PoolToken{
				{Address: state.tokenX, Swappable: true},
				{Address: state.tokenY, Swappable: true},
			},
			StaticExtra: string(staticExtraBytes),
		}
	}

	extraBytes, err := json.Marshal(state.extra)
	if err != nil {
		return nil, err
	}

	p.SwapFee = float64(state.extra.FeeQ48) / fQ48
	p.BlockNumber = state.blockNumber
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{state.reserveX.String(), state.reserveY.String()}
	p.Extra = string(extraBytes)
	return p, nil
}
