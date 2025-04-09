package ekubo

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
)

type OracleHook struct {
	NoOpHook
}

func NewOracleHook() *OracleHook {
	return &OracleHook{}
}

func (h *OracleHook) OnBeforeSwap(_ *PoolSwapParams) (uint64, error) {
	return quoting.GasCostOfUpdatingOracleSnapshot, nil
}

func (h *OracleHook) OnAfterSwap(_ *SwapResult) (uint64, error) {
	return 0, nil
}
