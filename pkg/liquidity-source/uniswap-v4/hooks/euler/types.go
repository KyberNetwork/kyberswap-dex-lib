package euler

import (
	"math/big"

	eulerswap "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap"
	"github.com/ethereum/go-ethereum/common"

	"github.com/holiman/uint256"
)

type Vault = eulerswap.Vault
type Extra = eulerswap.Extra
type PoolExtra = eulerswap.PoolExtra

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
}

type SwapInfo struct {
	NewReserve0 *uint256.Int
	NewReserve1 *uint256.Int
	DebtRepaid  *uint256.Int
	ZeroForOne  bool
}

type ReserveRPC struct {
	Reserve0 *big.Int
	Reserve1 *big.Int
	Status   uint32
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
