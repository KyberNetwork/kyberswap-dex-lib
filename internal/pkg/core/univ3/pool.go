package univ3

import (
	"fmt"
	"math"
	"math/big"
	"strings"

	aevmclient "github.com/KyberNetwork/aevm/client"
	aevmcommon "github.com/KyberNetwork/aevm/common"
	aevmtypes "github.com/KyberNetwork/aevm/types"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	coreEntities "github.com/daoleno/uniswap-sdk-core/entities"
	"github.com/daoleno/uniswapv3-sdk/constants"
	univ3utils "github.com/daoleno/uniswapv3-sdk/utils"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	aevmcore "github.com/KyberNetwork/router-service/internal/pkg/core/aevm"
	routerentity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/common"
)

type Pool struct {
	pool.Pool
	routerAddress gethcommon.Address
	chainID       uint
	aevmPool      *aevmcore.AEVMPool
}

func (p *Pool) UpdateBalance(params pool.UpdateBalanceParams) {
	if si, ok := params.SwapInfo.(*aevmcore.AEVMSwapInfo); ok {
		p.aevmPool.NextSwapInfo = si
		return
	}
}

func NewPoolAEVM(
	entityPool entity.Pool,
	routerAddress gethcommon.Address,
	chainID valueobject.ChainID,
	aevmClient aevmclient.Client,
	stateRoot gethcommon.Hash,
	tokenBalanceSlots map[gethcommon.Address]*routerentity.ERC20BalanceSlot,
) (*Pool, error) {
	if len(entityPool.Reserves) != 2 || len(entityPool.Tokens) != 2 {
		return nil, fmt.Errorf("reserves and tokens length must be 2")
	}

	tokens := []string{
		entityPool.Tokens[0].Address,
		entityPool.Tokens[1].Address,
	}
	reserves := []*big.Int{
		utils.NewBig10(entityPool.Reserves[0]),
		utils.NewBig10(entityPool.Reserves[1]),
	}

	return &Pool{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				SwapFee:    new(big.Int).SetUint64(uint64(entityPool.SwapFee)),
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reserves,
				Checked:    false,
			},
		},
		routerAddress: routerAddress,
		chainID:       uint(chainID),
		aevmPool: &aevmcore.AEVMPool{
			AEVMClient:        common.MakeNoClone(aevmClient),
			StateRoot:         stateRoot,
			TokenBalanceSlots: common.MakeNoClone(tokenBalanceSlots),
		},
	}, nil
}

func (p *Pool) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var (
		tokenInAddr  = gethcommon.HexToAddress(params.TokenAmountIn.Token)
		tokenOutAddr = gethcommon.HexToAddress(params.TokenOut)
	)
	blIn, ok := p.aevmPool.TokenBalanceSlots.Get().(routerentity.TokenBalanceSlots)[tokenInAddr]
	if !ok {
		return nil, fmt.Errorf("expected token balance slot for token %s", params.TokenAmountIn.Token)
	}
	wallet := gethcommon.HexToAddress(blIn.Wallet)
	swapCalls, err := p.swapCalls(
		params.TokenAmountIn.Amount,
		tokenInAddr,
		tokenOutAddr,
		wallet,
	)
	if err != nil {
		return nil, err
	}
	strategy := &aevmcore.AEVMSwapStrategy{
		Precheck: func() error {
			if err := p.checkAddress(params.TokenAmountIn.Token, params.TokenOut); err != nil {
				return fmt.Errorf("invalid pool address: %w", err)
			}
			return nil
		},
		SwapCalls:       swapCalls,
		AmountOutGetter: aevmcore.AmountOutGetterSwapOutput,
	}
	return aevmcore.CalcAmountOutAEVM(p.aevmPool, strategy, params.TokenAmountIn.Amount, tokenInAddr, tokenOutAddr)
}

func (p *Pool) checkAddress(tokenIn, tokenOut string) error {
	computedAddr, err := univ3utils.ComputePoolAddress(
		constants.FactoryAddress,
		coreEntities.NewToken(p.chainID, gethcommon.HexToAddress(tokenIn), 0, "", ""),
		coreEntities.NewToken(p.chainID, gethcommon.HexToAddress(tokenOut), 0, "", ""),
		constants.FeeAmount(p.Pool.Info.SwapFee.Uint64()),
		"",
	)
	if err != nil {
		return err
	}
	if !strings.EqualFold(computedAddr.Hex(), p.Pool.Info.Address) {
		return fmt.Errorf("expected pool address to be %s", computedAddr)
	}
	return nil
}

// build SwapRouter.exactInputSingle input
func (p *Pool) routerExactInputSingle(amountIn *big.Int, tokenIn, tokenOut, wallet gethcommon.Address) ([]byte, error) {
	return UniswapV3SwapRouterABI.Pack("exactInputSingle", &ExactInputSingleParams{
		TokenIn:           tokenIn,
		TokenOut:          tokenOut,
		Fee:               new(big.Int).Set(p.Pool.Info.SwapFee),
		Recipient:         wallet,
		Deadline:          new(big.Int).SetUint64(math.MaxUint64),
		AmountIn:          amountIn,
		AmountOutMinimum:  big.NewInt(0),
		SqrtPriceLimitX96: big.NewInt(0),
	})
}

func (p *Pool) swapCalls(amountIn *big.Int, tokenIn, tokenOut, wallet gethcommon.Address) (*aevmcore.AEVMSwapCalls, error) {
	// Some tokens requires allowance to be 0 before we set it to another value
	// https://github.com/Giveth/minime/blob/master/contracts/MiniMeToken.sol#L221-L225
	approveZeroInput, err := abis.ERC20.Pack("approve", p.routerAddress, big.NewInt(0))
	if err != nil {
		return nil, fmt.Errorf("could not build approve call: %w", err)
	}
	approveZeroCall := aevmtypes.SingleCall{
		From:  aevmcommon.Address(wallet),
		To:    aevmcommon.Address(tokenIn),
		Value: uint256.NewInt(0),
		Data:  approveZeroInput,
	}
	approveInput, err := abis.ERC20.Pack("approve", p.routerAddress, amountIn)
	if err != nil {
		return nil, fmt.Errorf("could not build approve call: %w", err)
	}
	approveCall := aevmtypes.SingleCall{
		From:  aevmcommon.Address(wallet),
		To:    aevmcommon.Address(tokenIn),
		Value: uint256.NewInt(0),
		Data:  approveInput,
	}
	swapInput, err := p.routerExactInputSingle(
		amountIn,
		tokenIn,
		tokenOut,
		wallet,
	)
	if err != nil {
		return nil, fmt.Errorf("could not build swap call: %w", err)
	}
	swapCall := aevmtypes.SingleCall{
		From:  aevmcommon.Address(wallet),
		To:    aevmcommon.Address(p.routerAddress),
		Value: uint256.NewInt(0),
		Data:  swapInput,
		Options: &aevmtypes.SingleCallOptions{
			ReturnStateAfter: true,
		},
	}
	return &aevmcore.AEVMSwapCalls{
		PreCalls: []aevmtypes.SingleCall{
			approveZeroCall,
			approveCall,
		},
		SwapCall: swapCall,
	}, nil
}

func (p *Pool) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return nil
}
