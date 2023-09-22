package traderjoev20

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	aevmclient "github.com/KyberNetwork/aevm/client"
	aevmcommon "github.com/KyberNetwork/aevm/common"
	aevmtypes "github.com/KyberNetwork/aevm/types"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	aevmcore "github.com/KyberNetwork/router-service/internal/pkg/core/aevm"
	routerentity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/common"
)

type Pool struct {
	pool.Pool
	aevmPool *aevmcore.AEVMPool
}

func NewPoolAEVM(
	entityPool entity.Pool,
	aevmClient aevmclient.Client,
	stateRoot gethcommon.Hash,
	tokenBalanceSlots map[gethcommon.Address]*routerentity.ERC20BalanceSlot,
) (*Pool, error) {
	if len(entityPool.Tokens) != 2 {
		return nil, errors.New("TraderJoe pool must have 2 tokens")
	}
	tokens := []string{
		entityPool.Tokens[0].Address,
		entityPool.Tokens[1].Address,
	}
	return &Pool{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  strings.ToLower(entityPool.Address),
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   tokens,
				Checked:  true,
			},
		},
		aevmPool: &aevmcore.AEVMPool{
			AEVMClient:        common.MakeNoClone(aevmClient),
			StateRoot:         stateRoot,
			TokenBalanceSlots: common.MakeNoClone(tokenBalanceSlots),
		},
	}, nil
}

func (p *Pool) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	return p.CalcAmountOutAEVM(tokenAmountIn, tokenOut)
}

func (p *Pool) CalcAmountOutAEVM(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	var (
		poolTokenY   = gethcommon.HexToAddress(p.Info.Tokens[1])
		tokenInAddr  = gethcommon.HexToAddress(tokenAmountIn.Token)
		tokenOutAddr = gethcommon.HexToAddress(tokenOut)
		swapForY     = tokenOutAddr == poolTokenY
	)
	blIn, ok := p.aevmPool.TokenBalanceSlots.Get().(routerentity.TokenBalanceSlots)[tokenInAddr]
	if !ok {
		return nil, fmt.Errorf("expected token balance slot for token %s", tokenAmountIn.Token)
	}
	wallet := gethcommon.HexToAddress(blIn.Wallet)
	swapCalls, err := p.swapCalls(
		tokenAmountIn.Amount,
		tokenInAddr,
		tokenOutAddr,
		wallet,
	)
	if err != nil {
		return nil, err
	}
	amountOutIndex := 0
	if swapForY {
		amountOutIndex = 1
	}
	strategy := &aevmcore.AEVMSwapStrategy{
		SwapCalls:       swapCalls,
		AmountOutGetter: aevmcore.AmountOutGetterSwapOutputTuple,
		AmountOutGetterArgs: aevmcore.AmountOutGetterSwapOutputTupleArgs{
			ElementIndex: amountOutIndex,
		},
	}
	return aevmcore.CalcAmountOutAEVM(p.aevmPool, strategy, tokenAmountIn.Amount, tokenInAddr, tokenOutAddr)
}

func (p *Pool) pairSwap(tokenOut, wallet gethcommon.Address) ([]byte, error) {
	poolTokenY := gethcommon.HexToAddress(p.Info.Tokens[1])
	swapForY := tokenOut == poolTokenY
	return pairABI.Pack("swap", swapForY, wallet)
}

func (p *Pool) swapCalls(amountIn *big.Int, tokenIn, tokenOut, wallet gethcommon.Address) (*aevmcore.AEVMSwapCalls, error) {
	poolAddress := gethcommon.HexToAddress(p.Info.Address)
	transferInput, err := abis.ERC20.Pack("transfer", poolAddress, amountIn)
	if err != nil {
		return nil, fmt.Errorf("could not build transfer call: %w", err)
	}
	transferCall := aevmtypes.SingleCall{
		From:  aevmcommon.Address(wallet),
		To:    aevmcommon.Address(tokenIn),
		Value: uint256.NewInt(0),
		Data:  transferInput,
	}
	swapInput, err := p.pairSwap(tokenOut, wallet)
	if err != nil {
		return nil, fmt.Errorf("could not build swap call: %w", err)
	}
	swapCall := aevmtypes.SingleCall{
		From:  aevmcommon.Address(wallet),
		To:    aevmcommon.Address(poolAddress),
		Value: uint256.NewInt(0),
		Data:  swapInput,
		Options: &aevmtypes.SingleCallOptions{
			ReturnStateAfter: true,
		},
	}
	return &aevmcore.AEVMSwapCalls{
		PreCalls: []aevmtypes.SingleCall{
			transferCall,
		},
		SwapCall: swapCall,
	}, nil
}

func (p *Pool) UpdateBalance(params pool.UpdateBalanceParams) {
	if si, ok := params.SwapInfo.(*aevmcore.AEVMSwapInfo); ok {
		p.aevmPool.NextSwapInfo = si
	}
}

func (p *Pool) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return nil
}
