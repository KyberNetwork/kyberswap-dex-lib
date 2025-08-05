package bunniv2

import (
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/hooklet"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"

	"github.com/holiman/uint256"
)

const (
	MAX_ABS_TICK_MOVE = 9116
)

const (
	STATIC               uint8 = iota // LDF does not change ever
	DYNAMIC_NOT_STATEFUL              // LDF can change, does not use ldfState
	DYNAMIC_AND_STATEFUL              // LDF can change, uses ldfState
)

var (
	ZERO_BALANCE = [32]byte{
		0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	OBSERVATION_BASE_SLOT   = common.LeftPadBytes(big.NewInt(6).Bytes(), 32)
	OBSERVATION_STATE_SLOT  = common.LeftPadBytes(big.NewInt(7).Bytes(), 32)
	VAULT_SHARE_PRICES_SLOT = common.LeftPadBytes(big.NewInt(11).Bytes(), 32)
	CURATOR_FEES_SLOT       = common.LeftPadBytes(big.NewInt(15).Bytes(), 32)
	HOOK_FEE_SLOT           = common.LeftPadBytes(big.NewInt(16).Bytes(), 32)

	SWAP_FEE_BASE        = uint256.NewInt(1e6)
	MODIFIER_BASE        = SWAP_FEE_BASE
	RAW_TOKEN_RATIO_BASE = SWAP_FEE_BASE

	CURATOR_FEE_BASE      = uint256.NewInt(1e5)
	MAX_SWAP_FEE_RATIO, _ = u256.NewUint256("28800000000000000000000") // 2.88e20
	MAX_SWAP_FEE          = SWAP_FEE_BASE                              // 1e6
	MIN_FEE_AMOUNT        = u256.U1
	EPSILON_FEE           = u256.U1
	SWAP_FEE_BASE_SQUARED = uint256.NewInt(1e12)
	LN2_WAD               = uint256.NewInt(693147180559945309)
	WAD                   = uint256.NewInt(1e18)
	Q96                   = new(uint256.Int).Lsh(u256.U1, 96)

	PoolManagerAddresses = map[valueobject.ChainID]common.Address{
		valueobject.ChainIDEthereum:    common.HexToAddress("0x000000000004444c5dc75cB358380D2e3dE08A90"),
		valueobject.ChainIDBase:        common.HexToAddress("0x498581ff718922c3f8e6a244956af099b2652b2b"),
		valueobject.ChainIDArbitrumOne: common.HexToAddress("0x360e68faccca8ca495c1b759fd9eee466db9fb32"),
		valueobject.ChainIDBSC:         common.HexToAddress("0x28e2ea090877bf75740558f6bfb36a5ffee9e9df"),
		valueobject.ChainIDUnichain:    common.HexToAddress("0x1f98400000000000000000000000000000000004"),
	}

	// mapping between hook address and hub address
	HookAddresses = map[common.Address]string{
		// v1.2.1 on Ethereum, Base, Bsc, Unichain
		common.HexToAddress("0x000052423c1dB6B7ff8641b85A7eEfc7B2791888"): "0x000000000049C7bcBCa294E63567b4D21EB765f1",
		// v1.2.1 on Arbitrum
		common.HexToAddress("0x0000EB22c45bDB564F985acE0B4d05a64fa71888"): "0x000000000049C7bcBCa294E63567b4D21EB765f1",
		// v1.2.0 on Unichain (only)
		common.HexToAddress("0x005aF73a245d8171A0550ffAe2631f12cc211888"): "0x00000091Cb2d7914C9cd196161Da0943aB7b92E1",
	}

	HookletAddresses = map[string]func(string) hooklet.IHooklet{
		"0x0000e819b8A536Cf8e5d70B9C49256911033000C": hooklet.NewFeeOverrideHooklet, // v1.0.0
		"0x00eCE5a72612258f20eB24573C544f9dD8c5000C": hooklet.NewFeeOverrideHooklet, // v1.0.1
	}

	LDFAddresses = map[string]func(int) ldf.ILiquidityDensityFunction{
		// Ethereum, Base, Unichain, BSC
		"0x00000000d5248262c18C5a8c706B2a3E740B8760": ldf.NewUniformDistribution,
		"0x00000000B79037C909ff75dAFbA91b374bE2124f": ldf.NewGeometricDistribution,
		"0x000000004a3e16323618D0E43e93b4DD64151eDB": ldf.NewDoubleGeometricDistribution,
		"0x000000007cA9919151b275FABEA64A4f557Aa1F6": ldf.NewCarpetedGeometricDistribution,
		"0x000000000b757686c9596caDA54fa28f8C429E0d": ldf.NewCarpetedDoubleGeometricDistribution,
		"0x00000000a7A466ca990dE359E77B9E492d8a2d05": ldf.NewBuyTheDipGeometricDistribution,

		// Arbitrum
		"0x000000d93DF3306877eCc66c6526c6DfC163D8b4": ldf.NewUniformDistribution,
		"0x0000004f528E4547fcC40710CC3BFC6b2aaD4cE3": ldf.NewGeometricDistribution,
		"0x00000079CEE5806435ED88Fd6BfA4A465c8D2F19": ldf.NewDoubleGeometricDistribution,
		"0x0000009d24460d8F6223E39Eb5fF421E4413cA1F": ldf.NewCarpetedGeometricDistribution,
		"0x000000E22477C615223E430266AD8d5285636e30": ldf.NewCarpetedDoubleGeometricDistribution,
		"0x000000B2C6052cE049C49C3f0899992074F0462d": ldf.NewBuyTheDipGeometricDistribution,
	}
)

func GetPoolManagerAddress(chainID valueobject.ChainID) common.Address {
	poolManagerAddress, exists := PoolManagerAddresses[chainID]
	if exists {
		return poolManagerAddress
	}

	return common.Address{}
}

func GetHubAddress(hookAddress common.Address) string {
	hubAddress, exists := HookAddresses[hookAddress]
	if exists {
		return hubAddress
	}

	return ""
}

func InitLDF(address string, tickSpacing int) ldf.ILiquidityDensityFunction {
	addr := strings.ToLower(address)
	initLDF, exists := LDFAddresses[addr]
	if exists {
		return initLDF(tickSpacing)
	}

	return nil
}

func InitHooklet(address, hookletExtra string) hooklet.IHooklet {
	addr := strings.ToLower(address)
	initHooklet, exists := HookletAddresses[addr]
	if exists {
		return initHooklet(hookletExtra)
	}

	return hooklet.NewNoopHooklet("")
}
