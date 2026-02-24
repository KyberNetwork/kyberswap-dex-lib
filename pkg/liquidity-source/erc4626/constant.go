package erc4626

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexType = "erc4626"

	erc4626MethodAsset           = "asset"
	erc4626MethodMaxDeposit      = "maxDeposit"
	erc4626MethodMaxRedeem       = "maxRedeem"
	erc4626MethodTotalAssets     = "totalAssets"
	erc4626MethodTotalSupply     = "totalSupply"
	ERC4626MethodPreviewDeposit  = "previewDeposit"
	ERC4626MethodPreviewRedeem   = "previewRedeem"
	ERC4626MethodConvertToAssets = "convertToAssets"
)

var (
	// PrefetchAmounts contains predefined amounts for prefetching ERC4626 conversion rates
	PrefetchAmounts = []*uint256.Int{
		big256.TenPow(6),
		big256.TenPow(12),
		big256.BONE,
		big256.TenPow(24),
		big256.TenPow(30),
	}

	AddrDummy = common.HexToAddress("0x1371783000000000000000000000000001371760")

	ErrInvalidToken              = errors.New("invalid token")
	ErrUnsupportedSwap           = errors.New("unsupported swap")
	ErrERC4626DepositMoreThanMax = errors.New("ERC4626: deposit more than max")
	ErrERC4626RedeemMoreThanMax  = errors.New("ERC4626: redeem more than max")

	ErrInvalidRate        = errors.New("invalid rate")
	ErrInvalidRedeemRate  = errors.New("invalid redeem rate")
	ErrInvalidDepositRate = errors.New("invalid deposit rate")
)
