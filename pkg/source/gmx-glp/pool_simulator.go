package gmxglp

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/logger"
	"github.com/tinylib/msgp/msgp"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type Gas struct {
	Swap int64
}

type poolSimulatorInner struct {
	pool.Pool
	vault           *Vault
	vaultUtils      *VaultUtils `msg:"-"`
	glpManager      *GlpManager
	yearnTokenVault *YearnTokenVault
	gas             Gas
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
		vault:           extra.Vault,
		vaultUtils:      NewVaultUtils(extra.Vault),
		glpManager:      extra.GlpManager,
		yearnTokenVault: extra.YearnTokenVault,
		gas:             DefaultGas,
	}}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	var amountOut, feeAmount *big.Int
	var err error
	swapInfo := &gmxGlpSwapInfo{yearnTokenVaultModified: &YearnTokenVault{}}

	if strings.EqualFold(tokenOut, p.yearnTokenVault.Address) {
		amountOut, err = p.MintAndStakeGlp(swapInfo, tokenAmountIn.Token, tokenAmountIn.Amount)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
		amountOut, err = p.yearnTokenVault.Deposit(amountOut)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
		swapInfo.calcAmountOutType = calcAmountOutTypeStake
	} else if strings.EqualFold(tokenAmountIn.Token, p.yearnTokenVault.Address) {
		amountOut, err = p.yearnTokenVault.Withdraw(tokenAmountIn.Amount, swapInfo.yearnTokenVaultModified)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
		amountOut, err = p.UnstakeAndRedeemGlp(swapInfo, tokenOut, amountOut)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
		swapInfo.calcAmountOutType = calcAmountOutTypeUnStake
	} else {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("pool gmx-glp %v only allows from/to wBLT token %v", p.Info.Address, p.yearnTokenVault.Address)
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
		SwapInfo:       *swapInfo,
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

	p.yearnTokenVault.Merge(swapInfo.yearnTokenVaultModified)
}

// CanSwapFrom only allows wBLT swap to other tokens or other tokens to wBLT
func (p *PoolSimulator) CanSwapFrom(address string) []string {
	return p.CanSwapTo(address)
}

// CanSwapTo only allows wBLT swap to other tokens or other tokens to wBLT
func (p *PoolSimulator) CanSwapTo(address string) []string {
	if !strings.EqualFold(address, p.yearnTokenVault.Address) {
		return []string{p.yearnTokenVault.Address}
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

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	var directionFlag uint8 = 0

	if strings.EqualFold(tokenIn, p.yearnTokenVault.Address) {
		directionFlag = 1
	}
	return Meta{
		GlpManager:    p.glpManager.Address,
		StakeGLP:      p.glpManager.StakeGlp,
		YearnVault:    p.yearnTokenVault.Address,
		DirectionFlag: directionFlag,
	}
}

func (p *PoolSimulator) initializePointerAlias() {
	if p.vaultUtils == nil {
		p.vaultUtils = NewVaultUtils(p.vault)
	}
}

func (p *PoolSimulator) EncodeMsg(en *msgp.Writer) (err error) {
	p.initializePointerAlias()
	err = p.poolSimulatorInner.EncodeMsg(en)
	return
}

func (p *PoolSimulator) MarshalMsg(b []byte) (o []byte, err error) {
	p.initializePointerAlias()
	o, err = p.poolSimulatorInner.MarshalMsg(b)
	return
}

func (p *PoolSimulator) DecodeMsg(dc *msgp.Reader) (err error) {
	err = p.poolSimulatorInner.DecodeMsg(dc)
	if err != nil {
		return
	}
	p.initializePointerAlias()
	return
}

func (p *PoolSimulator) UnmarshalMsg(bts []byte) (o []byte, err error) {
	o, err = p.poolSimulatorInner.UnmarshalMsg(bts)
	if err != nil {
		return
	}
	p.initializePointerAlias()
	return
}
