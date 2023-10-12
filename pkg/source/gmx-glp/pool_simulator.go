package gmxglp

import (
	"encoding/json"
	"fmt"
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

	vault      *Vault
	vaultUtils *VaultUtils

	glpManager *GlpManager

	gas Gas
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

	if strings.EqualFold(tokenOut, p.glpManager.Glp) {
		amountOut, err = p.MintAndStakeGlp(tokenAmountIn.Token, tokenAmountIn.Amount)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
	} else if strings.EqualFold(tokenAmountIn.Token, p.glpManager.Glp) {
		amountOut, err = p.UnstakeAndRedeemGlp(tokenOut, tokenAmountIn.Amount)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
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
	}, nil
}

// UpdateBalance update UsdgAmount only
// https://github.com/gmx-io/gmx-contracts/blob/787d767e033c411f6d083f2725fb54b7fa956f7e/contracts/core/Vault.sol#L547-L548
func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output, fee := params.TokenAmountIn, params.TokenAmountOut, params.Fee
	priceIn, err := p.vault.GetMinPrice(input.Token)
	if err != nil {
		return
	}

	usdgAmount := new(big.Int).Div(new(big.Int).Mul(input.Amount, priceIn), PricePrecision)
	usdgAmount = p.vault.AdjustForDecimals(usdgAmount, input.Token, p.vault.USDG.Address)

	p.vault.IncreaseUSDGAmount(input.Token, usdgAmount)
	p.vault.DecreaseUSDGAmount(output.Token, usdgAmount)
	p.vault.IncreasePoolAmount(input.Token, input.Amount)
	p.vault.DecreasePoolAmount(output.Token, new(big.Int).Add(output.Amount, fee.Amount))
}

// CanSwapFrom only allows to swap from address to glp
func (p *PoolSimulator) CanSwapFrom(address string) []string {
	if !strings.EqualFold(address, p.glpManager.Glp) {
		return []string{p.glpManager.Glp}
	}
	return p.CanSwapTo(address)
}

// CanSwapTo only allows glp swap to other tokens
func (p *PoolSimulator) CanSwapTo(address string) []string {
	if !strings.EqualFold(address, p.glpManager.Glp) {
		return nil
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
