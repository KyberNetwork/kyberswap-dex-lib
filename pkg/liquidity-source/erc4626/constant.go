package erc4626

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	DexType = "erc4626"

	erc4626MethodAsset          = "asset"
	erc4626MethodMaxDeposit     = "maxDeposit"
	erc4626MethodMaxRedeem      = "maxRedeem"
	erc4626MethodPreviewDeposit = "previewDeposit"
	erc4626MethodPreviewRedeem  = "previewRedeem"
)

var (
	AddrDummy = common.HexToAddress("0x1371783000000000000000000000000001371760")

	BiWad = bignumber.BONE
	UWad  = big256.BONE

	ErrInvalidToken              = errors.New("invalid token")
	ErrUnsupportedSwap           = errors.New("unsupported swap")
	ErrERC4626DepositMoreThanMax = errors.New("ERC4626: deposit more than max")
	ErrERC4626RedeemMoreThanMax  = errors.New("ERC4626: redeem more than max")
)
