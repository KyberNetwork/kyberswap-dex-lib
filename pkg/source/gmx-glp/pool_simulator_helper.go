package gmxglp

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"math/big"
)

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
