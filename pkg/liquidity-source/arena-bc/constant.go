package arenabc

import (
	"errors"

	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexType = "arena-bc"

	tokenManagerMethodPaused                = "paused"
	tokenManagerMethodTokenIdentifier       = "tokenIdentifier"
	tokenManagerMethodTokenParams           = "tokenParams"
	tokenManagerMethodCanDeployLp           = "canDeployLp"
	tokenManagerMethodTokenBalanceOf        = "tokenBalanceOf"
	tokenManagerMethodGetMaxTokensForSale   = "getMaxTokensForSale"
	tokenManagerMethodProtocolFeeBasisPoint = "protocolFeeBasisPoint"
	tokenManagerMethodReferralFeeBasisPoint = "referralFeeBasisPoint"
	tokenManagerMethodTokenSupply           = "tokenSupply"
)

const (
	initialTokenId = 1

	sellGas     = 81197
	buyGas      = 96799
	createLpGas = 2475955
)

var (
	granularityScaler = u256.TenPow(18)

	U100  = uint256.NewInt(100)
	U5000 = uint256.NewInt(5000)
)

var (
	ErrLpAlreadyDeployed                = errors.New("LP already deployed")
	ErrLpDeployNotAllowedRightNow       = errors.New("LP deploy not allowed right now")
	ErrInvalidToken                     = errors.New("invalid token")
	ErrPoolPaused                       = errors.New("pool paused")
	ErrZeroSwap                         = errors.New("zero swap")
	ErrNativeBalanceOverflowOrUnderflow = errors.New("native balance: overflow or underflow")
	ErrTotalSupplyOverflowOrUnderflow   = errors.New("total supply: overflow or underflow")
	ErrUnderflow                        = errors.New("underflow")
)
