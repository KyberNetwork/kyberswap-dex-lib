package uniswapv4

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswapv3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/uniswapv3-sdk-uint256/constants"
	v3Entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	"github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	coreEntities "github.com/daoleno/uniswap-sdk-core/entities"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

var (
	defaultGas = uniswapv3.Gas{BaseGas: 85000, CrossInitTickGas: 24000}
)

type PoolSimulator struct {
	pool.Pool

	IsNative  [2]bool
	HooksData []byte
	PoolKey

	v3Simulator *uniswapv3.PoolSimulator
	staticExtra StaticExtra
}

func NewPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID) (*PoolSimulator, error) {
	var (
		extra       Extra
		staticExtra StaticExtra
	)
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, fmt.Errorf("unmarshal static extra: %w", err)
	}
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, fmt.Errorf("unmarshal extra: %w", err)
	}

	if len(entityPool.Tokens) != 2 {
		return nil, fmt.Errorf("invalid tokens count: %d, expect: 2", len(entityPool.Tokens))
	}
	if len(entityPool.Reserves) != 2 {
		return nil, fmt.Errorf("invalid reserves count: %d, expect: 2", len(entityPool.Reserves))
	}

	token0 := coreEntities.NewToken(
		uint(chainID),
		common.HexToAddress(entityPool.Tokens[0].Address),
		uint(entityPool.Tokens[0].Decimals),
		entityPool.Tokens[0].Symbol,
		entityPool.Tokens[0].Name,
	)
	token1 := coreEntities.NewToken(
		uint(chainID),
		common.HexToAddress(entityPool.Tokens[1].Address),
		uint(entityPool.Tokens[1].Decimals),
		entityPool.Tokens[1].Symbol,
		entityPool.Tokens[1].Name,
	)
	swapFee := big.NewInt(int64(entityPool.SwapFee))

	tokens := []string{
		entityPool.Tokens[0].Address,
		entityPool.Tokens[1].Address,
	}
	reserves := []*big.Int{
		NewBig10(entityPool.Reserves[0]),
		NewBig10(entityPool.Reserves[1]),
	}

	v3Ticks := make([]v3Entities.Tick, 0, len(extra.Ticks))
	for _, tick := range extra.Ticks {
		if tick.LiquidityGross.Sign() == 0 {
			continue
		}

		liqNet := new(utils.Int128)
		liqNet.SetFromBig(tick.LiquidityNet)
		v3Ticks = append(v3Ticks, v3Entities.Tick{
			Index:          tick.Index,
			LiquidityGross: new(uint256.Int).SetBytes(tick.LiquidityGross.Bytes()),
			LiquidityNet:   liqNet,
		})
	}
	if len(v3Ticks) == 0 {
		return nil, fmt.Errorf("empty tick")
	}

	tickSpacing := int(extra.TickSpacing)
	// For some pools that not yet initialized tickSpacing in their extra,
	// we will get the tickSpacing through feeTier mapping.
	if tickSpacing == 0 {
		feeTier := constants.FeeAmount(entityPool.SwapFee)
		if _, ok := constants.TickSpacings[feeTier]; !ok {
			return nil, fmt.Errorf("invalid fee tier")
		}
		tickSpacing = constants.TickSpacings[feeTier]
	}

	ticks, err := v3Entities.NewTickListDataProvider(v3Ticks, tickSpacing)
	if err != nil {
		return nil, err
	}

	sqrtPriceX96 := new(utils.Uint160)
	sqrtPriceX96.SetFromBig(extra.SqrtPriceX96)
	liq := new(utils.Uint128)
	liq.SetFromBig(extra.Liquidity)

	v3Pool, err := v3Entities.NewPoolV2(
		token0,
		token1,
		constants.FeeAmount(entityPool.SwapFee),
		sqrtPriceX96,
		liq,
		int(extra.Tick.Int64()),
		ticks,
	)
	if err != nil {
		return nil, err
	}

	tickMin := v3Ticks[0].Index
	tickMax := v3Ticks[len(v3Ticks)-1].Index

	var info = pool.PoolInfo{
		Address:    strings.ToLower(entityPool.Address),
		ReserveUsd: entityPool.ReserveUsd,
		SwapFee:    swapFee,
		Exchange:   entityPool.Exchange,
		Type:       entityPool.Type,
		Tokens:     tokens,
		Reserves:   reserves,
	}

	poolKey := PoolKey{
		Currency0:   common.HexToAddress(staticExtra.Currency0),
		Currency1:   common.HexToAddress(staticExtra.Currency1),
		Fee:         uint32(staticExtra.Fee),
		TickSpacing: int32(staticExtra.TickSpacing),
		Hooks:       staticExtra.HooksAddress,
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				SwapFee:    big.NewInt(int64(entityPool.SwapFee)),
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reserves,
			},
		},
		v3Simulator: uniswapv3.NewPoolSimulatorV2(v3Pool, pool.Pool{
			Info: info,
		}, defaultGas, tickMin, tickMax),
		staticExtra: staticExtra,
		IsNative: [2]bool{
			poolKey.Currency0 == NativeTokenPlaceholderAddress,
			poolKey.Currency1 == NativeTokenPlaceholderAddress,
		},
		HooksData: nil,
		PoolKey:   poolKey,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	return p.v3Simulator.CalcAmountOut(param)
}

func NewBig10(s string) (res *big.Int) {
	res, _ = new(big.Int).SetString(s, 10)
	return res
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p

	cloned.v3Simulator = p.v3Simulator.CloneState().(*uniswapv3.PoolSimulator)

	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	p.v3Simulator.UpdateBalance(params)
}

// GetMetaInfo
// adapt from https://github.com/KyberNetwork/kyberswap-dex-lib-private/blob/c1877a8c19759faeb7d82b6902ed335f0657ce3e/pkg/liquidity-source/uniswap-v4/pool_simulator.go#L201
func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	tokenInAddress := common.HexToAddress(tokenIn)
	tokenOutAddress := common.HexToAddress(tokenOut)

	if p.IsNative[0] || p.IsNative[1] {
		if tokenInAddress == p.Currency0 || tokenInAddress == p.Currency1 {
			tokenOutAddress = NativeTokenPlaceholderAddress
		} else {
			tokenInAddress = NativeTokenPlaceholderAddress
		}
	}

	return PoolMetaInfo{
		Router:      p.staticExtra.UniversalRouterAddress,
		Permit2Addr: p.staticExtra.Permit2Address,
		TokenIn:     tokenInAddress,
		TokenOut:    tokenOutAddress,
		Fee:         p.staticExtra.Fee,
		TickSpacing: p.staticExtra.TickSpacing,
		HookAddress: p.staticExtra.HooksAddress,
		HookData:    []byte{},
	}
}
