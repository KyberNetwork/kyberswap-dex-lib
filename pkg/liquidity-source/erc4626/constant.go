package erc4626

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

const (
	DexType = "erc4626"

	erc4626MethodAsset                 = "asset"
	erc4626MethodTotalSupply           = "totalSupply"
	erc4626MethodTotalAssets           = "totalAssets"
	erc4626MethodMaxDeposit            = "maxDeposit"
	erc4626MethodMaxRedeem             = "maxRedeem"
	erc4626MethodEntryFeeBasisPoints   = "entryFeeBasisPoints"
	erc4626MethodExitFeeBasisPoints    = "exitFeeBasisPoints"
	erc4626MethodGetExitFeeBasisPoints = "getExitFeeBasisPoints"
	erc4626MethodMinRedeemRatio        = "minRedeemRatio"

	Bps            = 10000
	RatioPrecision = 1e18
)

var (
	AddrDummy = common.HexToAddress("0x1371783000000000000000000000000001371760")

	ErrInvalidToken              = errors.New("invalid token")
	ErrUnsupportedSwap           = errors.New("unsupported swap")
	ErrERC4626DepositMoreThanMax = errors.New("ERC4626: deposit more than max")
	ErrERC4626RedeemMoreThanMax  = errors.New("ERC4626: redeem more than max")
)
