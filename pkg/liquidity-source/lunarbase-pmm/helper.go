package lunarbase

import (
	"context"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type rpcState struct {
	blockNumber   uint64
	peripheryAddr string
	rawTokenX     string
	rawTokenY     string
	tokenX        string
	tokenY        string
	reserveX      *big.Int
	reserveY      *big.Int
	extra         Extra
	decimalsX     uint8
	decimalsY     uint8
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

func defaultPermit2(cfg *Config) string {
	if cfg != nil && cfg.Permit2Address != "" {
		return normalizeAddress(cfg.Permit2Address)
	}
	return normalizeAddress(defaultPermit2Address)
}

func wrappedNative(cfg *Config) string {
	chainID := cfg.ChainID
	if chainID == 0 {
		chainID = valueobject.ChainIDBase
	}

	return strings.ToLower(valueobject.WrappedNativeMap[chainID])
}

func fetchRPCState(
	ctx context.Context,
	cfg *Config,
	ethrpcClient *ethrpc.Client,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*rpcState, error) {
	coreAddress := defaultCore(cfg)
	peripheryAddress := defaultPeriphery(cfg)

	var (
		tokenX             common.Address
		tokenY             common.Address
		paused             bool
		blockDelay         uint64
		concentrationK     uint32
		concentrationAlpha uint8
		reserveX           *big.Int
		reserveY           *big.Int
		state              struct {
			PX96              *big.Int
			Fee               uint64
			LatestUpdateBlock uint64
		}
	)

	req := ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "X",
	}, []any{&tokenX})
	req.AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "Y",
	}, []any{&tokenY})
	req.AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "CONCENTRATION_ALPHA",
	}, []any{&concentrationAlpha})
	req.AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "blockDelay",
	}, []any{&blockDelay})
	req.AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "concentrationK",
	}, []any{&concentrationK})
	req.AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "getXReserve",
	}, []any{&reserveX})
	req.AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "getYReserve",
	}, []any{&reserveY})
	req.AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "paused",
	}, []any{&paused})
	req.AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "state",
	}, []any{&state})

	resp, err := req.Aggregate()
	if err != nil {
		return nil, err
	}
	blockNumber := resp.BlockNumber.Uint64()

	rawTokenX := strings.ToLower(tokenX.Hex())
	rawTokenY := strings.ToLower(tokenY.Hex())
	tokenXAddress := rawTokenX
	tokenYAddress := rawTokenY

	var decimalsX uint8 = 18
	var decimalsY uint8 = 18

	if tokenX == valueobject.AddrZero {
		tokenXAddress = wrappedNative(cfg)
	} else {
		req = ethrpcClient.NewRequest().SetContext(ctx)
		if overrides != nil {
			req.SetOverrides(overrides)
		}
		req.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: rawTokenX,
			Method: "decimals",
		}, []any{&decimalsX})
		if _, err = req.Aggregate(); err != nil {
			return nil, err
		}
	}

	if tokenY == valueobject.AddrZero {
		tokenYAddress = wrappedNative(cfg)
	} else {
		req = ethrpcClient.NewRequest().SetContext(ctx)
		if overrides != nil {
			req.SetOverrides(overrides)
		}
		req.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: rawTokenY,
			Method: "decimals",
		}, []any{&decimalsY})

		if _, err = req.Aggregate(); err != nil {
			return nil, err
		}
	}

	pX96 := uint256.NewInt(0)
	if state.PX96 != nil {
		pX96 = uint256.MustFromBig(state.PX96)
	}
	if reserveX == nil {
		reserveX = big.NewInt(0)
	}
	if reserveY == nil {
		reserveY = big.NewInt(0)
	}

	return &rpcState{
		blockNumber:   blockNumber,
		peripheryAddr: peripheryAddress,
		rawTokenX:     rawTokenX,
		rawTokenY:     rawTokenY,
		tokenX:        tokenXAddress,
		tokenY:        tokenYAddress,
		reserveX:      reserveX,
		reserveY:      reserveY,
		decimalsX:     decimalsX,
		decimalsY:     decimalsY,
		extra: Extra{
			PX96:               pX96,
			Fee:                state.Fee,
			LatestUpdateBlock:  state.LatestUpdateBlock,
			Paused:             paused,
			BlockDelay:         blockDelay,
			ConcentrationK:     concentrationK,
			ConcentrationAlpha: concentrationAlpha,
		},
	}, nil
}

func buildEntityPool(cfg *Config, state *rpcState) (entity.Pool, error) {
	extraBytes, err := json.Marshal(state.extra)
	if err != nil {
		return entity.Pool{}, err
	}

	staticExtraBytes, err := json.Marshal(StaticExtra{
		PeripheryAddress: state.peripheryAddr,
		Permit2Address:   defaultPermit2(cfg),
		RawTokenX:        state.rawTokenX,
		RawTokenY:        state.rawTokenY,
		WrappedNative:    wrappedNative(cfg),
	})
	if err != nil {
		return entity.Pool{}, err
	}

	return entity.Pool{
		Address:     defaultCore(cfg),
		Exchange:    cfg.DexID,
		Type:        DexType,
		BlockNumber: state.blockNumber,
		Timestamp:   0,
		Reserves: entity.PoolReserves{
			state.reserveX.String(),
			state.reserveY.String(),
		},
		Tokens: []*entity.PoolToken{
			{
				Address:   state.tokenX,
				Decimals:  state.decimalsX,
				Swappable: true,
			},
			{
				Address:   state.tokenY,
				Decimals:  state.decimalsY,
				Swappable: true,
			},
		},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
	}, nil
}
