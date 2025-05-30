package swapbasedperp

import (
	"maps"
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/goccy/go-json"

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
	gas        Gas
}

var _ = pool.RegisterFactory0(DexTypeSwapBasedPerp, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	tokens := make([]string, 0, len(entityPool.Tokens))
	for _, poolToken := range entityPool.Tokens {
		tokens = append(tokens, poolToken.Address)
	}

	reserves := make([]*big.Int, len(entityPool.Reserves))
	for i, reserve := range entityPool.Reserves {
		reserveBI, ok := new(big.Int).SetString(reserve, 10)
		if !ok {
			reserveBI = integer.Zero()
		}

		reserves[i] = reserveBI
	}

	info := pool.PoolInfo{
		Address:  entityPool.Address,
		Exchange: entityPool.Exchange,
		Type:     entityPool.Type,
		Tokens:   tokens,
		Reserves: reserves,
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: info,
		},
		vault:      extra.Vault,
		vaultUtils: NewVaultUtils(extra.Vault),
		gas:        DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	amountOutAfterFees, feeAmount, err := p.getAmountOut(tokenAmountIn.Token, tokenOut, tokenAmountIn.Amount)
	if err != nil {
		return nil, err
	}

	tokenAmountOut := &pool.TokenAmount{
		Token:  tokenOut,
		Amount: amountOutAfterFees,
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

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	vault := *p.vault
	vault.USDBAmounts = maps.Clone(p.vault.USDBAmounts)
	vault.PoolAmounts = maps.Clone(p.vault.PoolAmounts)
	cloned.vault = &vault
	return &cloned
}

// UpdateBalance update UsdbAmount only
// https://github.com/gmx-io/gmx-contracts/blob/787d767e033c411f6d083f2725fb54b7fa956f7e/contracts/core/Vault.sol#L547-L548
func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output, fee := params.TokenAmountIn, params.TokenAmountOut, params.Fee
	priceIn, err := p.vault.GetMinPrice(input.Token)
	if err != nil {
		return
	}

	usdbAmount := new(big.Int).Div(new(big.Int).Mul(input.Amount, priceIn), PricePrecision)
	usdbAmount = p.vault.AdjustForDecimals(usdbAmount, input.Token, p.vault.USDB.Address)

	p.vault.IncreaseUSDBAmount(input.Token, usdbAmount)
	p.vault.DecreaseUSDBAmount(output.Token, usdbAmount)
	p.vault.IncreasePoolAmount(input.Token, input.Amount)
	p.vault.DecreasePoolAmount(output.Token, new(big.Int).Add(output.Amount, fee.Amount))
}

func (p *PoolSimulator) CanSwapFrom(address string) []string { return p.CanSwapTo(address) }

func (p *PoolSimulator) CanSwapTo(address string) []string {
	whitelistedTokens := p.vault.WhitelistedTokens

	isTokenExists := false
	for _, token := range whitelistedTokens {
		if strings.EqualFold(token, address) {
			isTokenExists = true
		}
	}

	if !isTokenExists {
		return nil
	}

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

// getAmountOut returns amountOutAfterFees, feeAmount and error
func (p *PoolSimulator) getAmountOut(tokenIn string, tokenOut string, amountIn *big.Int) (*big.Int, *big.Int, error) {
	if !p.vault.IsSwapEnabled {
		return nil, nil, ErrVaultSwapsNotEnabled
	}

	priceIn, err := p.vault.GetMinPrice(tokenIn)
	if err != nil {
		return nil, nil, err
	}

	priceOut, err := p.vault.GetMaxPrice(tokenOut)
	if err != nil {
		return nil, nil, err
	}

	amountOut := new(big.Int).Div(new(big.Int).Mul(amountIn, priceIn), priceOut)
	amountOut = p.vault.AdjustForDecimals(amountOut, tokenIn, tokenOut)

	usdbAmount := new(big.Int).Div(new(big.Int).Mul(amountIn, priceIn), PricePrecision)
	usdbAmount = p.vault.AdjustForDecimals(usdbAmount, tokenIn, p.vault.USDB.Address)

	// in smart contract, this validation is implemented inside _increaseUsdbAmount method
	if err = p.validateMaxUsdbExceed(tokenIn, usdbAmount); err != nil {
		return nil, nil, err
	}

	// in smart contract, this validation is implemented inside _decreasePoolAmount method
	if err = p.validateMinPoolAmount(tokenOut, amountOut); err != nil {
		return nil, nil, err
	}

	// in smart contract, this validation is implemented inside _validateBufferAmount method
	if err = p.validateBufferAmount(tokenOut, amountOut); err != nil {
		return nil, nil, err
	}

	feeBasisPoints := p.vaultUtils.GetSwapFeeBasisPoints(tokenIn, tokenOut, usdbAmount)
	amountOutAfterFees := new(big.Int).Div(
		new(big.Int).Mul(
			amountOut,
			new(big.Int).Sub(BasisPointsDivisor, feeBasisPoints),
		),
		BasisPointsDivisor,
	)

	feeAmount := new(big.Int).Sub(amountOut, amountOutAfterFees)

	return amountOutAfterFees, feeAmount, nil
}

func (p *PoolSimulator) validateMaxUsdbExceed(token string, amount *big.Int) error {
	currentUsdbAmount := p.vault.USDBAmounts[token]
	newUsdbAmount := new(big.Int).Add(currentUsdbAmount, amount)

	maxUsdbAmount := p.vault.MaxUSDBAmounts[token]

	if maxUsdbAmount.Cmp(bignumber.ZeroBI) == 0 {
		return nil
	}

	if newUsdbAmount.Cmp(maxUsdbAmount) < 0 {
		return nil
	}

	return ErrVaultMaxUsdbExceeded
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

func (p *PoolSimulator) AfterMsgpackUnmarshal() error {
	if p.vaultUtils != nil {
		p.vaultUtils.vault = p.vault
	}
	return nil
}
