package quickperps

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/tinylib/msgp/msgp"
)

type Gas struct {
	Swap int64
}

type poolSimulatorInner struct {
	pool.Pool

	vault      *Vault
	vaultUtils *VaultUtils `msg:"-"`
	gas        Gas
}

type PoolSimulator struct {
	poolSimulatorInner
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

	return &PoolSimulator{poolSimulatorInner{
		Pool: pool.Pool{
			Info: info,
		},
		vault:      extra.Vault,
		vaultUtils: NewVaultUtils(extra.Vault),
		gas:        DefaultGas,
	}}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	amountOutAfterFees, feeAmount, err := p.getAmountOut(tokenAmountIn.Token, tokenOut, tokenAmountIn.Amount)
	if err != nil {
		return &pool.CalcAmountOutResult{}, err
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

// UpdateBalance update UsdqAmount only
// https://github.com/gmx-io/gmx-contracts/blob/787d767e033c411f6d083f2725fb54b7fa956f7e/contracts/core/Vault.sol#L547-L548
func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output, fee := params.TokenAmountIn, params.TokenAmountOut, params.Fee
	priceIn, err := p.vault.GetMinPrice(input.Token)
	if err != nil {
		return
	}

	usdqAmount := new(big.Int).Div(new(big.Int).Mul(input.Amount, priceIn), PricePrecision)
	usdqAmount = p.vault.AdjustForDecimals(usdqAmount, input.Token, p.vault.USDQ.Address)

	p.vault.IncreaseUSDQAmount(input.Token, usdqAmount)
	p.vault.DecreaseUSDQAmount(output.Token, usdqAmount)
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

	usdqAmount := new(big.Int).Div(new(big.Int).Mul(amountIn, priceIn), PricePrecision)
	usdqAmount = p.vault.AdjustForDecimals(usdqAmount, tokenIn, p.vault.USDQ.Address)

	// in smart contract, this validation is implemented inside _increaseUsdqAmount method
	if err = p.validateMaxUsdqExceed(tokenIn, usdqAmount); err != nil {
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

	feeBasisPoints := p.vaultUtils.GetSwapFeeBasisPoints(tokenIn, tokenOut, usdqAmount)
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

func (p *PoolSimulator) validateMaxUsdqExceed(token string, amount *big.Int) error {
	currentUsdqAmount := p.vault.USDQAmounts[token]
	newUsdqAmount := new(big.Int).Add(currentUsdqAmount, amount)

	maxUsdqAmount := p.vault.MaxUSDQAmounts[token]

	if maxUsdqAmount.Cmp(bignumber.ZeroBI) == 0 {
		return nil
	}

	if newUsdqAmount.Cmp(maxUsdqAmount) < 0 {
		return nil
	}

	return ErrVaultMaxUsdqExceeded
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

func (p *PoolSimulator) initializeSelfReferencingPointer() {
	if p.vaultUtils == nil {
		p.vaultUtils = NewVaultUtils(p.vault)
	}
}

func (p *PoolSimulator) EncodeMsg(en *msgp.Writer) (err error) {
	p.initializeSelfReferencingPointer()
	err = p.poolSimulatorInner.EncodeMsg(en)
	return
}

func (p *PoolSimulator) MarshalMsg(b []byte) (o []byte, err error) {
	p.initializeSelfReferencingPointer()
	o, err = p.poolSimulatorInner.MarshalMsg(b)
	return
}

func (p *PoolSimulator) DecodeMsg(dc *msgp.Reader) (err error) {
	err = p.poolSimulatorInner.DecodeMsg(dc)
	if err != nil {
		return
	}
	p.initializeSelfReferencingPointer()
	return
}

func (p *PoolSimulator) UnmarshalMsg(bts []byte) (o []byte, err error) {
	o, err = p.poolSimulatorInner.UnmarshalMsg(bts)
	if err != nil {
		return
	}
	p.initializeSelfReferencingPointer()
	return
}
