package uni

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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswap"
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
	aevmPool      *aevmcore.AEVMPool
}

func (t *Pool) UpdateBalance(params pool.UpdateBalanceParams) {
	t.aevmPool.UpdateBalance(params)
}

func NewPoolAEVM(
	entityPool entity.Pool,
	routerAddress gethcommon.Address,
	client aevmclient.Client,
	stateRoot gethcommon.Hash,
	tokenBalanceSlots map[gethcommon.Address]*routerentity.ERC20BalanceSlot,
) (*Pool, error) {
	if len(entityPool.Reserves) != 2 || len(entityPool.Tokens) != 2 {
		return nil, fmt.Errorf("reserves and tokens length must be 2")
	}

	swapFeeFl := new(big.Float).Mul(big.NewFloat(entityPool.SwapFee), bOneFloat)
	swapFee, _ := swapFeeFl.Int(nil)

	tokens := []string{entityPool.Tokens[0].Address, entityPool.Tokens[1].Address}
	reserves := []*big.Int{utils.NewBig10(entityPool.Reserves[0]), utils.NewBig10(entityPool.Reserves[1])}

	return &Pool{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				SwapFee:    swapFee,
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reserves,
				Checked:    false,
			},
		},
		routerAddress: routerAddress,
		aevmPool: &aevmcore.AEVMPool{
			AEVMClient:        common.MakeNoClone(client),
			StateRoot:         stateRoot,
			TokenBalanceSlots: common.MakeNoClone(tokenBalanceSlots),
		},
	}, nil
}

func (t *Pool) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var (
		tokenInAddr  = gethcommon.HexToAddress(params.TokenAmountIn.Token)
		tokenOutAddr = gethcommon.HexToAddress(params.TokenOut)
	)
	blIn, ok := t.aevmPool.TokenBalanceSlots.Get().(routerentity.TokenBalanceSlots)[tokenInAddr]
	if !ok {
		return nil, fmt.Errorf("expected token balance slot for token %s", params.TokenAmountIn.Token)
	}
	wallet := gethcommon.HexToAddress(blIn.Wallet)
	swapCalls, err := t.swapCalls(
		params.TokenAmountIn.Amount,
		tokenInAddr,
		tokenOutAddr,
		wallet,
	)
	if err != nil {
		return nil, err
	}
	strategy := &aevmcore.AEVMSwapStrategy{
		SwapCalls:       swapCalls,
		AmountOutGetter: aevmcore.AmountOutGetterDelta,
		AmountOutGetterArgs: aevmcore.AmountOutGetterDeltaArgs{
			BalanceOfBeforeIndex: 0,
			BalanceOfAfterIndex:  4,
		},
	}
	return aevmcore.CalcAmountOutAEVM(t.aevmPool, strategy, params.TokenAmountIn.Amount, tokenInAddr, tokenOutAddr)
}

// build UniswapV2Router02.swapExactTokensForTokensSupportingFeeOnTransferTokens input
func (t *Pool) routerSwapExactTokensForTokensSupportingFeeOnTransferTokens(
	amountIn *big.Int, tokenIn, tokenOut, wallet gethcommon.Address,
) ([]byte, error) {
	return UniswapV2Router02ABI.Pack(
		"swapExactTokensForTokensSupportingFeeOnTransferTokens",
		amountIn,
		big.NewInt(0),
		[]gethcommon.Address{tokenIn, tokenOut},
		wallet,
		new(big.Int).SetUint64(math.MaxUint64),
	)
}

func (t *Pool) swapCalls(amountIn *big.Int, tokenIn, tokenOut, wallet gethcommon.Address) (*aevmcore.AEVMSwapCalls, error) {
	balanceOfInput, err := abis.ERC20.Pack("balanceOf", wallet)
	if err != nil {
		return nil, fmt.Errorf("could not build balanceOf call: %w", err)
	}
	balanceOfCall := aevmtypes.SingleCall{
		From:  aevmcommon.Address(wallet),
		To:    aevmcommon.Address(tokenOut),
		Value: (*aevmcommon.Uint256)(uint256.NewInt(0)),
		Data:  balanceOfInput,
	}
	// Some tokens requires allowance to be 0 before we set it to another value
	// https://github.com/Giveth/minime/blob/master/contracts/MiniMeToken.sol#L221-L225
	approveZeroInput, err := abis.ERC20.Pack("approve", t.routerAddress, big.NewInt(0))
	if err != nil {
		return nil, fmt.Errorf("could not build approve call: %w", err)
	}
	approveZeroCall := aevmtypes.SingleCall{
		From:  aevmcommon.Address(wallet),
		To:    aevmcommon.Address(tokenIn),
		Value: (*aevmcommon.Uint256)(uint256.NewInt(0)),
		Data:  approveZeroInput,
	}
	approveInput, err := abis.ERC20.Pack("approve", t.routerAddress, amountIn)
	if err != nil {
		return nil, fmt.Errorf("could not build approve call: %w", err)
	}
	approveCall := aevmtypes.SingleCall{
		From:  aevmcommon.Address(wallet),
		To:    aevmcommon.Address(tokenIn),
		Value: (*aevmcommon.Uint256)(uint256.NewInt(0)),
		Data:  approveInput,
	}
	swapInput, err := t.routerSwapExactTokensForTokensSupportingFeeOnTransferTokens(
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
		To:    aevmcommon.Address(t.routerAddress),
		Value: (*aevmcommon.Uint256)(uint256.NewInt(0)),
		Data:  swapInput,
		Options: &aevmtypes.SingleCallOptions{
			ReturnStateAfter: true,
		},
	}
	return &aevmcore.AEVMSwapCalls{
		PreCalls: []aevmtypes.SingleCall{
			balanceOfCall,
			approveZeroCall,
			approveCall,
		},
		SwapCall: swapCall,
		PostCalls: []aevmtypes.SingleCall{
			balanceOfCall,
		},
	}, nil
}

func (t *Pool) GetMetaInfo(_ string, _ string) interface{} {
	if t.GetInfo().SwapFee == nil {
		return uniswap.Meta{
			SwapFee: defaultSwapFee,
		}
	}

	return uniswap.Meta{
		SwapFee: t.GetInfo().SwapFee.String(),
	}
}
