package kuruob

import (
	"math"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	*orderbook.PoolSimulator
	decimals   [2]uint8
	precisions [2]int
	hasNative  bool
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	poolSim, err := orderbook.NewPoolSimulatorWith(entityPool, math.MaxInt64)
	if err != nil {
		return nil, err
	}
	poolSim.Gas = defaultGas
	var staticExtra StaticExtra
	_ = json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra)
	return &PoolSimulator{PoolSimulator: poolSim,
		decimals:   [2]uint8{entityPool.Tokens[0].Decimals, entityPool.Tokens[1].Decimals},
		precisions: [2]int{staticExtra.SizePrecision, staticExtra.PricePrecision},
		hasNative:  staticExtra.HasNative}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.PoolSimulator = p.PoolSimulator.CloneState().(*orderbook.PoolSimulator)
	return &cloned
}

func (p *PoolSimulator) GetMetaInfo(tokenIn, _ string) any {
	idxIn := p.GetTokenIndex(tokenIn)
	return MetaInfo{
		Decimals:  p.decimals[idxIn],
		Precision: p.precisions[idxIn],
		IdxIn:     idxIn,
		HasNative: p.hasNative,
	}
}

func (s *PoolSimulator) SwapReceiveNativeIn(tokenIn, tokenOut string, chainId valueobject.ChainID) bool {
	meta := s.GetMetaInfo(tokenIn, tokenOut).(MetaInfo)
	return meta.HasNative && valueobject.IsWrappedNative(tokenIn, chainId)
}

func (s *PoolSimulator) SwapReturnNativeOut(tokenIn, tokenOut string, chainId valueobject.ChainID) bool {
	meta := s.GetMetaInfo(tokenIn, tokenOut).(MetaInfo)
	return meta.HasNative && valueobject.IsWrappedNative(tokenOut, chainId)
}
