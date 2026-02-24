package v1

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
)

type PoolExtra struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type StaticExtra struct {
	Vault0               string         `json:"v0"`
	Vault1               string         `json:"v1"`
	EulerAccount         string         `json:"ea"`
	Fee                  *uint256.Int   `json:"f"`
	ProtocolFee          *uint256.Int   `json:"pf"`
	EquilibriumReserve0  *uint256.Int   `json:"er0"`
	EquilibriumReserve1  *uint256.Int   `json:"er1"`
	PriceX               *uint256.Int   `json:"px"`
	PriceY               *uint256.Int   `json:"py"`
	ConcentrationX       *uint256.Int   `json:"cx"`
	ConcentrationY       *uint256.Int   `json:"cy"`
	ProtocolFeeRecipient common.Address `json:"pfr"`
	EVC                  string         `json:"evc"`
}

type Extra struct {
	Pause           uint32                `json:"p,omitempty"`
	Vaults          [3]*shared.VaultState `json:"v"`
	ControllerVault string                `json:"cV,omitempty"`
	Collaterals     []*uint256.Int        `json:"c,omitempty"`
}

type ParamsRPC struct {
	Data struct {
		Vault0               common.Address `abi:"vault0"`
		Vault1               common.Address `abi:"vault1"`
		EulerAccount         common.Address `abi:"eulerAccount"`
		EquilibriumReserve0  *big.Int       `abi:"equilibriumReserve0"`
		EquilibriumReserve1  *big.Int       `abi:"equilibriumReserve1"`
		PriceX               *big.Int       `abi:"priceX"`
		PriceY               *big.Int       `abi:"priceY"`
		ConcentrationX       *big.Int       `abi:"concentrationX"`
		ConcentrationY       *big.Int       `abi:"concentrationY"`
		Fee                  *big.Int       `abi:"fee"`
		ProtocolFee          *big.Int       `abi:"protocolFee"`
		ProtocolFeeRecipient common.Address `abi:"protocolFeeRecipient"`
	}
}
