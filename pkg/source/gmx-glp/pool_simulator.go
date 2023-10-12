package gmxglp

import (
	"encoding/json"
	"fmt"
	"github.com/KyberNetwork/logger"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type Gas struct {
	Swap int64
}

type PoolSimulator struct {
	pool.Pool
	vault           *Vault
	vaultUtils      *VaultUtils
	glpManager      *GlpManager
	yearnTokenVault *YearnTokenVault
	gas             Gas
	swapInfo        *gmxGlpSwapInfo
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	tokens := make([]string, 0, len(entityPool.Tokens))
	for _, poolToken := range entityPool.Tokens {
		tokens = append(tokens, poolToken.Address)
	}

	info := pool.PoolInfo{
		Address:  entityPool.Address,
		Exchange: entityPool.Exchange,
		Type:     entityPool.Type,
		Tokens:   tokens,
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: info,
		},
		vault:      extra.Vault,
		vaultUtils: NewVaultUtils(extra.Vault),
		glpManager: extra.GlpManager,
		gas:        DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	var amountOut, feeAmount *big.Int
	var err error
	p.swapInfo = &gmxGlpSwapInfo{}

	if strings.EqualFold(tokenOut, p.glpManager.Glp) {
		amountOut, err = p.MintAndStakeGlp(tokenAmountIn.Token, tokenAmountIn.Amount)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
		amountOut, err = p.yearnTokenVault.Deposit(amountOut)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
		p.swapInfo.calcAmountOutType = calcAmountOutTypeStake
	} else if strings.EqualFold(tokenAmountIn.Token, p.glpManager.Glp) {
		amountOut, err = p.UnstakeAndRedeemGlp(tokenOut, tokenAmountIn.Amount)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
		p.swapInfo.calcAmountOutType = calcAmountOutTypeUnStake
	} else {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("pool gmx-glp %v only allows from/to glp token", p.Info.Address)
	}

	tokenAmountOut := &pool.TokenAmount{
		Token:  tokenOut,
		Amount: amountOut,
	}
	tokenAmountFee := &pool.TokenAmount{
		Token:  tokenOut,
		Amount: feeAmount,
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: tokenAmountOut,
		Fee:            tokenAmountFee,
		Gas:            p.gas.Swap,
		SwapInfo: gmxGlpSwapInfo{
			calcAmountOutType: p.swapInfo.calcAmountOutType,
			mintAmount:        p.swapInfo.mintAmount,
			amountAfterFees:   p.swapInfo.amountAfterFees,
			redemptionAmount:  p.swapInfo.redemptionAmount,
			usdgAmount:        p.swapInfo.usdgAmount,
		},
	}, nil
}

// UpdateBalance update UsdgAmount only
// https://github.com/gmx-io/gmx-contracts/blob/787d767e033c411f6d083f2725fb54b7fa956f7e/contracts/core/Vault.sol#L547-L548
func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(gmxGlpSwapInfo)
	if !ok {
		logger.Error("failed to UpdateBalancer for GmpGlp pool, wrong swapInfo type")
		return
	}

	switch swapInfo.calcAmountOutType {
	case calcAmountOutTypeStake:
		p.vault.IncreaseUSDGAmount(params.TokenAmountIn.Token, swapInfo.mintAmount)
		p.vault.IncreasePoolAmount(params.TokenAmountIn.Token, swapInfo.amountAfterFees)
	case calcAmountOutTypeUnStake:
		p.vault.DecreaseUSDGAmount(params.TokenAmountOut.Token, swapInfo.usdgAmount)
		p.vault.DecreasePoolAmount(params.TokenAmountOut.Token, swapInfo.redemptionAmount)
	}
}

// CanSwapFrom only allows glp swap to other tokens or other tokens to glp
func (p *PoolSimulator) CanSwapFrom(address string) []string {
	return p.CanSwapTo(address)
}

// CanSwapTo only allows glp swap to other tokens or other tokens to glp
func (p *PoolSimulator) CanSwapTo(address string) []string {
	if !strings.EqualFold(address, p.glpManager.Glp) {
		return []string{p.glpManager.Glp}
	}

	whitelistedTokens := p.vault.WhitelistedTokens
	swappableTokens := make([]string, 0, len(whitelistedTokens)-1)
	for _, token := range whitelistedTokens {
		tokenAddress := token

		if address == tokenAddress {
			continue
		}

		swappableTokens = append(swappableTokens, tokenAddress)
	}

	return swappableTokens
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} { return nil }

func (p *PoolSimulator) validateMaxUsdgExceed(token string, amount *big.Int) error {
	currentUsdgAmount := p.vault.USDGAmounts[token]
	newUsdgAmount := new(big.Int).Add(currentUsdgAmount, amount)

	maxUsdgAmount := p.vault.MaxUSDGAmounts[token]

	if maxUsdgAmount.Cmp(bignumber.ZeroBI) == 0 {
		return nil
	}

	if newUsdgAmount.Cmp(maxUsdgAmount) < 0 {
		return nil
	}

	return ErrVaultMaxUsdgExceeded
}

func (p *PoolSimulator) validateMinPoolAmount(token string, amount *big.Int) error {
	currentPoolAmount := p.vault.PoolAmounts[token]

	if currentPoolAmount.Cmp(amount) < 0 {
		return ErrVaultPoolAmountExceeded
	}

	newPoolAmount := new(big.Int).Sub(currentPoolAmount, amount)
	reservedAmount := p.vault.ReservedAmounts[token]

	if reservedAmount.Cmp(newPoolAmount) > 0 {
		return ErrVaultReserveExceedsPool
	}

	return nil
}

func (p *PoolSimulator) validateBufferAmount(token string, amount *big.Int) error {
	currentPoolAmount := p.vault.PoolAmounts[token]
	newPoolAmount := new(big.Int).Sub(currentPoolAmount, amount)

	bufferAmount := p.vault.BufferAmounts[token]

	if newPoolAmount.Cmp(bufferAmount) < 0 {
		return ErrVaultPoolAmountLessThanBufferAmount
	}

	return nil
}
