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
	blockNumber uint64
	hasNative   bool
	tokenX      string
	tokenY      string
	reserveX    *big.Int
	reserveY    *big.Int
	extra       Extra
}

func fetchRPCState(ctx context.Context, p *entity.Pool, cfg *Config, ethrpcClient *ethrpc.Client,
	overrides map[common.Address]gethclient.OverrideAccount) (*rpcState, error) {
	var coreAddress string
	if p != nil {
		coreAddress = p.Address
	} else {
		coreAddress = strings.ToLower(cfg.CoreAddress)
	}

	var (
		tokenX         common.Address
		tokenY         common.Address
		paused         bool
		blockDelay     uint64
		concentrationK uint32
		anchorPrice    *big.Int
		reserveX       *big.Int
		reserveY       *big.Int
		state          struct {
			AnchorPrice       *big.Int
			FeeAskX24         uint32
			FeeBidX24         uint32
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
	}, []any{&state}).AddCall(&ethrpc.Call{
		ABI:    coreABI,
		Target: coreAddress,
		Method: "anchorPrice",
	}, []any{&anchorPrice}).Aggregate()
	if err != nil {
		return nil, err
	}
	blockNumber := resp.BlockNumber.Uint64()

	tokenXAddress := valueobject.WrapNativeZeroLower(hexutil.Encode(tokenX[:]), cfg.ChainID)
	tokenYAddress := valueobject.WrapNativeZeroLower(hexutil.Encode(tokenY[:]), cfg.ChainID)

	// Prefer the `state()` tuple as the authoritative source; fall back to the
	// dedicated `anchorPrice()` view if the tuple decode produced nil.
	sqrtPriceBig := state.AnchorPrice
	if sqrtPriceBig == nil {
		sqrtPriceBig = anchorPrice
	}
	sqrtPriceX96 := lo.CoalesceOrEmpty(uint256.MustFromBig(sqrtPriceBig), big256.U0)
	if reserveX == nil {
		reserveX = bignumber.ZeroBI
	}
	if reserveY == nil {
		reserveY = bignumber.ZeroBI
	}

	return &rpcState{
		blockNumber: blockNumber,
		hasNative:   valueobject.IsNativeOrZeroAddr(tokenX) || valueobject.IsNativeOrZeroAddr(tokenY),
		tokenX:      tokenXAddress,
		tokenY:      tokenYAddress,
		reserveX:    reserveX,
		reserveY:    reserveY,
		extra: Extra{
			SqrtPriceX96:      sqrtPriceX96,
			FeeAskX24:         state.FeeAskX24,
			FeeBidX24:         state.FeeBidX24,
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
			HasNative: state.hasNative,
		})
		p = &entity.Pool{
			Address:  strings.ToLower(cfg.CoreAddress),
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

	// Report the larger of the two directional fees as the entity's nominal
	// SwapFee — routers use it for coarse cost estimation; the simulator
	// applies the precise per-direction value at quote time.
	maxFee := state.extra.FeeAskX24
	if state.extra.FeeBidX24 > maxFee {
		maxFee = state.extra.FeeBidX24
	}
	p.SwapFee = float64(maxFee) / fQ24
	p.BlockNumber = state.blockNumber
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{state.reserveX.String(), state.reserveY.String()}
	p.Extra = string(extraBytes)
	return p, nil
}
